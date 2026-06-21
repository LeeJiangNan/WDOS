// Package alarm 报警处理服务
package alarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LeeJiangNan/WDOS/internal/model"
	"github.com/LeeJiangNan/WDOS/internal/pkg/config"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service 报警处理服务
type Service struct {
	db      *gorm.DB
	rdb     *redis.Client
	minio   *minio.Client
	bucket  string
	prefix  string // Redis key 前缀
	cripCfg config.CRIPConfig
	sugar   *zap.SugaredLogger
}

// New 创建报警服务
func New(db *gorm.DB, rdb *redis.Client, minioClient *minio.Client, bucket, redisPrefix string, cripCfg config.CRIPConfig, sugar *zap.SugaredLogger) *Service {
	return &Service{
		db:      db,
		rdb:     rdb,
		minio:   minioClient,
		bucket:  bucket,
		prefix:  redisPrefix,
		cripCfg: cripCfg,
		sugar:   sugar,
	}
}

// ProcessCallback 处理 CRIP Callback（默认来源: callback）
func (s *Service) ProcessCallback(ctx context.Context, cb *model.CRIPCallback) (*model.CallbackResponse, error) {
	return s.ProcessCallbackWithSource(ctx, cb, "callback")
}

// ProcessCallbackWithSource 处理 CRIP Callback，指定数据来源
func (s *Service) ProcessCallbackWithSource(ctx context.Context, cb *model.CRIPCallback, source string) (*model.CallbackResponse, error) {
	dedupKey := s.prefix + "alarm:" + cb.SnowflakeID

	// 1. 去重检查
	ok, err := s.rdb.SetNX(ctx, dedupKey, "1", 24*time.Hour).Result()
	if err != nil {
		return nil, fmt.Errorf("去重检查失败: %w", err)
	}
	if !ok {
		return &model.CallbackResponse{
			Action:    "ignored",
			Suppressed: true,
			Reason:    "重复 snowflake_id，已忽略",
		}, nil
	}

	// 2. 解析报警等级
	degree, _ := strconv.Atoi(cb.Degree)

	// 3. 下载报警图片到 MinIO
	alarmPicLocal := cb.AlarmPicURL
	if cb.AlarmPicURL != "" {
		uploaded, err := s.downloadImage(ctx, cb.AlarmPicURL, "alarms/raw/"+cb.SnowflakeID+".jpg")
		if err != nil {
			s.sugar.Warnf("下载报警图片失败, snowflake=%s, url=%s, err=%v", cb.SnowflakeID, cb.AlarmPicURL, err)
		} else {
			alarmPicLocal = uploaded
		}
	}

	// 4. 序列化完整 JSON
	rawJSON, _ := json.Marshal(cb)

	// 5. 序列化 camera_group
	camGroup, _ := json.Marshal(cb.CameraGroup)

	// 6. 解析时间
	alarmTime, _ := time.Parse("2006-01-02 15:04:05", cb.Timestamp)
	if alarmTime.IsZero() {
		alarmTime = time.Now()
	}

	// 7. 存入 MySQL
	raw := &model.CRIPAlarmRaw{
		SnowflakeID:    cb.SnowflakeID,
		CameraID:       cb.CameraID,
		CameraUUID:     cb.CameraUUID,
		CameraName:     cb.CameraName,
		CameraGroup:    string(camGroup),
		AlgorithmID:    cb.AlgorithmID,
		AlgorithmName:  cb.AlgorithmName,
		AlgorithmNameEn: cb.AlgorithmNameEn,
		Degree:         degree,
		AlarmPicURL:    alarmPicLocal,
		VideoURL:       cb.VideoURL,
		GPS:            cb.GPS,
		RawJSON:        string(rawJSON),
		AlarmTimestamp: alarmTime,
		Source:         source,
	}
	if err := s.db.Create(raw).Error; err != nil {
		s.sugar.Errorf("存储原始报警失败: %v", err)
		return nil, fmt.Errorf("存储原始报警失败: %w", err)
	}

	// 8. 检查抑制规则（同摄像头+同算法有未处理工单？）
	var pendingOrder model.WorkOrder
	suppressed := false
	err = s.db.Where("camera_name = ? AND algorithm_name = ? AND status IN ('pending','processing')",
		cb.CameraName, cb.AlgorithmName).First(&pendingOrder).Error
	if err == nil {
		// 存在未处理工单 → 抑制
		pendingOrder.DuplicateCount++
		pendingOrder.AlarmPicURL = alarmPicLocal // 更新最新截图
		s.db.Save(&pendingOrder)
		suppressed = true
		s.sugar.Infof("报警抑制: snowflake=%s, 追加到工单 %d, 累计 %d 次",
			cb.SnowflakeID, pendingOrder.ID, pendingOrder.DuplicateCount)
	}

	if suppressed {
		return &model.CallbackResponse{
			Action:      "suppressed",
			WorkOrderID: pendingOrder.ID,
			Suppressed:  true,
			Reason:      fmt.Sprintf("同摄像头同算法存在未处理工单，已追加为第 %d 次重复报警", pendingOrder.DuplicateCount),
		}, nil
	}

	// 9. 生成工单（后续阶段会完善：路由 + 通知）
	defaultFormData := "{}"
	order := &model.WorkOrder{
		OrderNo:        s.generateOrderNo(),
		SnowflakeID:    cb.SnowflakeID,
		Title:          fmt.Sprintf("%s - %s", cb.AlgorithmName, cb.CameraName),
		Status:         "pending",
		Priority:       degreeToPriority(degree),
		Degree:         degree,
		FormData:       &defaultFormData,
		CameraName:     cb.CameraName,
		AlgorithmName:  cb.AlgorithmName,
		AlarmPicURL:    alarmPicLocal,
		AlarmTime:      alarmTime,
		DuplicateCount: 1,
	}
	if err := s.db.Create(order).Error; err != nil {
		s.sugar.Errorf("生成工单失败: %v", err)
		return nil, fmt.Errorf("生成工单失败: %w", err)
	}

	// 10. 生成操作日志
	s.db.Create(&model.WorkOrderLog{
		OrderID:      order.ID,
		Action:       "created",
		OperatorName: "系统",
		ToStatus:     "pending",
		Comment:      "CRIP自动生成工单",
	})

	// 11. 关联 raw alarm 到工单
	s.db.Model(raw).Update("suppressed_by_order_id", order.ID)

	// 12. 心跳计数器
	s.rdb.Incr(ctx, s.prefix+"callback:count")

	s.sugar.Infof("工单已生成: snowflake=%s, order=%s", cb.SnowflakeID, order.OrderNo)
	return &model.CallbackResponse{
		Action:      "created",
		WorkOrderID: order.ID,
		Suppressed:  false,
		Reason:      "成功生成工单",
	}, nil
}

