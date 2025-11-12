package routes

import (
	"dinsos_kuburaya/controllers"

	"github.com/gin-gonic/gin"
)

func DocumentRoutes(r *gin.Engine) {
	api := r.Group("/api/documents")
	{
		api.POST("/", controllers.CreateDocument)
		api.GET("/", controllers.GetDocuments)
		api.GET("/:id", controllers.GetDocumentByID)
		api.PUT("/:id", controllers.UpdateDocument)
		api.DELETE("/:id", controllers.DeleteDocument)
	}
}
