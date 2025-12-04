package routes

import (
	"dinsos_kuburaya/controllers"
	"dinsos_kuburaya/middleware"

	"github.com/gin-gonic/gin"
)

func UserRoutes(router *gin.RouterGroup) {
	users := router.Group("/users")

	// users.POST("/superadmin/init", controllers.CreateSuperAdmin)
	users.POST("/push-token", middleware.AuthMiddleware(), controllers.StorePushToken)

	// ============
	// CREATE USERS
	// ============
	// Superadmin membuat superadmin, admin, staff
	users.POST("/superadmin", middleware.AuthMiddleware(), middleware.RoleMiddleware("superadmin"), controllers.CreateSuperAdmin)

	// superadmin boleh buat admin
	users.POST("/admin", middleware.AuthMiddleware(), middleware.RoleMiddleware("superadmin"), controllers.CreateAdmin)

	// superadmin boleh buat staff
	users.POST("/staff", middleware.AuthMiddleware(), middleware.RoleMiddleware("superadmin"), controllers.CreateStaff)

	// ============
	// PROTECTED ROUTES
	// ============
	usersAuth := users.Group("")
	usersAuth.Use(middleware.AuthMiddleware())
	{
		// STAFF: hanya bisa GET /me
		usersAuth.GET("/me", controllers.GetMe)

		// SUPERADMIN & ADMIN → Get All Users
		usersAuth.GET("", middleware.RoleMiddleware("admin", "superadmin"), controllers.GetUsers)

		// SUPERADMIN & ADMIN → Get User by ID
		usersAuth.GET("/:id", middleware.RoleMiddleware("admin", "superadmin"), controllers.GetUserByID)

		// UPDATE:
		usersAuth.PUT("/:id", middleware.UserSelfOrSuperAdmin(), controllers.UpdateUser)

		// DELETE: hanya superadmin
		usersAuth.DELETE("/:id", middleware.RoleMiddleware("superadmin"), controllers.DeleteUser)

		usersAuth.GET("/for-filter", middleware.RoleMiddleware("admin", "superadmin"), controllers.GetUsersForFilter)
	}
}
