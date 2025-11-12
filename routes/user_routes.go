package routes

import (
	"dinsos_kuburaya/controllers"

	"github.com/gin-gonic/gin"
)

func UserRoutes(r *gin.Engine) {
	users := r.Group("/api/users")
	{
		users.POST("/", controllers.CreateUser)
		users.GET("/", controllers.GetUsers)
		users.GET("/:id", controllers.GetUserByID)
		users.PUT("/:id", controllers.UpdateUser)
		users.DELETE("/:id", controllers.DeleteUser)
	}
}
