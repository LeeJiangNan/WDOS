package model

// AlgorithmRoutingRule 算法工单配置 — 按算法名称分配工单到部门
type AlgorithmRoutingRule struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	AlgorithmPattern string    `gorm:"size:100;not null;comment:算法匹配模式" json:"algorithm_pattern"`
	DisplayName      string    `gorm:"size:100;comment:显示名称" json:"display_name"`
	DepartmentID     uint64    `gorm:"not null;index;comment:分配部门ID" json:"department_id"`
	Category         string    `gorm:"size:50;comment:分类(消防/安防/其他)" json:"category"`
	Priority         int       `gorm:"default:10;comment:优先级，越大越优先" json:"priority"`
	IsActive         bool      `gorm:"default:true;comment:是否启用" json:"is_active"`
	CreatedAt        LocalTime `gorm:"autoCreateTime" json:"created_at"`
}

func (AlgorithmRoutingRule) TableName() string {
	return "algorithm_routing_rule"
}
