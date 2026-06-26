// Package alarm 报警处理服务
package alarm

import (
	"bytes"
	"context"
	"encoding/base64"
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
	"github.com/LeeJiangNan/WDOS/internal/service/notify"
	"github.com/LeeJiangNan/WDOS/internal/service/route"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service 报警处理服务
type Service struct {
	db          *gorm.DB
	rdb         *redis.Client
	cosClient   *cos.Client      // COS 客户端（图片直存）
	cosCfg      config.COSConfig // COS 配置
	minio       *minio.Client    // MinIO 客户端（COS不可用时的兜底）
	bucket      string           // MinIO bucket
	prefix      string           // Redis key 前缀
	cripCfg     config.CRIPConfig
	sugar       *zap.SugaredLogger
	routeEngine *route.Engine
	notifyHub   *notify.Hub
}

// New 创建报警服务
func New(db *gorm.DB, rdb *redis.Client, cosClient *cos.Client, cosCfg config.COSConfig, minioClient *minio.Client, bucket, redisPrefix string, cripCfg config.CRIPConfig, sugar *zap.SugaredLogger, routeEngine *route.Engine, notifyHub *notify.Hub) *Service {
	return &Service{
		db:          db,
		rdb:         rdb,
		cosClient:   cosClient,
		cosCfg:      cosCfg,
		minio:       minioClient,
		bucket:      bucket,
		prefix:      redisPrefix,
		cripCfg:     cripCfg,
		sugar:       sugar,
		routeEngine: routeEngine,
		notifyHub:   notifyHub,
	}
}

// ProcessCallback 处理 CRIP Callback（默认来源: callback）
func (s *Service) ProcessCallback(ctx context.Context, cb *model.CRIPCallback) (*model.CallbackResponse, error) {
	return s.ProcessCallbackWithSource(ctx, cb, "callback")
}

// ProcessCallbackWithSource 处理 CRIP Callback，指定数据来源
func (s *Service) ProcessCallbackWithSource(ctx context.Context, cb *model.CRIPCallback, source string) (*model.CallbackResponse, error) {
	dedupKey := s.prefix + "alarm:" + string(cb.SnowflakeID)

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
	degree, err := strconv.Atoi(string(cb.Degree))
	if err != nil {
		s.sugar.Warnf("解析报警等级失败: snowflake=%s, degree=%s, err=%v", string(cb.SnowflakeID), string(cb.Degree), err)
		degree = 0
	}

	// 3. 获取报警图片（URL → base64 兜底）+ 画检测框 → COS
	alarmPicLocal := cb.AlarmPicURL
	objectName := "alarms/raw/" + string(cb.SnowflakeID) + ".jpg"
	var imgData []byte

	// 先尝试从 URL 下载
	if cb.AlarmPicURL != "" && !strings.Contains(cb.AlarmPicURL, "example.com") {
		imgData, err = s.downloadImageData(ctx, cb.AlarmPicURL)
		if err != nil {
			s.sugar.Warnf("下载报警图片失败, snowflake=%s, url=%s, err=%v", string(cb.SnowflakeID), cb.AlarmPicURL, err)
		}
	}

	// URL 下载失败 → 从 base64 解码
	if len(imgData) == 0 && cb.AlarmPicData != "" {
		imgData, err = base64.StdEncoding.DecodeString(cb.AlarmPicData)
		if err != nil {
			s.sugar.Warnf("base64 解码失败: %v", err)
		}
	}

	// 画框已由 CRIP 端完成，直接上传图片到 COS
	if len(imgData) > 0 {
		if s.cosClient != nil {
			_, uploadErr := s.cosClient.Object.Put(ctx, objectName, bytes.NewReader(imgData),
				&cos.ObjectPutOptions{ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: "image/jpeg"}})
			if uploadErr != nil {
				s.sugar.Warnf("上传图片到 COS 失败: %v", uploadErr)
			} else {
				alarmPicLocal = s.cosCfg.PublicURL + "/" + objectName
			}
		} else if s.minio != nil {
			_, uploadErr := s.minio.PutObject(ctx, s.bucket, objectName,
				bytes.NewReader(imgData), int64(len(imgData)),
				minio.PutObjectOptions{ContentType: "image/jpeg"})
			if uploadErr != nil {
				s.sugar.Warnf("上传图片到 MinIO 失败: %v", uploadErr)
			} else {
				alarmPicLocal = "/minio/" + s.bucket + "/" + objectName
			}
		}
	}

	// 4. 序列化完整 JSON
	rawJSON, err := json.Marshal(cb)
	if err != nil {
		s.sugar.Warnf("序列化 callback JSON 失败: snowflake=%s, err=%v", string(cb.SnowflakeID), err)
		rawJSON = []byte("{}")
	}

	// 5. 序列化 camera_group
	camGroup, err := json.Marshal(cb.CameraGroup)
	if err != nil {
		s.sugar.Warnf("序列化 camera_group 失败: snowflake=%s, err=%v", string(cb.SnowflakeID), err)
		camGroup = []byte("[]")
	}

	// 6. 解析时间
	alarmTime, err := time.Parse("2006-01-02 15:04:05", cb.Timestamp)
	if err != nil {
		s.sugar.Warnf("解析报警时间失败: snowflake=%s, timestamp=%s, err=%v, 使用当前时间", string(cb.SnowflakeID), cb.Timestamp, err)
	}
	if alarmTime.IsZero() {
		alarmTime = time.Now()
	}

	// 7. 存入 MySQL
	raw := &model.CRIPAlarmRaw{
		SnowflakeID:    string(cb.SnowflakeID),
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
		AlarmTimestamp: model.LocalTime{T: alarmTime},
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
		// 存在未处理工单 → 原子更新抑制计数（避免并发竞争）
		s.db.Model(&pendingOrder).Updates(map[string]interface{}{
			"duplicate_count": gorm.Expr("duplicate_count + 1"),
			"alarm_pic_url":   alarmPicLocal,
		})
		suppressed = true
		// 标记 raw alarm 为已抑制
		s.db.Model(raw).Updates(map[string]interface{}{
			"suppressed":             true,
			"suppressed_by_order_id": pendingOrder.ID,
		})
		s.sugar.Infof("报警抑制: snowflake=%s, 追加到工单 %d",
			string(cb.SnowflakeID), pendingOrder.ID)
	}

	if suppressed {
		return &model.CallbackResponse{
			Action:      "suppressed",
			WorkOrderID: pendingOrder.ID,
			Suppressed:  true,
			Reason:      fmt.Sprintf("同摄像头同算法存在未处理工单，已追加为第 %d 次重复报警", pendingOrder.DuplicateCount+1),
		}, nil
	}

	// 9. 生成工单
	// 9.0 匹配活跃模板并填充表单数据
	var tpl model.WorkOrderTemplate
	formData := "{}"
	templateID := uint64(0)
	if err := s.db.Where("is_active = ?", true).First(&tpl).Error; err == nil {
		templateID = tpl.ID
		// 用 CRIP callback 数据填充表单
		formData = s.fillFormFromCallback(&tpl, cb)
	}

	order := &model.WorkOrder{
		OrderNo:        s.generateOrderNo(),
		SnowflakeID:    string(cb.SnowflakeID),
		Title:          fmt.Sprintf("%s - %s", cb.AlgorithmName, cb.CameraName),
		Status:         "pending",
		Priority:       degreeToPriority(degree),
		Degree:         degree,
		TemplateID:     templateID,
		FormData:       &formData,
		CameraName:     cb.CameraName,
		AlgorithmName:  cb.AlgorithmName,
		AlarmPicURL:    alarmPicLocal,
		AlarmTime: model.LocalTime{T: alarmTime},
		DuplicateCount: 1,
	}

	// 9.1 路由：根据 camera_group 分配部门和班组
	if s.routeEngine != nil {
		result := s.routeEngine.Route(cb.CameraGroup)
		if result != nil {
			order.DepartmentID = result.DepartmentID
			order.AssigneeID = result.HandlerGroupID
				// 查询部门名称
				var dept model.Department
				if s.db.Where("id = ?", result.DepartmentID).First(&dept).Error == nil {
					order.DepartmentName = dept.Name
				}
				// 查询用户组名称作为指派人
				var ug model.UserGroup
				if s.db.Where("id = ?", result.HandlerGroupID).First(&ug).Error == nil {
					order.AssigneeName = ug.Name
				}
		}
	}

	if err := s.db.Create(order).Error; err != nil {
		s.sugar.Errorf("生成工单失败: %v", err)
		return nil, fmt.Errorf("生成工单失败: %w", err)
	}

	// 10. 生成操作日志
	if err := s.db.Create(&model.WorkOrderLog{
		OrderID:      order.ID,
		Action:       "created",
		OperatorName: "系统",
		ToStatus:     "pending",
		Comment:      "CRIP自动生成工单",
	}).Error; err != nil {
		s.sugar.Warnf("创建操作日志失败: order=%d, err=%v", order.ID, err)
	}

	// 11. 关联 raw alarm 到工单
	if err := s.db.Model(raw).Update("suppressed_by_order_id", order.ID).Error; err != nil {
		s.sugar.Warnf("关联原始报警失败: raw=%d, order=%d, err=%v", raw.ID, order.ID, err)
	}

	// 12. 心跳计数器
	s.rdb.Incr(ctx, s.prefix+"callback:count")

	// 13. 发送 WebSocket 通知
	if s.notifyHub != nil {
		s.notifyHub.NewOrder(notify.NewOrderPayload{
			OrderID: order.ID,
			Title:   order.Title,
		})
	}

	s.sugar.Infof("工单已生成: snowflake=%s, order=%s", string(cb.SnowflakeID), order.OrderNo)
	return &model.CallbackResponse{
		Action:      "created",
		WorkOrderID: order.ID,
		Suppressed:  false,
		Reason:      "成功生成工单",
	}, nil
}

