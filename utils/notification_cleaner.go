package utils

import (
	"dinsos_kuburaya/config"
	"log"
	"time"
)

func StartNotificationCleaner() {
	go func() {
		for {
			time.Sleep(24 * time.Hour) // jalan 1x sehari

			err := config.DB.
				Where("created_at <= ?", time.Now().AddDate(0, 0, -30)).
				Delete(nil, "notifications").
				Error

			if err != nil {
				log.Println("❌ Gagal menghapus notifikasi lama:", err)
			} else {
				log.Println("🧹 Notifikasi lama (>30 hari) dibersihkan")
			}
		}
	}()
}
