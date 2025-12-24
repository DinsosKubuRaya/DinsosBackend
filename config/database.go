package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	// Load .env (aman, di Railway akan dilewati)
	_ = godotenv.Load()

	dsn := os.Getenv("MYSQL_URL")
	if dsn == "" {
		log.Fatal("❌ MYSQL_URL tidak ditemukan")
	}

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Gagal koneksi database:", err)
	}

	// aktifkan proteksi SQL injection
	registerQueryProtector(database)

	DB = database
	log.Println("✅ Database Railway MySQL terkoneksi")
}