// downloadImageData 下载图片返回原始数据
func (s *Service) downloadImageData(ctx context.Context, imageURL string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载图片失败: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// downloadImage 下载图片并上传到 MinIO
func (s *Service) downloadImage(ctx context.Context, url, objectName string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载图片失败: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	if s.cosClient != nil {
		_, err = s.cosClient.Object.Put(ctx, objectName, bytes.NewReader(data),
			&cos.ObjectPutOptions{ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{ContentType: contentType}})
		if err != nil {
			return "", err
		}
		return s.cosCfg.PublicURL + "/" + objectName, nil
	}
	if s.minio != nil {
		_, err = s.minio.PutObject(ctx, s.bucket, objectName,
			bytes.NewReader(data), int64(len(data)),
			minio.PutObjectOptions{ContentType: contentType})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("/minio/%s/%s", s.bucket, objectName), nil
	}
	return "", fmt.Errorf("无可用的图片存储")
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
	return fmt.Sprintf("WD-%s-%06d", now.Format("20060102"), now.UnixMilli()%1000000)
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
// fillFormFromCallback 根据模板的字段映射，从 CRIP callback 数据填充表单
func (s *Service) fillFormFromCallback(tpl *model.WorkOrderTemplate, cb *model.CRIPCallback) string {
	if tpl.FormSchema == "" || tpl.FormSchema == "{}" {
		return "{}"
	}

	var schema struct {
		Components []struct {
			Type    string `json:"type"`
			Label   string `json:"label"`
			FieldID string `json:"field_id"`
			Mapping string `json:"mapping"`
		} `json:"components"`
	}
	if err := json.Unmarshal([]byte(tpl.FormSchema), &schema); err != nil {
		s.sugar.Warnf("解析模板表单 schema 失败: tpl=%d, err=%v", tpl.ID, err)
		return "{}"
	}

	// 将 callback 数据平铺为可映射字段
	cbFields := map[string]interface{}{
		"camera_id":        cb.CameraID,
		"camera_uuid":      cb.CameraUUID,
		"camera_name":      cb.CameraName,
		"algorithm_id":     cb.AlgorithmID,
		"algorithm_name":   cb.AlgorithmName,
		"algorithm_name_en": cb.AlgorithmNameEn,
		"degree":           string(cb.Degree),
		"gps":              cb.GPS,
		"timestamp":        cb.Timestamp,
	}

	formData := make(map[string]interface{})
	for _, comp := range schema.Components {
		if comp.Mapping != "" {
			if val, ok := cbFields[comp.Mapping]; ok {
				formData[comp.FieldID] = val
			}
		}
	}

	result, _ := json.Marshal(formData)
	return string(result)
}
