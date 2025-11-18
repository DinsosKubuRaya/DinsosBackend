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
		// SEMUA USER (Staff + Admin) - Read & Download
		documents.GET("", controllers.GetDocuments)
		documents.GET("/", controllers.GetDocuments)
		documents.GET("/:id", controllers.GetDocumentByID)
		documents.GET("/download/:id", controllers.DownloadDocument)

		// HANYA ADMIN - Create, Update, Delete
		documents.POST("", middleware.AdminOnly(), controllers.CreateDocument)
		documents.POST("/", middleware.AdminOnly(), controllers.CreateDocument)
		documents.PUT("/:id", middleware.AdminOnly(), controllers.UpdateDocument)
		documents.DELETE("/:id", middleware.AdminOnly(), controllers.DeleteDocument)
	}
}
