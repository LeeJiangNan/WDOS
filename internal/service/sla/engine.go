// Package sla SLA 超时引擎
package sla

import (
	"context"
	"strconv"
	"time"

	"github.com/LeeJiangNan/WDOS/internal/model"
	"github.com/LeeJiangNan/WDOS/internal/service/notify"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Engine SLA 引擎
type Engine struct {
	db    *gorm.DB
	sugar *zap.SugaredLogger
	// 接单超时阈值
	acceptL1, acceptL2, acceptL3 int
	// 处理超时阈值
	processL1, processL2, processL3 int
	notifyHub *notify.Hub
}

// New 创建 SLA 引擎
func New(db *gorm.DB, acceptL1, acceptL2, acceptL3, processL1, processL2, processL3 int, sugar *zap.SugaredLogger, notifyHub *notify.Hub) *Engine {
	return &Engine{
		db:        db,
		sugar:     sugar,
		acceptL1:  acceptL1,
		acceptL2:  acceptL2,
		acceptL3:  acceptL3,
		processL1: processL1,
		processL2: processL2,
		processL3: processL3,
		notifyHub: notifyHub,
	}
}

// EscalationEvent 上报事件
type EscalationEvent struct {
	OrderID   uint64 `json:"order_id"`
	OrderNo   string `json:"order_no"`
	Stage     string `json:"stage"`     // accept / process
	Level     int    `json:"level"`     // 1=班长, 2=经理, 3=总监
	Threshold int    `json:"threshold"` // 超时阈值(秒)
	OverSeconds int `json:"over_seconds"` // 已超时秒数
}

// Run 启动 SLA 扫描（阻塞，在 goroutine 中调用）
func (e *Engine) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	e.sugar.Infof("SLA 引擎已启动, 间隔: %v", interval)

	for {
		select {
		case <-ctx.Done():
			e.sugar.Info("SLA 引擎已停止")
			return
		case <-ticker.C:
			e.scan()
		}
	}
}

// scan 扫描超时工单
func (e *Engine) scan() {
	now := time.Now()

	// 1. 扫描待接单超时（不限时间窗口，所有未处理工单均参与扫描）
	var pendingOrders []model.WorkOrder
	e.db.Where("status = ?", "pending").Find(&pendingOrders)
	for _, o := range pendingOrders {
		// 转交的工单以 transferred_at 为新计时起点，否则用创建时间
		startTime := o.CreatedAt.Time()
		if o.TransferredAt != nil {
			startTime = o.TransferredAt.Time()
		}
		seconds := int(now.Sub(startTime).Seconds())
		e.checkAccept(o, seconds)
	}

	// 2. 扫描处理中超时（不限时间窗口）
	var processingOrders []model.WorkOrder
	e.db.Where("status = ?", "processing").Find(&processingOrders)
	for _, o := range processingOrders {
		if o.AcceptedAt != nil {
			seconds := int(now.Sub(o.AcceptedAt.Time()).Seconds())
			e.checkProcess(o, seconds)
		}
	}
}

func (e *Engine) checkAccept(order model.WorkOrder, elapsed int) {
	prevLevel := order.EscalatedLevel

	// 逐级触发，避免 switch 跳级（如 elapsed 同时满足 L1/L2/L3 时只触发 L3）
	if elapsed >= e.acceptL1 && prevLevel < 1 {
		e.escalate(order, "accept", 1, e.acceptL1, elapsed)
	}
	if elapsed >= e.acceptL2 && prevLevel < 2 {
		e.escalate(order, "accept", 2, e.acceptL2, elapsed)
	}
	if elapsed >= e.acceptL3 && prevLevel < 3 {
		e.escalate(order, "accept", 3, e.acceptL3, elapsed)
	}
}

func (e *Engine) checkProcess(order model.WorkOrder, elapsed int) {
	prevLevel := order.EscalatedLevel

	// 逐级触发，避免 switch 跳级
	if elapsed >= e.processL1 && prevLevel < 1 {
		e.escalate(order, "process", 1, e.processL1, elapsed)
	}
	if elapsed >= e.processL2 && prevLevel < 2 {
		e.escalate(order, "process", 2, e.processL2, elapsed)
	}
	if elapsed >= e.processL3 && prevLevel < 3 {
		e.escalate(order, "process", 3, e.processL3, elapsed)
	}
}

func (e *Engine) escalate(order model.WorkOrder, stage string, level, threshold, elapsed int) {
	levelNames := map[int]string{1: "班长", 2: "经理", 3: "总监"}
	stageNames := map[string]string{"accept": "接单", "process": "处理"}

	e.sugar.Warnf("🔔 SLA上报: %s %s超时, L%d-%s, 阈值%ds, 已超%d秒",
		order.OrderNo, stageNames[stage], level, levelNames[level], threshold, elapsed)

	// 更新上报层级
	e.db.Model(&order).Update("escalated_level", level)

	// L2 触发锁定
	if stage == "accept" && level >= 2 {
		e.db.Model(&order).Updates(map[string]interface{}{
			"is_locked": true,
			"locked_at": time.Now(),
			"lock_mode": "algo_only",
		})
		e.sugar.Warnf("🔒 点位锁定: %s, camera=%s, algo=%s", order.OrderNo, order.CameraName, order.AlgorithmName)
	}

	// 写操作日志
	e.db.Create(&model.WorkOrderLog{
		OrderID:      order.ID,
		Action:       "escalated",
		OperatorName: "系统",
		Comment:      buildEscalationComment(stage, level, levelNames[level], threshold, elapsed),
	})

	// 推送 WebSocket 通知
	if e.notifyHub != nil {
		e.notifyHub.Escalation(notify.EscalationPayload{
			OrderID:     order.ID,
			OrderNo:     order.OrderNo,
			Stage:       stage,
			Level:       level,
			LevelName:   levelNames[level],
			OverSeconds: elapsed - threshold,
		})
	}
}

func buildEscalationComment(stage string, level int, name string, threshold, elapsed int) string {
	stageName := "接单"
	if stage == "process" {
		stageName = "处理"
	}
	return stageName + "超时" + strconv.Itoa(level) + "级上报: " + name + ", 阈值" + strconv.Itoa(threshold) + "秒, 已超" + strconv.Itoa(elapsed) + "秒"
}
