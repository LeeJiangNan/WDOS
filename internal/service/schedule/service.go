// Package schedule 排班管理
package schedule

import (
	"fmt"
	"time"

	"github.com/LeeJiangNan/WDOS/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service 排班服务
type Service struct {
	db    *gorm.DB
	sugar *zap.SugaredLogger
}

// New 创建排班服务
func New(db *gorm.DB, sugar *zap.SugaredLogger) *Service {
	return &Service{db: db, sugar: sugar}
}

// SetScheduleReq 设置排班请求
type SetScheduleReq struct {
	Date         string `json:"date" binding:"required"`
	DepartmentID uint64 `json:"department_id"`
	Shifts       []ShiftReq `json:"shifts"`
}

// ShiftReq 班次请求
type ShiftReq struct {
	Type       string   `json:"type"`
	UserIDs    []uint64 `json:"user_ids"`
	OnCallUserID uint64 `json:"on_call_user_id"`
	Area       string   `json:"area"`
}

// SetSchedule 设置某天排班（覆盖，事务保护）
func (s *Service) SetSchedule(req *SetScheduleReq) ([]model.StaffSchedule, error) {
	var created []model.StaffSchedule
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 先删除这天的已有排班
		if err := tx.Where("shift_date = ? AND shift_type IN ?", req.Date, []string{"day", "night"}).Delete(&model.StaffSchedule{}).Error; err != nil {
			return err
		}
		for _, shift := range req.Shifts {
			for _, uid := range shift.UserIDs {
				sch := model.StaffSchedule{
					UserID:    uid,
					ShiftDate: req.Date,
					ShiftType: shift.Type,
					Area:      shift.Area,
					IsOnCall:  uid == shift.OnCallUserID,
				}
				if err := tx.Create(&sch).Error; err != nil {
					return err
				}
				created = append(created, sch)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("设置排班失败: %w", err)
	}
	s.sugar.Infof("排班已设置: %s, %d条", req.Date, len(created))
	return created, nil
}

// GetByDate 按日期查排班（支持按部门过滤）
func (s *Service) GetByDate(date string, departmentID uint64) (map[string][]model.StaffSchedule, error) {
	var schedules []model.StaffSchedule
	query := s.db.Table("staff_schedule").Where("shift_date = ?", date)
	if departmentID > 0 {
		query = query.Joins("JOIN users ON users.id = staff_schedule.user_id").
			Where("users.department_id = ?", departmentID)
	}
	query.Find(&schedules)

	result := make(map[string][]model.StaffSchedule)
	for _, sch := range schedules {
		result[sch.ShiftType] = append(result[sch.ShiftType], sch)
	}
	return result, nil
}

// BatchSet 批量排班（事务保护）
func (s *Service) BatchSet(startDate, endDate string, departmentID uint64, dayUserIDs, nightUserIDs []uint64, onCallDay, onCallNight uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除区间内已有排班
		if err := tx.Where("shift_date >= ? AND shift_date <= ?", startDate, endDate).Delete(&model.StaffSchedule{}).Error; err != nil {
			return err
		}
		count := 0
		current := startDate
		for current <= endDate {
			for _, uid := range dayUserIDs {
				if err := tx.Create(&model.StaffSchedule{
					UserID: uid, ShiftDate: current, ShiftType: "day",
					IsOnCall: uid == onCallDay,
				}).Error; err != nil {
					return err
				}
				count++
			}
			for _, uid := range nightUserIDs {
				if err := tx.Create(&model.StaffSchedule{
					UserID: uid, ShiftDate: current, ShiftType: "night",
					IsOnCall: uid == onCallNight,
				}).Error; err != nil {
					return err
				}
				count++
			}
			current = nextDay(current)
		}
		s.sugar.Infof("批量排班: %s~%s, %d条", startDate, endDate, count)
		return nil
	})
}

// GetOnCallUser 获取某日某班的值班人
func (s *Service) GetOnCallUser(date, shiftType, area string) (uint64, error) {
	var sch model.StaffSchedule
	err := s.db.Where("shift_date = ? AND shift_type = ? AND area = ? AND is_on_call = ?",
		date, shiftType, area, true).First(&sch).Error
	if err != nil {
		return 0, fmt.Errorf("未找到值班人员: %s %s %s", date, shiftType, area)
	}
	return sch.UserID, nil
}

func nextDay(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, 1).Format("2006-01-02")
}
