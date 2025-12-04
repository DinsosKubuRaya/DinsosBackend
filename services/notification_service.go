package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	ws "dinsos_kuburaya/websocket"
)

// =========================
// Struct Push Notification
// =========================
type ExpoMessage struct {
	To    string                 `json:"to"`
	Title string                 `json:"title"`
	Body  string                 `json:"body"`
	Sound string                 `json:"sound"`
	Data  map[string]interface{} `json:"data"`
}

// =========================
// Kirim Push Jika Tersedia
// =========================
func sendPushIfAvailable(user models.User, title, body string) error {
	if user.PushToken == nil || *user.PushToken == "" {
		return nil // user tidak punya push token → skip
	}

	msg := ExpoMessage{
		To:    *user.PushToken,
		Title: title,
		Body:  body,
		Sound: "default",
		Data: map[string]interface{}{
			"user_id": user.ID,
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://exp.host/--/api/v2/push/send", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return errors.New("Expo Push API error")
	}

	return nil
}

// =========================
// Notify Semua User
// =========================
func NotifyAllUsers(message, link string) {
	var users []models.User
	if err := config.DB.Find(&users).Error; err != nil {
		log.Println("NotifyAllUsers DB error:", err)
		return
	}

	for _, u := range users {
		notif := models.Notification{
			UserID:  u.ID,
			Message: message,
			Link:    link,
		}
		if err := config.DB.Create(&notif).Error; err != nil {
			log.Println("Error creating notification:", err)
		}

		// WebSocket
		EmitNewNotification(u.ID, "new_notification")

		// Push notification (jalankan async agar tidak blocking)
		go func(user models.User) {
			if err := sendPushIfAvailable(user, "Notifikasi Baru", message); err != nil {
				log.Println("Push error:", err)
			}
		}(u)
	}
}

// =========================
// Notify Admins
// =========================
func NotifyAdmins(message, link string) {
	var users []models.User
	if err := config.DB.Where("role IN ?", []string{"admin", "superadmin"}).Find(&users).Error; err != nil {
		log.Println("NotifyAdmins DB error:", err)
		return
	}

	for _, u := range users {
		notif := models.Notification{
			UserID:  u.ID,
			Message: message,
			Link:    link,
		}
		if err := config.DB.Create(&notif).Error; err != nil {
			log.Println("Error creating admin notification:", err)
		}

		EmitNewNotification(u.ID, "new_notification")

		go func(user models.User) {
			if err := sendPushIfAvailable(user, "Notifikasi Admin", message); err != nil {
				log.Println("Push admin error:", err)
			}
		}(u)
	}
}

// =========================
// Notify User Tertentu
// =========================
func NotifySpecificUser(userID, message, link string) {
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		log.Println("NotifySpecificUser: user not found:", userID)
		return
	}

	notif := models.Notification{
		UserID:  userID,
		Message: message,
		Link:    link,
	}
	if err := config.DB.Create(&notif).Error; err != nil {
		log.Println("Error creating specific user notification:", err)
	}

	EmitNewNotification(userID, "new_notification")

	go func(u models.User) {
		if err := sendPushIfAvailable(u, "Notifikasi Baru", message); err != nil {
			log.Println("Push specific user error:", err)
		}
	}(user)
}

// =========================
// Emit WebSocket Notification
// =========================
func EmitNewNotification(userID string, message string) {
	if ws.HubInstance == nil {
		return
	}

	ws.HubInstance.Emit(ws.NotificationEvent{
		UserID:  userID,
		Type:    "notification_added",
		Message: message,
	})
}
