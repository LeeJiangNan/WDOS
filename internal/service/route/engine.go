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

// ListRules 获取活跃路由规则
func (e *Engine) ListRules() []model.AreaRoutingRule {
	var rules []model.AreaRoutingRule
	e.db.Where("is_active = ?", true).Find(&rules)
	return rules
}

// CreateRule 创建路由规则
func (e *Engine) CreateRule(rule *model.AreaRoutingRule) error {
	rule.IsActive = true
	return e.db.Create(rule).Error
}

// matchPattern 通配符匹配 (* 匹配任意)
func matchPattern(pattern, target string) bool {
	if pattern == "*" { return true }
	if !strings.Contains(pattern, "*") {
		return pattern == target
	}
	prefix := strings.TrimSuffix(pattern, "*")
	return strings.HasPrefix(target, prefix)
}
