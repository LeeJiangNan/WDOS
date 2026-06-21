package workorder

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/LeeJiangNan/WDOS/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service 工单服务
type Service struct {
	db    *gorm.DB
	sugar *zap.SugaredLogger
}

// NewService 创建工单服务
func NewService(db *gorm.DB, sugar *zap.SugaredLogger) *Service {
	return &Service{db: db, sugar: sugar}
}

// ========== 工单生成 ==========

// CreateOrderReq 创建工单请求
type CreateOrderReq struct {
	SnowflakeID   string `json:"snowflake_id"`
	CameraName    string `json:"camera_name"`
	AlgorithmName string `json:"algorithm_name"`
	Degree        int    `json:"degree"`
	AlarmPicURL   string `json:"alarm_pic_url"`
	AlarmTime     time.Time `json:"alarm_time"`
}

// CreateOrder 创建工单
func (s *Service) CreateOrder(req *CreateOrderReq) (*model.WorkOrder, error) {
	orderNo := s.generateOrderNo()
	defaultForm := "{}"

	order := &model.WorkOrder{
		OrderNo:       orderNo,
		SnowflakeID:   req.SnowflakeID,
		Title:         fmt.Sprintf("%s - %s", req.AlgorithmName, req.CameraName),
		Status:        "pending",
		Priority:      degreeToPriority(req.Degree),
		Degree:        req.Degree,
		CameraName:    req.CameraName,
		AlgorithmName: req.AlgorithmName,
		AlarmPicURL:   req.AlarmPicURL,
		AlarmTime:     req.AlarmTime,
		DuplicateCount: 1,
		FormData:      &defaultForm,
	}

	// 设置接单 SLA 截止时间
	acceptDeadline := time.Now().Add(30 * time.Second)
	order.SlaAcceptDeadline = &acceptDeadline

	if err := s.db.Create(order).Error; err != nil {
		return nil, fmt.Errorf("创建工单失败: %w", err)
	}

	s.addLog(order.ID, 0, "系统", "", "pending", "CRIP自动生成工单")
	s.sugar.Infof("工单已创建: %s", orderNo)
	return order, nil
}

// MatchTemplate 根据算法名匹配活跃模板
func (s *Service) MatchTemplate(algorithmName string) *model.WorkOrderTemplate {
	var tpl model.WorkOrderTemplate
	err := s.db.Where("is_active = ?", true).First(&tpl).Error
	if err != nil {
		return nil
	}
	return &tpl
}

