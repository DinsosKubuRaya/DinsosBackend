package routes

import (
	"dinsos_kuburaya/controllers"
	"dinsos_kuburaya/middleware"

	"github.com/gin-gonic/gin"
)

func DocumentStaffRoutes(r *gin.RouterGroup) {
	docStaff := r.Group("/document_staff")
	docStaff.Use(middleware.AuthMiddleware())
	{
		// STAFF & ADMIN - Read
		docStaff.GET("", controllers.GetDocumentStaffs)
		docStaff.GET("/", controllers.GetDocumentStaffs)
		docStaff.GET("/:id", controllers.GetDocumentStaffByID)
		docStaff.GET("/personal", controllers.GetPersonalDocumentStaffs)

		docStaff.GET("/:id/download", controllers.DownloadDocumentStaff)

		// STAFF - Create
		docStaff.POST("", controllers.CreateDocumentStaff)
		docStaff.POST("/", controllers.CreateDocumentStaff)

		// STAFF - Update & Delete
		docStaff.PUT("/:id", controllers.UpdateDocumentStaff)
		docStaff.DELETE("/:id", controllers.DeleteDocumentStaff)
	}
}
