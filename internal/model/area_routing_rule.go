package model

import "time"

// AreaRoutingRule 区域路由规则
type AreaRoutingRule struct {
	ID                  uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CameraGroupPattern  string    `gorm:"size:100;comment:摄像头分组匹配(支持通配符)" json:"camera_group_pattern"`
	AreaName            string    `gorm:"size:50;comment:区域名称" json:"area_name"`
	DepartmentID        uint64    `gorm:"comment:负责部门" json:"department_id"`
	HandlerGroupID      uint64    `gorm:"comment:默认处理人组" json:"handler_group_id"`
	BackupGroupID       uint64    `gorm:"comment:备用处理人组" json:"backup_group_id"`
	Priority            int       `gorm:"default:0;comment:优先级" json:"priority"`
	IsActive            bool      `gorm:"default:true;comment:是否启用" json:"is_active"`
	CreatedAt           time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AreaRoutingRule) TableName() string { return "area_routing_rule" }
