package routes

import (
	"dinsos_kuburaya/controllers"
	"dinsos_kuburaya/middleware"

	"github.com/gin-gonic/gin"
)

func ActivityLogRoutes(router *gin.RouterGroup) {
	logs := router.Group("/activity-logs")
	logs.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("admin", "superadmin"),
	)
	{
		logs.GET("", controllers.GetAllActivityLogs)
	}
}
