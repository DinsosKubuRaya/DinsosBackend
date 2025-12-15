package routes

import (
	"dinsos_kuburaya/controllers"
	"dinsos_kuburaya/middleware"

	"github.com/gin-gonic/gin"
)

func DocumentRoutes(r *gin.RouterGroup) {
	documents := r.Group("/documents")
	documents.Use(middleware.AuthMiddleware())

	documents.GET("", controllers.GetDocuments)

	documents.GET("/", controllers.GetDocuments)

	documents.GET("/:id", controllers.GetDocumentByID)

	documents.GET("/summary", controllers.GetDocumentSummary)

	documents.Use(middleware.RoleMiddleware("admin", "superadmin"))
	{
		documents.GET("/:id/download", controllers.DownloadDocument)

		documents.POST("", controllers.CreateDocument)

		documents.POST("/", controllers.CreateDocument)

		documents.PUT("/:id", controllers.UpdateDocument)

		documents.DELETE("/:id", controllers.DeleteDocument)
	}
}
