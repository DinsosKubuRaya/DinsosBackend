package websocket

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // sesuaikan jika perlu
	},
}

func WebSocketHandler(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Query("user_id")

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		client := &Client{
			UserID: userID,
			Conn:   conn,
		}

		hub.register <- client
		go func() {
			defer func() {
				hub.unregister <- client
				client.Conn.Close()
			}()

			for {
				// Baca message tapi tidak dipakai
				_, _, err := client.Conn.ReadMessage()
				if err != nil {
					log.Println("WS read error / closed:", err)
					break
				}
			}
		}()

	}
}
