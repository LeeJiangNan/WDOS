package model


// WorkOrderTemplate 工单模板
type WorkOrderTemplate struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:100;not null;comment:模板名称" json:"name"`
	Description string    `gorm:"size:500;comment:模板描述" json:"description"`
	FlowID      uint64    `gorm:"comment:关联工作流" json:"flow_id"`
	FormSchema  string    `gorm:"type:json;comment:表单定义" json:"form_schema"`
	IsActive    bool      `gorm:"default:true;comment:是否启用" json:"is_active"`
	CreatedAt   LocalTime `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   LocalTime `gorm:"autoUpdateTime" json:"updated_at"`
}

func (WorkOrderTemplate) TableName() string { return "work_order_template" }
