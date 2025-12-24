package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	_ = godotenv.Load()

	rawURL := os.Getenv("MYSQL_URL")
	if rawURL == "" {
		log.Fatal("❌ MYSQL_URL tidak ditemukan")
	}

	// convert mysql://user:pass@host:port/db
	dsn := strings.TrimPrefix(rawURL, "mysql://") +
		"?charset=utf8mb4&parseTime=True&loc=Local"

	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Gagal koneksi database:", err)
	}

	registerQueryProtector(database)

	DB = database
	log.Println("✅ Database Railway MySQL terkoneksi")
}
