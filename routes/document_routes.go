package routes

import (
	"dinsos_kuburaya/controllers"
	"dinsos_kuburaya/middleware"

	"github.com/gin-gonic/gin"
)

func DocumentRoutes(r *gin.RouterGroup) {
	documents := r.Group("/documents")
	documents.Use(middleware.AuthMiddleware())
	{
		// Semua user (Staff + Admin) bisa lihat dan download
		documents.GET("", controllers.GetDocuments)
		documents.GET("/", controllers.GetDocuments)
		documents.GET("/:id", controllers.GetDocumentByID)

		// Admin Route
		documents.POST("", middleware.AdminOnly(), controllers.CreateDocument)
		documents.POST("/", middleware.AdminOnly(), controllers.CreateDocument)
		documents.PUT("/:id", middleware.AdminOnly(), controllers.UpdateDocument)
		documents.DELETE("/:id", middleware.AdminOnly(), controllers.DeleteDocument)
	}
}
