package model

import "time"

// WorkOrderLog 工单操作日志
type WorkOrderLog struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID      uint64    `gorm:"index:idx_order;not null;comment:工单ID" json:"order_id"`
	Action       string    `gorm:"size:50;comment:操作类型" json:"action"`
	OperatorID   uint64    `gorm:"comment:操作人ID" json:"operator_id"`
	OperatorName string    `gorm:"size:50;comment:操作人姓名" json:"operator_name"`
	FromStatus   string    `gorm:"size:20;comment:操作前状态" json:"from_status"`
	ToStatus     string    `gorm:"size:20;comment:操作后状态" json:"to_status"`
	Comment      string    `gorm:"type:text;comment:备注" json:"comment"`
	Metadata     string    `gorm:"type:json;comment:附加元数据" json:"metadata"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (WorkOrderLog) TableName() string { return "work_order_log" }
