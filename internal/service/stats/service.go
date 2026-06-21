// Package stats 统计服务
package stats

import (
	"time"

	"github.com/LeeJiangNan/WDOS/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	db    *gorm.DB
	sugar *zap.SugaredLogger
}

func New(db *gorm.DB, sugar *zap.SugaredLogger) *Service {
	return &Service{db: db, sugar: sugar}
}

// DailyOverview 每日报警概览
func (s *Service) DailyOverview(date string) map[string]interface{} {
	var totalAlarms, totalOrders, completedOrders, overtimeOrders int64
	s.db.Model(&model.CRIPAlarmRaw{}).Where("DATE(alarm_timestamp) = ?", date).Count(&totalAlarms)
	s.db.Model(&model.WorkOrder{}).Where("DATE(created_at) = ?", date).Count(&totalOrders)
	s.db.Model(&model.WorkOrder{}).Where("DATE(completed_at) = ? AND status = ?", date, "completed").Count(&completedOrders)
	s.db.Model(&model.WorkOrder{}).Where("DATE(created_at) = ? AND escalated_level > 0", date).Count(&overtimeOrders)

	completionRate := 0.0
	if totalOrders > 0 { completionRate = float64(completedOrders) / float64(totalOrders) }

	return map[string]interface{}{
		"date":              date,
		"total_alarms":      totalAlarms,
		"total_orders":      totalOrders,
		"completed_orders":  completedOrders,
		"completion_rate":   completionRate,
		"overtime_orders":   overtimeOrders,
	}
}

// ByAlgorithm 每类算法报警数
func (s *Service) ByAlgorithm(date string) []map[string]interface{} {
	type row struct {
		AlgorithmName string
		Count         int64
	}
	var rows []row
	s.db.Model(&model.CRIPAlarmRaw{}).Select("algorithm_name, count(*) as count").
		Where("DATE(alarm_timestamp) = ?", date).Group("algorithm_name").Order("count DESC").Find(&rows)

	total := int64(0)
	for _, r := range rows { total += r.Count }

	var result []map[string]interface{}
	for _, r := range rows {
		ratio := 0.0
		if total > 0 { ratio = float64(r.Count) / float64(total) }
		result = append(result, map[string]interface{}{
			"algorithm_name": r.AlgorithmName, "count": r.Count, "ratio": ratio,
		})
	}
	return result
}

// ByArea 各大区域统计
func (s *Service) ByArea(date string) []map[string]interface{} {
	var result []map[string]interface{}
	// 按 camera_name 前缀分组（简化版，完整版需关联 area_routing_rule 表）
	areas := []string{"B1停车场", "B2停车场", "1楼", "2楼", "5楼", "外围"}
	for _, area := range areas {
		var count int64
		s.db.Model(&model.CRIPAlarmRaw{}).Where("DATE(alarm_timestamp) = ? AND camera_name LIKE ?", date, area+"%").Count(&count)
		result = append(result, map[string]interface{}{"area_name": area, "alarm_count": count})
	}
	return result
}

// ProcessTimeDist 处理耗时分布
func (s *Service) ProcessTimeDist(date string) []map[string]interface{} {
	buckets := []struct{ Label string; Min, Max int }{
		{"0-30秒", 0, 30}, {"30-60秒", 30, 60}, {"60-120秒", 60, 120}, {"120-300秒", 120, 300}, {">300秒", 300, 999999},
	}
	total := int64(0)
	var result []map[string]interface{}
	for _, b := range buckets {
		var count int64
		s.db.Model(&model.WorkOrder{}).Where("DATE(completed_at) = ? AND status = 'completed' AND TIMESTAMPDIFF(SECOND, accepted_at, completed_at) BETWEEN ? AND ?", date, b.Min, b.Max).Count(&count)
		total += count
		ratio := 0.0
		if total > 0 { ratio = float64(count) / float64(total) }
		result = append(result, map[string]interface{}{"label": b.Label, "count": count, "ratio": ratio})
	}
	return result
}

// Trend 近N天趋势
func (s *Service) Trend(days int) []map[string]interface{} {
	var result []map[string]interface{}
	for i := days - 1; i >= 0; i-- {
		dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		var count int64
		s.db.Model(&model.CRIPAlarmRaw{}).Where("DATE(alarm_timestamp) = ?", dateStr).Count(&count)
		result = append(result, map[string]interface{}{"date": dateStr, "alarm_count": count})
	}
	return result
}

// UserRanking 人员绩效排行
func (s *Service) UserRanking(date string) []map[string]interface{} {
	type row struct {
		UserName string
		Count    int64
	}
	var rows []row
	s.db.Model(&model.WorkOrder{}).Select("accepter_name as user_name, count(*) as count").
		Where("DATE(completed_at) = ? AND status = 'completed' AND accepter_name != ''", date).
		Group("accepter_name").Order("count DESC").Limit(20).Find(&rows)

	var result []map[string]interface{}
	for i, r := range rows {
		result = append(result, map[string]interface{}{"rank": i + 1, "user_name": r.UserName, "completed_count": r.Count})
	}
	return result
}
