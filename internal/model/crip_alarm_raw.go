package model

import "time"

// CRIPAlarmRaw CRIP 原始报警记录
type CRIPAlarmRaw struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SnowflakeID    string    `gorm:"uniqueIndex;size:64;not null;comment:CRIP雪花ID，去重主键" json:"snowflake_id"`
	CameraID       int       `gorm:"index:idx_camera_algo;comment:摄像头ID" json:"camera_id"`
	CameraUUID     string    `gorm:"size:64;comment:摄像头UUID" json:"camera_uuid"`
	CameraName     string    `gorm:"size:100;comment:摄像头名称" json:"camera_name"`
	CameraGroup    string    `gorm:"type:json;comment:摄像头分组" json:"camera_group"`
	AlgorithmID    int       `gorm:"index:idx_camera_algo;comment:算法ID" json:"algorithm_id"`
	AlgorithmName  string    `gorm:"size:50;comment:算法中文名" json:"algorithm_name"`
	AlgorithmNameEn string   `gorm:"size:100;comment:算法英文名" json:"algorithm_name_en"`
	Degree         int       `gorm:"comment:报警等级" json:"degree"`
	AlarmPicURL    string    `gorm:"size:500;comment:报警图片URL" json:"alarm_pic_url"`
	VideoURL       string    `gorm:"size:500;comment:视频URL" json:"video_url"`
	GPS            string    `gorm:"size:50;comment:GPS坐标" json:"gps"`
	RawJSON        string    `gorm:"type:json;comment:完整Callback JSON" json:"raw_json"`
	AlarmTimestamp time.Time `gorm:"index:idx_alarm_time;comment:报警时间" json:"alarm_timestamp"`
	ReceivedAt     time.Time `gorm:"autoCreateTime;comment:接收时间" json:"received_at"`
	Source         string    `gorm:"type:enum('callback','compensation');default:callback;comment:数据来源" json:"source"`
	Suppressed     bool      `gorm:"default:false;comment:是否被抑制" json:"suppressed"`
	SuppressedByID *uint64   `gorm:"comment:被哪个工单抑制" json:"suppressed_by_order_id"`
}

func (CRIPAlarmRaw) TableName() string { return "crip_alarm_raw" }
