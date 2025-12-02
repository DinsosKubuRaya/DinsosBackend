package services

import (
	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"log"
)

func CreateActivity(userID, userName, action, message string) {
	logData := models.ActivityLog{
		UserID:   userID,
		UserName: userName,
		Action:   action,
		Message:  message,
	}

	if err := config.DB.Create(&logData).Error; err != nil {
		log.Println("Gagal menyimpan activity log:", err)
	}
}
