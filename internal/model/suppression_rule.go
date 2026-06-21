package model

import "time"

// SuppressionRule 报警抑制规则
type SuppressionRule struct {
	ID                  uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                string    `gorm:"size:100;comment:规则名称" json:"name"`
	CameraGroupFilter   string    `gorm:"type:json;comment:适用摄像头分组" json:"camera_group_filter"`
	AlgorithmIDs        string    `gorm:"type:json;comment:适用算法ID" json:"algorithm_ids"`
	SuppressEnabled     bool      `gorm:"default:true;comment:启用抑制" json:"suppress_enabled"`
	LockEnabled         bool      `gorm:"default:false;comment:启用锁定" json:"lock_enabled"`
	LockAfterSeconds    int       `gorm:"default:300;comment:锁定触发时间(秒)" json:"lock_after_seconds"`
	LockMode            string    `gorm:"type:enum('algo_only','full_camera');default:algo_only;comment:锁定模式" json:"lock_mode"`
	MaxLockSeconds      int       `gorm:"default:3600;comment:最大锁定时长(秒)" json:"max_lock_seconds"`
	UnlockOnDegreeUp    bool      `gorm:"default:true;comment:等级升级解锁" json:"unlock_on_degree_up"`
	UnlockOnNewAlgo     bool      `gorm:"default:true;comment:新算法解锁" json:"unlock_on_new_algo"`
	RecordSuppressed    bool      `gorm:"default:true;comment:记录被抑制报警" json:"record_suppressed"`
	NotifyOnLock        bool      `gorm:"default:true;comment:锁定通知" json:"notify_on_lock"`
	SummaryOnUnlock     bool      `gorm:"default:true;comment:解锁摘要" json:"summary_on_unlock"`
	IsActive            bool      `gorm:"default:true;comment:是否启用" json:"is_active"`
	CreatedAt           time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (SuppressionRule) TableName() string { return "suppression_rule" }
