package model

import "time"

// User 用户表
type User struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string    `gorm:"size:50;not null;comment:姓名" json:"name"`
	Phone        string    `gorm:"uniqueIndex;size:20;not null;comment:手机号" json:"phone"`
	Password     string    `gorm:"size:255;comment:密码(bcrypt)" json:"-"`
	Role         string    `gorm:"size:20;default:handler;comment:角色" json:"role"`
	DepartmentID uint64    `gorm:"comment:部门ID" json:"department_id"`
	GroupID      uint64    `gorm:"comment:用户组ID" json:"group_id"`
	OpenID       string    `gorm:"size:64;comment:微信openid" json:"open_id"`
	Status       string    `gorm:"type:enum('active','disabled');default:active;comment:状态" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string { return "users" }