// downloadImage 下载图片并上传到 MinIO
func (s *Service) downloadImage(ctx context.Context, url, objectName string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	_, err = s.minio.PutObject(ctx, s.bucket, objectName,
		bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/minio/%s/%s", s.bucket, objectName), nil
}

// drawBoundingBoxes 在图片上绘制检测框（预留，阶段 4 实现）
func drawBoundingBoxes(imgData []byte, objects []model.CRIPDetectObject) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	for _, obj := range objects {
		boxColor := color.RGBA{255, 0, 0, 255}
		if obj.Score < 0.8 {
			boxColor = color.RGBA{255, 150, 50, 255}
		}
		if obj.Score < 0.6 {
			boxColor = color.RGBA{0, 200, 80, 255}
		}

		// 画四条边（粗 2px）
		for t := 0; t < 2; t++ {
			for x := obj.Rect.X; x < obj.Rect.X+obj.Rect.Width; x++ {
				if y := obj.Rect.Y + t; y < bounds.Max.Y {
					rgba.Set(x, y, boxColor)
				}
				if y := obj.Rect.Y + obj.Rect.Height - t - 1; y >= 0 {
					rgba.Set(x, y, boxColor)
				}
			}
			for y := obj.Rect.Y; y < obj.Rect.Y+obj.Rect.Height; y++ {
				if x := obj.Rect.X + t; x < bounds.Max.X {
					rgba.Set(x, y, boxColor)
				}
				if x := obj.Rect.X + obj.Rect.Width - t - 1; x >= 0 {
					rgba.Set(x, y, boxColor)
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// generateOrderNo 生成工单编号
func (s *Service) generateOrderNo() string {
	now := time.Now()
	return fmt.Sprintf("WD-%s-%04d", now.Format("20060102"), now.UnixMilli()%10000)
}

// degreeToPriority 报警等级转优先级
func degreeToPriority(degree int) string {
	switch {
	case degree >= 4:
		return "critical"
	case degree >= 3:
		return "high"
	case degree >= 2:
		return "medium"
	default:
		return "low"
	}
}

// SuppressionCheck 检查是否需要抑制
func (s *Service) SuppressionCheck(cameraName, algorithmName string) (*model.WorkOrder, bool) {
	var order model.WorkOrder
	err := s.db.Where("camera_name = ? AND algorithm_name = ? AND status IN ('pending','processing')",
		cameraName, algorithmName).First(&order).Error
	if err != nil {
		return nil, false
	}
	return &order, true
}

// HeartbeatCount 获取心跳计数值
func (s *Service) HeartbeatCount(ctx context.Context) (int64, error) {
	val, err := s.rdb.Get(ctx, s.prefix+"callback:count").Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// ResetHeartbeat 重置心跳计数器
func (s *Service) ResetHeartbeat(ctx context.Context) error {
	return s.rdb.Del(ctx, s.prefix+"callback:count").Err()
}

// underscored 转换驼峰为下划线（简单版）
func underscored(name string) string {
	var buf strings.Builder
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				buf.WriteByte('_')
			}
			buf.WriteRune(r + 32)
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
