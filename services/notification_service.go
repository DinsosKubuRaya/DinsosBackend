package services

import (
	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	ws "dinsos_kuburaya/websocket"
)

func NotifyAllUsers(message, link string) {
	var users []models.User
	config.DB.Find(&users)

	for _, u := range users {
		notif := models.Notification{
			UserID:  u.ID, // string
			Message: message,
			Link:    link,
		}
		config.DB.Create(&notif)

		EmitNewNotification(u.ID, "new_notification")
	}
}

func NotifyAdmins(message, link string) {
	var users []models.User

	// Hanya admin & superadmin yang boleh menerima
	config.DB.Where("role IN ?", []string{"admin", "superadmin"}).Find(&users)

	for _, u := range users {
		notif := models.Notification{
			UserID:  u.ID,
			Message: message,
			Link:    link,
		}
		config.DB.Create(&notif)

		EmitNewNotification(u.ID, "new_notification")
	}
}

func NotifySpecificUser(userID, message, link string) {
	notif := models.Notification{
		UserID:  userID,
		Message: message,
		Link:    link,
	}

	config.DB.Create(&notif)

	EmitNewNotification(userID, "new_notification")
}

func EmitNewNotification(userID string, message string) {
	if ws.HubInstance == nil {
		return
	}

	ws.HubInstance.Emit(ws.NotificationEvent{
		UserID:  userID, // string
		Type:    "notification_added",
		Message: message,
	})
}
