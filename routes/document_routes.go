package routes

import (
	"dinsos_kuburaya/controllers"
	"dinsos_kuburaya/middleware"

	"github.com/gin-gonic/gin"
)

func DocumentRoutes(r *gin.RouterGroup) {
	documents := r.Group("/documents")

	// Harus login + hanya admin dan superadmin yang boleh akses
	documents.Use(
		middleware.AuthMiddleware(),
		middleware.RoleMiddleware("admin", "superadmin"),
	)

	{
		// ADMIN & SUPERADMIN BOLEH READ
		documents.GET("", controllers.GetDocuments)
		documents.GET("/", controllers.GetDocuments)
		documents.GET("/:id", controllers.GetDocumentByID)
		documents.GET("/:id/download", controllers.DownloadDocument)

		// ADMIN & SUPERADMIN BOLEH CREATE, UPDATE, DELETE
		documents.POST("", controllers.CreateDocument)
		documents.POST("/", controllers.CreateDocument)
		documents.PUT("/:id", controllers.UpdateDocument)
		documents.DELETE("/:id", controllers.DeleteDocument)
	}
}