// AutoFillForm 根据 CRIP callback 数据自动填充表单
func (s *Service) AutoFillForm(tpl *model.WorkOrderTemplate, cbFields map[string]interface{}) string {
	if tpl == nil || tpl.FormSchema == "" || tpl.FormSchema == "{}" {
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
	json.Unmarshal([]byte(tpl.FormSchema), &schema)

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

// ========== 工单流转 ==========

// AcceptOrder 接单
func (s *Service) AcceptOrder(orderID, userID uint64, userName string) (*model.WorkOrder, error) {
	var order model.WorkOrder
	if err := s.db.First(&order, orderID).Error; err != nil {
		return nil, fmt.Errorf("工单不存在: id=%d", orderID)
	}
	if order.Status != "pending" {
		return nil, fmt.Errorf("工单状态不允许接单: %s", order.Status)
	}

	now := time.Now()
	order.Status = "processing"
	order.AccepterID = userID
	order.AccepterName = userName
	order.AssigneeID = userID
	order.AcceptedAt = &now

	// 设置处理 SLA 截止时间
	processDeadline := now.Add(150 * time.Second)
	order.SlaProcessDeadline = &processDeadline

	if err := s.db.Save(&order).Error; err != nil {
		return nil, fmt.Errorf("接单失败: %w", err)
	}

	s.addLog(order.ID, userID, userName, "pending", "processing", "接单")
	s.sugar.Infof("工单已接单: %s by %s", order.OrderNo, userName)
	return &order, nil
}

// SubmitOrder 提交处理
func (s *Service) SubmitOrder(orderID, userID uint64, userName, resolution string, formData, proofImages string) (*model.WorkOrder, error) {
	var order model.WorkOrder
	if err := s.db.First(&order, orderID).Error; err != nil {
		return nil, fmt.Errorf("工单不存在: id=%d", orderID)
	}
	if order.Status != "processing" {
		return nil, fmt.Errorf("工单状态不允许提交: %s", order.Status)
	}

	now := time.Now()
	order.Status = "completed"
	order.Resolution = resolution
	if formData != "" {
		order.FormData = &formData
	}
	order.CompletedAt = &now

	if err := s.db.Save(&order).Error; err != nil {
		return nil, fmt.Errorf("提交失败: %w", err)
	}

	s.addLog(order.ID, userID, userName, "processing", "completed", resolution)
	s.sugar.Infof("工单已完成: %s by %s", order.OrderNo, userName)
	return &order, nil
}

// TransferOrder 转交工单
func (s *Service) TransferOrder(orderID, toUserID uint64, toUserName, fromUser, reason string) (*model.WorkOrder, error) {
	var order model.WorkOrder
	if err := s.db.First(&order, orderID).Error; err != nil {
		return nil, fmt.Errorf("工单不存在: id=%d", orderID)
	}

	order.Status = "pending"
	order.AssigneeID = toUserID
	order.AccepterID = 0
	order.AccepterName = ""

	if err := s.db.Save(&order).Error; err != nil {
		return nil, fmt.Errorf("转交失败: %w", err)
	}

	s.addLog(order.ID, 0, fromUser, "processing", "pending", fmt.Sprintf("转交至 %s: %s", toUserName, reason))
	s.sugar.Infof("工单已转交: %s → %s", order.OrderNo, toUserName)
	return &order, nil
}

// ========== 工单查询 ==========

// ListByStatus 按状态查询工单（角色限定）
func (s *Service) ListByStatus(status, role string, userID, departmentID uint64, page, size int) ([]model.WorkOrder, int64, error) {
	var list []model.WorkOrder
	var total int64

	query := s.db.Model(&model.WorkOrder{}).Where("status = ?", status)

	// 角色数据范围限定
	switch role {
	case "handler", "":
		query = query.Where("assignee_id = ?", userID)
	case "supervisor":
		query = query.Where("department_id = ?", departmentID)
	case "manager":
		query = query.Where("department_id = ?", departmentID)
	case "admin", "director":
		// 全量
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 { page = 1 }
	if size < 1 || size > 100 { size = 20 }

	if err := query.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// GetOrder 获取工单详情
func (s *Service) GetOrder(id uint64) (*model.WorkOrder, []model.WorkOrderLog, error) {
	var order model.WorkOrder
	if err := s.db.First(&order, id).Error; err != nil {
		return nil, nil, fmt.Errorf("工单不存在: id=%d", id)
	}

	var logs []model.WorkOrderLog
	s.db.Where("order_id = ?", id).Order("id ASC").Find(&logs)

	return &order, logs, nil
}

// CheckSuppression 检查是否需要抑制
func (s *Service) CheckSuppression(cameraName, algorithmName string) (*model.WorkOrder, bool) {
	var order model.WorkOrder
	err := s.db.Where("camera_name = ? AND algorithm_name = ? AND status IN ('pending','processing')",
		cameraName, algorithmName).First(&order).Error
	if err != nil {
		return nil, false
	}
	return &order, true
}

// AddSuppressCount 追加抑制计数
func (s *Service) AddSuppressCount(orderID uint64, newPicURL string) {
	s.db.Model(&model.WorkOrder{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"duplicate_count": gorm.Expr("duplicate_count + 1"),
		"alarm_pic_url":   newPicURL,
	})
	s.addLog(orderID, 0, "系统", "", "", fmt.Sprintf("第N次重复报警已抑制"))
}

// ========== 内部方法 ==========

func (s *Service) addLog(orderID, operatorID uint64, operatorName, fromStatus, toStatus, comment string) {
	s.db.Create(&model.WorkOrderLog{
		OrderID:      orderID,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Action:       "created",
		FromStatus:   fromStatus,
		ToStatus:     toStatus,
		Comment:      comment,
	})
}

func (s *Service) generateOrderNo() string {
	return fmt.Sprintf("WD-%s-%04d", time.Now().Format("20060102"), time.Now().UnixMilli()%10000)
}

func degreeToPriority(degree int) string {
	switch {
	case degree >= 4: return "critical"
	case degree >= 3: return "high"
	case degree >= 2: return "medium"
	default: return "low"
	}
}
