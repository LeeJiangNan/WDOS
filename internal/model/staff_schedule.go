package model


// StaffSchedule 人员排班
type StaffSchedule struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"index:idx_user_date;comment:人员ID" json:"user_id"`
	ShiftDate string    `gorm:"index:idx_date_shift;size:10;comment:排班日期 YYYY-MM-DD" json:"shift_date"`
	ShiftType string    `gorm:"type:enum('day','night');index:idx_date_shift;comment:班次" json:"shift_type"`
	Area      string    `gorm:"size:50;comment:负责区域" json:"area"`
	IsOnCall  bool      `gorm:"default:false;comment:是否值班" json:"is_on_call"`
	CreatedAt LocalTime `gorm:"autoCreateTime" json:"created_at"`
}

func (StaffSchedule) TableName() string { return "staff_schedule" }
