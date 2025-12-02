package routes

import (
	"dinsos_kuburaya/controllers"
	"dinsos_kuburaya/middleware"

	"github.com/gin-gonic/gin"
)

func SuperiorOrderRoutes(router *gin.RouterGroup) {
	superior := router.Group("/superior_orders")
	superior.Use(middleware.AuthMiddleware())

	{
		superior.POST("", controllers.CreateSuperiorOrder)
		superior.GET("", controllers.GetSuperiorOrders)
		superior.POST("/", controllers.CreateSuperiorOrder)
		superior.GET("/", controllers.GetSuperiorOrders)

		// Gunakan parameter yang konsisten - semua menggunakan :id
		superior.GET("/:id", controllers.GetSuperiorOrdersByDocument)
		superior.PUT("/:id", controllers.UpdateSuperiorOrder)
		superior.DELETE("/:id", controllers.DeleteSuperiorOrder)
	}
}
