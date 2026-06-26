// Package route 区域路由引擎
package route

import (
	"strings"

	"github.com/LeeJiangNan/WDOS/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Engine struct {
	db    *gorm.DB
	sugar *zap.SugaredLogger
}

func NewEngine(db *gorm.DB, sugar *zap.SugaredLogger) *Engine {
	return &Engine{db: db, sugar: sugar}
}

// RouteResult 路由结果
type RouteResult struct {
	AreaName       string
	DepartmentID   uint64
	HandlerGroupID uint64
	BackupGroupID  uint64
}

// Route 根据摄像头分组路由到处理部门
func (e *Engine) Route(cameraGroups []string) *RouteResult {
	if len(cameraGroups) == 0 {
		return &RouteResult{AreaName: "未分配"}
	}

	// 加载所有活跃路由规则
	var rules []model.AreaRoutingRule
	e.db.Where("is_active = ?", true).Order("priority DESC").Find(&rules)

	for _, group := range cameraGroups {
		for _, rule := range rules {
			if matchPattern(rule.CameraGroupPattern, group) {
				e.sugar.Infof("路由匹配: camera_group=%s → area=%s, dept=%d", group, rule.AreaName, rule.DepartmentID)
				return &RouteResult{
					AreaName:       rule.AreaName,
					DepartmentID:   rule.DepartmentID,
					HandlerGroupID: rule.HandlerGroupID,
					BackupGroupID:  rule.BackupGroupID,
				}
			}
		}
	}

	return &RouteResult{AreaName: "未分配"}
}

// ListRules 获取活跃路由规则（按优先级降序）
func (e *Engine) ListRules() []model.AreaRoutingRule {
	var rules []model.AreaRoutingRule
	e.db.Where("is_active = ?", true).Order("priority DESC").Find(&rules)
	return rules
}

// CreateRule 创建路由规则
func (e *Engine) CreateRule(rule *model.AreaRoutingRule) error {
	rule.IsActive = true
	return e.db.Create(rule).Error
}

// matchPattern 通配符匹配
// "B1*"   前缀匹配 → 以 B1 开头
// "*机房" 后缀匹配 → 以 机房 结尾
// "*扶梯*" 包含匹配 → 包含 扶梯
// 无 *   → 精确匹配
func matchPattern(pattern, target string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == target
	}
	// 前后都有 * → 包含匹配
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		sub := strings.Trim(pattern, "*")
		return strings.Contains(target, sub)
	}
	// 只有前面有 * → 后缀匹配
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(target, suffix)
	}
	// 只有后面有 * → 前缀匹配
	prefix := strings.TrimSuffix(pattern, "*")
	return strings.HasPrefix(target, prefix)
}
