package routes

import (
	"dinsos_kuburaya/controllers"
	"dinsos_kuburaya/middleware"
	"dinsos_kuburaya/services"

	"github.com/gin-gonic/gin"
)

func NotificationRoutes(router *gin.RouterGroup) {
	notifications := router.Group("/notifications")
	notifications.Use(middleware.AuthMiddleware())
	{
		// Handle tanpa trailing slash
		notifications.GET("", controllers.GetNotifications)

		// Handle dengan trailing slash (fallback)
		notifications.GET("/", controllers.GetNotifications)

		notifications.POST("/:id/read", controllers.MarkNotificationAsRead)

		notifications.POST("/read-all", controllers.MarkAllAsRead)
		notifications.POST("/test-firebase", func(c *gin.Context) {
			err := services.TestFirebaseConnection()
			if err != nil {
				c.JSON(500, gin.H{"error": "Firebase connection failed", "details": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Firebase connection successful"})
		})
	}
}
