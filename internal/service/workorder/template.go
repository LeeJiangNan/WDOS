package workorder

import (
	"encoding/json"
	"fmt"

	"github.com/LeeJiangNan/WDOS/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TemplateService 工单模板服务
type TemplateService struct {
	db    *gorm.DB
	sugar *zap.SugaredLogger
}

// NewTemplateService 创建模板服务
func NewTemplateService(db *gorm.DB, sugar *zap.SugaredLogger) *TemplateService {
	return &TemplateService{db: db, sugar: sugar}
}

// Create 创建模板
func (s *TemplateService) Create(name, description string, formSchema json.RawMessage, flowID uint64) (*model.WorkOrderTemplate, error) {
	if name == "" {
		return nil, fmt.Errorf("模板名称不能为空")
	}

	// 检查重名
	var exist model.WorkOrderTemplate
	if err := s.db.Where("name = ?", name).First(&exist).Error; err == nil {
		return nil, fmt.Errorf("模板名称已存在: %s", name)
	}

	schema := string(formSchema)
	if schema == "" || schema == "null" {
		schema = "{}"
	}

	tpl := &model.WorkOrderTemplate{
		Name:        name,
		Description: description,
		FlowID:      flowID,
		FormSchema:  schema,
		IsActive:    true,
	}
	if err := s.db.Create(tpl).Error; err != nil {
		return nil, fmt.Errorf("创建模板失败: %w", err)
	}

	s.sugar.Infof("工单模板已创建: id=%d, name=%s", tpl.ID, name)
	return tpl, nil
}

// Update 更新模板
func (s *TemplateService) Update(id uint64, name, description string, formSchema json.RawMessage, flowID uint64) (*model.WorkOrderTemplate, error) {
	var tpl model.WorkOrderTemplate
	if err := s.db.First(&tpl, id).Error; err != nil {
		return nil, fmt.Errorf("模板不存在: id=%d", id)
	}

	if name != "" && name != tpl.Name {
		var exist model.WorkOrderTemplate
		if err := s.db.Where("name = ? AND id != ?", name, id).First(&exist).Error; err == nil {
			return nil, fmt.Errorf("模板名称已被占用: %s", name)
		}
		tpl.Name = name
	}
	if description != "" {
		tpl.Description = description
	}
	if flowID > 0 {
		tpl.FlowID = flowID
	}
	schema := string(formSchema)
	if schema != "" && schema != "null" {
		tpl.FormSchema = schema
	}

	if err := s.db.Save(&tpl).Error; err != nil {
		return nil, fmt.Errorf("更新模板失败: %w", err)
	}

	s.sugar.Infof("工单模板已更新: id=%d", tpl.ID)
	return &tpl, nil
}

// List 模板列表
func (s *TemplateService) List(status string, page, size int) ([]model.WorkOrderTemplate, int64, error) {
	var list []model.WorkOrderTemplate
	var total int64

	query := s.db.Model(&model.WorkOrderTemplate{})
	if status == "active" {
		query = query.Where("is_active = ?", true)
	} else if status == "inactive" {
		query = query.Where("is_active = ?", false)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	if err := query.Order("id DESC").Offset(offset).Limit(size).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// Get 获取单个模板
func (s *TemplateService) Get(id uint64) (*model.WorkOrderTemplate, error) {
	var tpl model.WorkOrderTemplate
	if err := s.db.First(&tpl, id).Error; err != nil {
		return nil, fmt.Errorf("模板不存在: id=%d", id)
	}
	return &tpl, nil
}

// Toggle 切换启用/停用
func (s *TemplateService) Toggle(id uint64, active bool) (*model.WorkOrderTemplate, error) {
	var tpl model.WorkOrderTemplate
	if err := s.db.First(&tpl, id).Error; err != nil {
		return nil, fmt.Errorf("模板不存在: id=%d", id)
	}

	tpl.IsActive = active
	if err := s.db.Save(&tpl).Error; err != nil {
		return nil, fmt.Errorf("切换状态失败: %w", err)
	}

	status := "启用"
	if !active {
		status = "停用"
	}
	s.sugar.Infof("工单模板已%s: id=%d, name=%s", status, tpl.ID, tpl.Name)
	return &tpl, nil
}
