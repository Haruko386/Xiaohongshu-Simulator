package socket

import (
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// Upgrader 负责将普通的 HTTP 请求升级为 WebSocket 协议
var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ClientManager struct {
	Clients      map[uint]*websocket.Conn // 记录 UserID -> 对应的网络连接
	sync.RWMutex                          // 读写锁，防止并发操作 map 导致程序崩溃
}

// GlobalManager 全局唯一的连接池实例
var GlobalManager = &ClientManager{
	Clients: make(map[uint]*websocket.Conn),
}

// SendMessage 是暴露给外部的推送方法，精确定向发给某个 UserID
func (m *ClientManager) SendMessage(userID uint, message interface{}) {
	m.RLock() // 加读锁
	conn, ok := m.Clients[userID]
	m.RUnlock()

	if ok {
		// 如果这个用户在线，就直接顺着网线把 JSON 发过去
		_ = conn.WriteJSON(message)
	}
}
