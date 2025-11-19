package utils

import (
	"dinsos_kuburaya/config"
	"log"
	"time"
)

func StartActivityLogCleaner() {
	go func() {
		for {
			// Jalankan sekali sehari
			time.Sleep(24 * time.Hour)

			// Hapus log yang lebih dari 30 hari
			err := config.DB.
				Where("created_at <= ?", time.Now().AddDate(0, 0, -30)).
				Delete(nil, "activity_logs").
				Error

			if err != nil {
				log.Println("❌ Gagal menghapus activity log lama:", err)
			} else {
				log.Println("🧹 Log lama (>30 hari) berhasil dibersihkan")
			}
		}
	}()
}
