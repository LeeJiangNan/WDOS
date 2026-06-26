// Package stats 统计服务
package stats

import (
	"strconv"
	"strings"
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
	var totalAlarms, totalOrders, completedOrders, overtimeOrders, processingOrders, pendingOrders int64
	s.db.Model(&model.CRIPAlarmRaw{}).Where("DATE(alarm_timestamp) = ?", date).Count(&totalAlarms)
	s.db.Model(&model.WorkOrder{}).Where("DATE(created_at) = ?", date).Count(&totalOrders)
	s.db.Model(&model.WorkOrder{}).Where("DATE(completed_at) = ? AND status = ?", date, "completed").Count(&completedOrders)
	s.db.Model(&model.WorkOrder{}).Where("DATE(created_at) = ? AND escalated_level > 0", date).Count(&overtimeOrders)
	s.db.Model(&model.WorkOrder{}).Where("DATE(created_at) = ? AND status = ?", date, "processing").Count(&processingOrders)
	s.db.Model(&model.WorkOrder{}).Where("DATE(created_at) = ? AND status = ?", date, "pending").Count(&pendingOrders)

	completionRate := 0.0
	if totalOrders > 0 { completionRate = float64(completedOrders) / float64(totalOrders) }

	return map[string]interface{}{
		"date":              date,
		"total_alarms":      totalAlarms,
		"total_orders":      totalOrders,
		"completed_orders":  completedOrders,
		"completion_rate":   completionRate,
		"overtime_orders":   overtimeOrders,
		"processing_orders": processingOrders,
		"pending_orders":    pendingOrders,
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

// ByArea 各大区域统计（从路由规则表动态获取区域列表）
func (s *Service) ByArea(date string) []map[string]interface{} {
	// 从 area_routing_rule 表获取所有活跃区域
	var rules []model.AreaRoutingRule
	s.db.Where("is_active = ?", true).Find(&rules)

	var result []map[string]interface{}
	for _, rule := range rules {
		var count int64
		// 按摄像头分组模糊匹配（通配符用 SQL % 替换 *）
		pattern := strings.ReplaceAll(rule.CameraGroupPattern, "*", "%")
		s.db.Model(&model.CRIPAlarmRaw{}).
			Where("DATE(alarm_timestamp) = ? AND camera_group LIKE ?", date, "%"+pattern+"%").
			Count(&count)
		result = append(result, map[string]interface{}{
			"area_name": rule.AreaName, "alarm_count": count,
		})
	}

	// 如果没有路由规则，回退到硬编码兜底
	if len(result) == 0 {
		areas := []string{"B1停车场", "B2停车场", "1楼", "2楼", "5楼", "外围"}
		for _, area := range areas {
			var count int64
			s.db.Model(&model.CRIPAlarmRaw{}).Where("DATE(alarm_timestamp) = ? AND camera_name LIKE ?", date, area+"%").Count(&count)
			result = append(result, map[string]interface{}{"area_name": area, "alarm_count": count})
		}
	}
	return result
}

// ProcessTimeDist 处理耗时分布
func (s *Service) ProcessTimeDist(date string) []map[string]interface{} {
	buckets := []struct{ Label string; Min, Max int }{
		{"0-30秒", 0, 30}, {"31-60秒", 31, 60}, {"61-120秒", 61, 120}, {"121-300秒", 121, 300}, {">300秒", 301, 999999},
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

// DailyOverviewFiltered 过去24h工单概览
// pending/processing/overtime 统计全量，completed 统计过去24h
func (s *Service) DailyOverviewFiltered(date string, role string, userID, departmentID uint64, departmentIDs string) map[string]interface{} {
	since := time.Now().Add(-24 * time.Hour)
	return s.overviewWithRules("过去24小时", role, userID, departmentID, departmentIDs, since)
}

// WeeklyOverviewFiltered 过去7天工单概览
// pending/processing/overtime 统计全量，completed 统计过去7天
func (s *Service) WeeklyOverviewFiltered(role string, userID, departmentID uint64, departmentIDs string) map[string]interface{} {
	since := time.Now().Add(-7 * 24 * time.Hour)
	return s.overviewWithRules("过去7天", role, userID, departmentID, departmentIDs, since)
}

// MonthlyOverviewFiltered 过去30天工单概览
// pending/processing/overtime 统计全量，completed 统计过去30天
func (s *Service) MonthlyOverviewFiltered(role string, userID, departmentID uint64, departmentIDs string) map[string]interface{} {
	since := time.Now().Add(-30 * 24 * time.Hour)
	return s.overviewWithRules("过去30天", role, userID, departmentID, departmentIDs, since)
}

// overviewWithRules 统一统计逻辑：
// - pending/processing: 全量（不漏掉历史遗留工单）
// - completed: 仅统计 since 之后的已完工单
// - overtime: 全量
func (s *Service) overviewWithRules(label, role string, userID, departmentID uint64, departmentIDs string, since time.Time) map[string]interface{} {
	// 基础过滤（角色权限）
	roleFilter := func(q *gorm.DB) *gorm.DB {
		switch role {
		case "handler", "":
			q = q.Where("assignee_id = ?", userID)
		case "supervisor", "manager":
			q = q.Where("department_id IN ?", splitDeptIDs(departmentIDs, departmentID))
		case "admin", "director":
			// 全量
		}
		return q
	}

	var completedOrders int64
	var overtimeOrders int64
	var processingOrders int64
	var pendingOrders int64

	base := s.db.Model(&model.WorkOrder{})
	roleFilter(base).Where("status = 'pending'").Count(&pendingOrders)
	roleFilter(s.db.Model(&model.WorkOrder{})).Where("status = 'processing'").Count(&processingOrders)
	roleFilter(s.db.Model(&model.WorkOrder{})).Where("status = 'completed' AND completed_at >= ?", since).Count(&completedOrders)
	roleFilter(s.db.Model(&model.WorkOrder{})).Where("escalated_level > 0").Count(&overtimeOrders)

	totalOrders := pendingOrders + processingOrders + completedOrders
	rate := 0.0
	if totalOrders > 0 { rate = float64(completedOrders) / float64(totalOrders) }
	return map[string]interface{}{
		"label": label,
		"total_orders": totalOrders, "completed_orders": completedOrders,
		"completion_rate": rate, "overtime_orders": overtimeOrders,
		"processing_orders": processingOrders, "pending_orders": pendingOrders,
	}
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

// splitDeptIDs 解析逗号分隔的部门ID字符串为切片
func splitDeptIDs(deptIDsStr string, defaultID uint64) []uint64 {
	if deptIDsStr == "" {
		return []uint64{defaultID}
	}
	parts := strings.Split(deptIDsStr, ",")
	ids := make([]uint64, 0, len(parts))
	for _, p := range parts {
		if id, err := strconv.ParseUint(strings.TrimSpace(p), 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return []uint64{defaultID}
	}
	return ids
}

