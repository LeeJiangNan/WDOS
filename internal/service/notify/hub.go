// Package notify 通知推送服务
package notify

import (
	"encoding/json"
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

// Hub WebSocket 连接管理中心
type Hub struct {
	clients    map[*websocket.Conn]uint64 // conn → userID
	mu         sync.RWMutex
	db         *gorm.DB
	sugar      *zap.SugaredLogger
}

// NewHub 创建通知 Hub
func NewHub(db *gorm.DB, sugar *zap.SugaredLogger) *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]uint64),
		db:      db,
		sugar:   sugar,
	}
}

// Register 注册 WebSocket 连接
func (h *Hub) Register(conn *websocket.Conn, userID uint64) {
	h.mu.Lock()
	h.clients[conn] = userID
	h.mu.Unlock()
	h.sugar.Infof("WebSocket 连接: user=%d, 当前连接数=%d", userID, len(h.clients))
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
			go h.Unregister(conn)
		}
	}
}

// BroadcastToRole 广播给指定角色的连接
func (h *Hub) BroadcastToRole(msg Message, role string) {
	data, _ := json.Marshal(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 简化版：广播给所有连接（实际应按 user 的 role 过滤）
	for conn := range h.clients {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

// NewOrder 新工单通知
func (h *Hub) NewOrder(payload NewOrderPayload) {
	h.Broadcast(Message{Type: "new_order", Data: payload, CreatedAt: time.Now()})
	h.saveHistory(0, "new_order", "新工单", payload.Title, payload.OrderID)
}

// Escalation 超时上报通知
func (h *Hub) Escalation(payload EscalationPayload) {
	h.Broadcast(Message{Type: "escalation_l" + itoa(payload.Level), Data: payload, CreatedAt: time.Now()})
	h.saveHistory(0, "escalation_l"+itoa(payload.Level),
		payload.LevelName+"超时提醒",
		payload.OrderNo+" "+payload.Stage+"超时L"+itoa(payload.Level),
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

func itoa(n int) string {
	if n == 0 { return "0" }
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
