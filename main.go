package main

import (
	"log"
	"net/http"

	"dinsos_kuburaya/middleware" // ganti sesuai nama modul kamu

	"github.com/gin-gonic/gin"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"dinsos_kuburaya/routes"
)

func main() {
	r := gin.Default()

	routes.UserRoutes(r)
	routes.DocumentRoutes(r)

	config.ConnectDatabase()

	if err := config.DB.AutoMigrate(&models.User{}, &models.Document{}); err != nil {
		log.Fatal("Gagal migrasi tabel:", err)
	}

	// Gunakan rate limiter
	r.Use(middleware.RateLimiter())
	r.Use(middleware.CORSMiddleware())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Halo, masih dalam batas aman 😎",
		})
	})

	r.Run(":8080")
}
