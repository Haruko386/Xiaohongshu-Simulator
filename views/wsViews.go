package views

import (
	"Xiaohongshu_Simulator/socket"
	"github.com/gin-gonic/gin"
	"log"
)

func ConnectWS(c *gin.Context) {
	userID := c.MustGet("user_id").(uint)

	conn, err := socket.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket 升级失败:", err)
		return
	}

	// 用户上线！登记到调度中心的账本上
	socket.GlobalManager.Lock()
	socket.GlobalManager.Clients[userID] = conn
	socket.GlobalManager.Unlock()
	log.Printf("用户上线: UserID = %d, 当前在线人数: %d\n", userID, len(socket.GlobalManager.Clients))

	defer func() {
		socket.GlobalManager.Lock()
		delete(socket.GlobalManager.Clients, userID)
		socket.GlobalManager.Unlock()
		_ = conn.Close()
		log.Printf("用户下线: UserID = %d\n", userID)
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break // 如果前端关掉网页或断网，ReadMessage 会报错，跳出循环，触发上面的 defer 登记下线
		}
	}
}
