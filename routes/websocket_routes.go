package routes

import (
	ws "dinsos_kuburaya/websocket"

	"github.com/gin-gonic/gin"
)

func WebSocketRoutes(r *gin.RouterGroup, hub *ws.Hub) {
	r.GET("/ws/all", ws.WebSocketHandler(hub))
}
