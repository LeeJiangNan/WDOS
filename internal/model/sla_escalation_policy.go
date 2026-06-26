package model


// SlaEscalationPolicy SLA 上报策略
type SlaEscalationPolicy struct {
	ID                  uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                string    `gorm:"size:100;comment:策略名称" json:"name"`
	TemplateIDs         string    `gorm:"type:json;comment:适用工单模板" json:"template_ids"`
	AcceptL1Seconds     int       `gorm:"comment:接单一级超时(秒)" json:"accept_l1_seconds"`
	AcceptL1GroupID     uint64    `gorm:"comment:一级上报用户组" json:"accept_l1_group_id"`
	AcceptL2Seconds     int       `gorm:"comment:接单二级超时(秒)" json:"accept_l2_seconds"`
	AcceptL2GroupID     uint64    `gorm:"comment:二级上报用户组" json:"accept_l2_group_id"`
	AcceptL3Seconds     int       `gorm:"comment:接单三级超时(秒)" json:"accept_l3_seconds"`
	AcceptL3GroupID     uint64    `gorm:"comment:三级上报用户组" json:"accept_l3_group_id"`
	ProcessL1Seconds    int       `gorm:"comment:处理一级超时(秒)" json:"process_l1_seconds"`
	ProcessL1GroupID    uint64    `gorm:"comment:处理一级上报用户组" json:"process_l1_group_id"`
	ProcessL2Seconds    int       `gorm:"comment:处理二级超时(秒)" json:"process_l2_seconds"`
	ProcessL2GroupID    uint64    `gorm:"comment:处理二级上报用户组" json:"process_l2_group_id"`
	ProcessL3Seconds    int       `gorm:"comment:处理三级超时(秒)" json:"process_l3_seconds"`
	ProcessL3GroupID    uint64    `gorm:"comment:处理三级上报用户组" json:"process_l3_group_id"`
	NotifyChannels      string    `gorm:"type:json;comment:通知方式" json:"notify_channels"`
	IsActive            bool      `gorm:"default:true;comment:是否启用" json:"is_active"`
	CreatedAt           LocalTime `gorm:"autoCreateTime" json:"created_at"`
}

func (SlaEscalationPolicy) TableName() string { return "sla_escalation_policy" }
