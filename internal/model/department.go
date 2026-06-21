package model

import "time"

// Department 部门
type Department struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:50;not null;comment:部门名称" json:"name"`
	ManagerID uint64    `gorm:"comment:负责人ID" json:"manager_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (Department) TableName() string { return "departments" }
