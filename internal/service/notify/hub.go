// Package notify 通知推送服务
package notify

import (
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Message 通知消息
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	CreatedAt time.Time   `json:"created_at"`
}

// NewOrderPayload 新工单通知
type NewOrderPayload struct {
	OrderID       uint64 `json:"order_id"`
	OrderNo       string `json:"order_no"`
	Title         string `json:"title"`
	CameraName    string `json:"camera_name"`
	AlgorithmName string `json:"algorithm_name"`
	Degree        int    `json:"degree"`
}

// EscalationPayload 上报通知
type EscalationPayload struct {
	OrderID     uint64 `json:"order_id"`
	OrderNo     string `json:"order_no"`
	Stage       string `json:"stage"`
	Level       int    `json:"level"`
	LevelName   string `json:"level_name"`
	OverSeconds int    `json:"over_seconds"`
}

// clientInfo 连接元信息
type clientInfo struct {
	userID uint64
	role   string
}

// Hub WebSocket 连接管理中心
type Hub struct {
	clients map[*websocket.Conn]clientInfo // conn → {userID, role}
	mu      sync.RWMutex
	db      *gorm.DB
	sugar   *zap.SugaredLogger
}

// NewHub 创建通知 Hub
func NewHub(db *gorm.DB, sugar *zap.SugaredLogger) *Hub {
	h := &Hub{
		clients: make(map[*websocket.Conn]clientInfo),
		db:      db,
		sugar:   sugar,
	}
	go h.heartbeatLoop()
	return h
}

// heartbeatLoop 定期 ping 所有连接，清理僵尸连接
func (h *Hub) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.RLock()
		dead := []*websocket.Conn{}
		for conn := range h.clients {
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				dead = append(dead, conn)
			}
		}
		h.mu.RUnlock()
		for _, c := range dead {
			h.Unregister(c)
		}
	}
}

// Register 注册 WebSocket 连接
func (h *Hub) Register(conn *websocket.Conn, userID uint64, role string) {
	h.mu.Lock()
	h.clients[conn] = clientInfo{userID: userID, role: role}
	h.mu.Unlock()
	h.sugar.Infof("WebSocket 连接: user=%d, role=%s, 当前连接数=%d", userID, role, len(h.clients))
}

// Unregister 注销连接
func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
	conn.Close()
}

// Broadcast 广播给所有连接
func (h *Hub) Broadcast(msg Message) {
	data, _ := json.Marshal(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			c := conn
			go h.Unregister(c) // 捕获当前 conn，避免闭包变量覆盖
		}
	}
}

// BroadcastToRole 广播给指定角色的连接
func (h *Hub) BroadcastToRole(msg Message, role string) {
	data, _ := json.Marshal(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn, info := range h.clients {
		if info.role == role {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				c := conn
				go h.Unregister(c)
			}
		}
	}
}

// NewOrder 新工单通知
func (h *Hub) NewOrder(payload NewOrderPayload) {
	h.Broadcast(Message{Type: "new_order", Data: payload, CreatedAt: time.Now()})
	h.saveHistory(0, "new_order", "新工单", payload.Title, payload.OrderID)
}

// Escalation 超时上报通知
func (h *Hub) Escalation(payload EscalationPayload) {
	level := strconv.Itoa(payload.Level)
	// L1→supervisor, L2→manager, L3→admin/director
	targetRole := map[int]string{1: "supervisor", 2: "manager", 3: "director"}[payload.Level]
	msg := Message{Type: "escalation_l" + level, Data: payload, CreatedAt: time.Now()}
	if targetRole != "" {
		h.BroadcastToRole(msg, targetRole)
		// 同时广播给 admin
		h.BroadcastToRole(msg, "admin")
	} else {
		h.Broadcast(msg)
	}
	h.saveHistory(0, "escalation_l"+level,
		payload.LevelName+"超时提醒",
		payload.OrderNo+" "+payload.Stage+"超时L"+level,
		payload.OrderID)
}

// OrderAccepted 接单通知
func (h *Hub) OrderAccepted(orderID uint64, orderNo, accepter string) {
	h.Broadcast(Message{Type: "order_accepted", Data: map[string]interface{}{
		"order_id": orderID, "order_no": orderNo, "accepter": accepter,
	}, CreatedAt: time.Now()})
}

// OrderCompleted 完成通知
func (h *Hub) OrderCompleted(orderID uint64, orderNo string) {
	h.Broadcast(Message{Type: "order_completed", Data: map[string]interface{}{
		"order_id": orderID, "order_no": orderNo,
	}, CreatedAt: time.Now()})
}

// saveHistory 保存通知历史
func (h *Hub) saveHistory(userID uint64, msgType, title, message string, orderID uint64) {
	// 写 MySQL 通知表（如果后续创建的话），目前先记日志
	h.sugar.Infof("📬 通知: [%s] %s - %s (order=%d)", msgType, title, message, orderID)
}
