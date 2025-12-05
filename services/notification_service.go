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
	log.Println("[Push] Checking push token for user:", user.ID)

	if user.PushToken == nil || *user.PushToken == "" {
		log.Println("[Push] User", user.ID, "does NOT have a push token. Skipping.")
		return nil
	}

	log.Println("[Push] Sending push to token:", *user.PushToken)

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
		log.Println("[Push] JSON marshal error:", err)
		return err
	}

	log.Println("[Push] Payload:", string(payload))

	req, err := http.NewRequest("POST",
		"https://exp.host/--/api/v2/push/send",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		log.Println("[Push] Error creating request:", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("[Push] Request error:", err)
		return err
	}
	defer resp.Body.Close()

	log.Println("[Push] Expo response status:", resp.StatusCode)

	var respBody map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respBody)
	log.Println("[Push] Expo response body:", respBody)

	if resp.StatusCode >= 400 {
		log.Println("[Push] Expo returned error status:", resp.StatusCode)
		return errors.New("Expo Push API error")
	}

	return nil
}

// =========================
// Notify Semua User
// =========================
func NotifyAllUsers(message, link string) {
	log.Println("[NotifyAll] Sending notification to ALL users")

	var users []models.User
	if err := config.DB.Find(&users).Error; err != nil {
		log.Println("[NotifyAll] DB error:", err)
		return
	}

	log.Println("[NotifyAll] Total users:", len(users))

	for _, u := range users {
		log.Println("[NotifyAll] Processing user:", u.ID)

		notif := models.Notification{
			UserID:  u.ID,
			Message: message,
			Link:    link,
		}
		if err := config.DB.Create(&notif).Error; err != nil {
			log.Println("[NotifyAll] Error creating notification:", err)
		}

		EmitNewNotification(u.ID, "new_notification")

		go func(user models.User) {
			log.Println("[NotifyAll] Sending push async for user:", user.ID)
			if err := sendPushIfAvailable(user, "Notifikasi Baru", message); err != nil {
				log.Println("[NotifyAll] Push error:", err)
			}
		}(u)
	}
}

// =========================
// Notify Admins
// =========================
func NotifyAdmins(message, link string) {
	log.Println("[NotifyAdmins] Sending notification ONLY to admins")

	var users []models.User
	if err := config.DB.Where("role IN ?", []string{"admin", "superadmin"}).Find(&users).Error; err != nil {
		log.Println("[NotifyAdmins] DB error:", err)
		return
	}

	log.Println("[NotifyAdmins] Total admin users:", len(users))

	for _, u := range users {
		log.Println("[NotifyAdmins] Processing user:", u.ID)

		notif := models.Notification{
			UserID:  u.ID,
			Message: message,
			Link:    link,
		}
		if err := config.DB.Create(&notif).Error; err != nil {
			log.Println("[NotifyAdmins] Create notification error:", err)
		}

		EmitNewNotification(u.ID, "new_notification")

		go func(user models.User) {
			log.Println("[NotifyAdmins] Sending push async for user:", user.ID)
			if err := sendPushIfAvailable(user, "Notifikasi Admin", message); err != nil {
				log.Println("[NotifyAdmins] Push admin error:", err)
			}
		}(u)
	}
}

// =========================
// Notify User Tertentu
// =========================
func NotifySpecificUser(userID, message, link string) {
	log.Println("[NotifySpecific] Sending notification to:", userID)

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		log.Println("[NotifySpecific] User not found:", userID)
		return
	}

	log.Println("[NotifySpecific] User found:", user.ID, "Token:", user.PushToken)

	notif := models.Notification{
		UserID:  userID,
		Message: message,
		Link:    link,
	}
	if err := config.DB.Create(&notif).Error; err != nil {
		log.Println("[NotifySpecific] DB create error:", err)
	}

	EmitNewNotification(userID, "new_notification")

	go func(u models.User) {
		log.Println("[NotifySpecific] Sending push async for user:", u.ID)
		if err := sendPushIfAvailable(u, "Notifikasi Baru", message); err != nil {
			log.Println("[NotifySpecific] Push error:", err)
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
