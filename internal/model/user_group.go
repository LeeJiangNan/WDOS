package model

import "time"

// UserGroup 用户组
type UserGroup struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"size:50;not null;comment:组名称" json:"name"`
	DepartmentID uint64    `gorm:"comment:所属部门" json:"department_id"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (UserGroup) TableName() string { return "user_groups" }
