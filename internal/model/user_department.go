package model

// UserDepartment 用户-部门关联（多对多）
type UserDepartment struct {
	UserID       uint64 `gorm:"primaryKey;comment:用户ID" json:"user_id"`
	DepartmentID uint64 `gorm:"primaryKey;index:idx_dept;comment:部门ID" json:"department_id"`
	IsPrimary    bool   `gorm:"default:false;comment:是否主部门" json:"is_primary"`
}

func (UserDepartment) TableName() string { return "user_department" }
