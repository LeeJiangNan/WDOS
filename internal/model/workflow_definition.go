package model

import "time"

// WorkflowDefinition 工作流定义
type WorkflowDefinition struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"size:100;not null;comment:流程名称" json:"name"`
	Description  string    `gorm:"size:500;comment:流程描述" json:"description"`
	TemplateType string    `gorm:"size:30;comment:模板类型" json:"template_type"`
	States       string    `gorm:"type:json;comment:状态节点定义" json:"states"`
	Transitions  string    `gorm:"type:json;comment:状态流转规则" json:"transitions"`
	SLAConfig    string    `gorm:"type:json;comment:SLA超时配置" json:"sla_config"`
	IsActive     bool      `gorm:"default:true;comment:是否启用" json:"is_active"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (WorkflowDefinition) TableName() string { return "workflow_definition" }
