package model

import "time"

// WorkOrder 工单主表
type WorkOrder struct {
	ID                  uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderNo             string     `gorm:"uniqueIndex;size:32;not null;comment:工单编号" json:"order_no"`
	TemplateID          uint64     `gorm:"not null;comment:工单模板ID" json:"template_id"`
	SnowflakeID         string     `gorm:"size:64;comment:关联原始报警" json:"snowflake_id"`
	Title               string     `gorm:"size:200;comment:工单标题" json:"title"`
	Status              string     `gorm:"type:enum('pending','processing','completed','transferred','closed');default:pending;index:idx_status;comment:状态" json:"status"`
	Priority            string     `gorm:"type:enum('low','medium','high','critical');default:medium;comment:优先级" json:"priority"`
	Degree              int        `gorm:"comment:报警等级" json:"degree"`
	DepartmentID        uint64     `gorm:"index:idx_department;comment:负责部门" json:"department_id"`
	DepartmentName      string     `gorm:"size:50" json:"department_name"`
	AssigneeID          uint64     `gorm:"index:idx_assignee;comment:当前处理人" json:"assignee_id"`
	AccepterID          uint64     `gorm:"comment:接单人" json:"accepter_id"`
	AccepterName        string     `gorm:"size:50" json:"accepter_name"`
	DuplicateCount      int        `gorm:"default:1;comment:抑制累计次数" json:"duplicate_count"`
	SuppressedAlarmCount int       `gorm:"default:0;comment:锁定期被抑制的报警数" json:"suppressed_alarm_count"`
	IsLocked            bool       `gorm:"default:false;index:idx_locked;comment:是否处于锁定状态" json:"is_locked"`
	LockedAt            *time.Time `gorm:"comment:锁定时间" json:"locked_at"`
	LockMode            string     `gorm:"type:enum('none','algo_only','full_camera');default:none;comment:锁定模式" json:"lock_mode"`
	EscalatedLevel      int        `gorm:"default:0;comment:上报层级" json:"escalated_level"`
	SlaAcceptDeadline   *time.Time `gorm:"comment:接单SLA截止" json:"sla_accept_deadline"`
	SlaProcessDeadline  *time.Time `gorm:"comment:处理SLA截止" json:"sla_process_deadline"`
	FormData            *string    `gorm:"type:json;comment:表单数据" json:"form_data"`
	Resolution          string     `gorm:"type:text;comment:处理结果" json:"resolution"`
	CameraName          string     `gorm:"size:100" json:"camera_name"`
	AlgorithmName       string     `gorm:"size:50" json:"algorithm_name"`
	AlarmPicURL         string     `gorm:"size:500" json:"alarm_pic_url"`
	AlarmTime           time.Time  `json:"alarm_time"`
	CreatedAt           time.Time  `gorm:"autoCreateTime;index:idx_created" json:"created_at"`
	AcceptedAt          *time.Time `json:"accepted_at"`
	CompletedAt         *time.Time `json:"completed_at"`
	ClosedAt            *time.Time `json:"closed_at"`
}

func (WorkOrder) TableName() string { return "work_order" }
