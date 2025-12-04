package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/middleware"
	"dinsos_kuburaya/models"
	"dinsos_kuburaya/routes"
	"dinsos_kuburaya/utils"

	// 🔥 Tambahkan ini
	ws "dinsos_kuburaya/websocket"
)

func main() {
	r := gin.Default()
	r.MaxMultipartMemory = 100 << 20

	config.ConnectDatabase()
	utils.StartActivityLogCleaner()
	utils.StartNotificationCleaner()

	if err := config.DB.AutoMigrate(
		&models.User{},
		&models.Document{},
		&models.SecretToken{},
		&models.DocumentStaff{},
		&models.Notification{},
		&models.ActivityLog{},
	); err != nil {
		log.Fatal("Gagal migrasi tabel:", err)
	}

	r.Use(middleware.RateLimiter())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.XSSBlocker())

	websocketHub := ws.NewHub()
	go websocketHub.Run()

	// Simpan hub ke tempat global agar bisa dipakai oleh service
	ws.HubInstance = websocketHub
	r.GET("/ws/notifications", ws.WebSocketHandler(websocketHub))

	api := r.Group("/api")
	{
		routes.LoginRoutes(api)
		routes.LogoutRoutes(api)
		routes.UserRoutes(api)
		routes.DocumentRoutes(api)
		routes.DocumentStaffRoutes(api)
		routes.NotificationRoutes(api)
		routes.ActivityLogRoutes(api)
	}

	// ======================================================
	// RUN SERVER
	// ======================================================
	log.Println("✅ Server berjalan di port 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Gagal menjalankan server:", err)
	}
}
