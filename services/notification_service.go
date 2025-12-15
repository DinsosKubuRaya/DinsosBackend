package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	ws "dinsos_kuburaya/websocket"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
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
// Firebase App Instance
// =========================
var firebaseApp *firebase.App

func initFirebaseApp() error {
	if firebaseApp != nil {
		return nil
	}

	serviceAccountPath := "notifikasi-dinsos-firebase-adminsdk-fbsvc-e877342c09.json"

	if _, err := os.Stat(serviceAccountPath); os.IsNotExist(err) {
		log.Printf("[Firebase] ❌ Service account file not found at: %s", serviceAccountPath)
		return fmt.Errorf("service account file not found: %s", serviceAccountPath)
	}

	log.Printf("[Firebase] 📁 Loading service account from: %s", serviceAccountPath)

	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Printf("[Firebase] ❌ Failed to initialize app: %v", err)
		return err
	}

	firebaseApp = app
	log.Println("[Firebase] ✅ App initialized successfully")
	return nil
}

// =========================
// Helper: Deteksi tipe token
// =========================
func getTokenType(token string) string {
	if strings.Contains(token, "ExponentPushToken") {
		return "expo"
	} else if len(token) > 100 && !strings.Contains(token, " ") {
		return "fcm"
	}
	return "unknown"
}

// =========================
// Kirim via FCM HTTP v1
// =========================
func sendViaFCM(token, title, body, userID string) error {
	log.Printf("[FCM] 🔥 Sending via FCM HTTP v1 | Token: %s...", token[:20])

	if err := initFirebaseApp(); err != nil {
		log.Printf("[FCM] ❌ Failed to initialize Firebase: %v", err)
		return err
	}

	ctx := context.Background()
	client, err := firebaseApp.Messaging(ctx)
	if err != nil {
		log.Printf("[FCM] ❌ Failed to get messaging client: %v", err)
		return err
	}

	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: map[string]string{
			"user_id":      userID,
			"type":         "notification",
			"click_action": "FLUTTER_NOTIFICATION_CLICK",
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Title:       title,
				Body:        body,
				ChannelID:   "high-priority",
				Sound:       "default",
				Icon:        "notification_icon",
				Color:       "#125696",
				ClickAction: "OPEN_APP",
				Tag:         "dinsos_notification",
			},
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": "10",
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: title,
						Body:  body,
					},
					Sound: "default",
					Badge: func() *int { i := 1; return &i }(),
				},
			},
		},
		Webpush: &messaging.WebpushConfig{
			Notification: &messaging.WebpushNotification{
				Title: title,
				Body:  body,
				Icon:  "https://your-domain.com/icon.png",
			},
		},
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		log.Printf("[FCM] ❌ Failed to send message: %v", err)

		// Jika token invalid, hapus dari database
		if strings.Contains(err.Error(), "registration-token-not-registered") ||
			strings.Contains(err.Error(), "invalid-registration-token") ||
			strings.Contains(err.Error(), "Unregistered") {
			log.Println("[FCM] 🗑️ Invalid token, removing from DB")
			config.DB.Model(&models.User{}).
				Where("id = ?", userID).
				Update("push_token", "")
		}
		return err
	}

	log.Printf("[FCM] ✅ Successfully sent message: %v", response)
	return nil
}

// =========================
// Kirim via Expo (untuk Expo Go)
// =========================
func sendViaExpo(token, title, body, userID string) error {
	log.Printf("[Expo] 📱 Sending via Expo API | Token: %s", token)

	msg := ExpoMessage{
		To:    token,
		Title: title,
		Body:  body,
		Sound: "default",
		Data: map[string]interface{}{
			"user_id":   userID,
			"type":      "notification",
			"channelId": "high-priority",
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Println("[Expo] ❌ JSON marshal error:", err)
		return err
	}

	log.Println("[Expo] 📦 Payload:", string(payload))

	req, err := http.NewRequest("POST",
		"https://exp.host/--/api/v2/push/send",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		log.Println("[Expo] ❌ Error creating request:", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("apns-topic", "com.dinsos.arsipapp")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("[Expo] ❌ Request error:", err)
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("[Expo] 📥 Response status: %d", resp.StatusCode)
	log.Printf("[Expo] 📥 Response body: %s", string(bodyBytes))

	var respBody map[string]interface{}
	json.Unmarshal(bodyBytes, &respBody)

	if resp.StatusCode >= 400 {
		log.Printf("[Expo] ❌ Expo returned error: %v - %v", resp.StatusCode, respBody)

		if strings.Contains(fmt.Sprintf("%v", respBody), "DeviceNotRegistered") ||
			strings.Contains(fmt.Sprintf("%v", respBody), "InvalidCredentials") {
			log.Println("[Expo] 🗑️ Invalid token, removing from DB")
			config.DB.Model(&models.User{}).
				Where("id = ?", userID).
				Update("push_token", "")
		}

		return errors.New("expo push API returned an error")
	}

	return nil
}

// =========================
// Kirim Push Jika Tersedia (MAIN FUNCTION)
// =========================
func sendPushIfAvailable(user models.User, title, body string) error {
	log.Printf("[Push] 🔍 Checking push token for user: %s (%s)", user.ID, user.Name)

	if user.PushToken == nil || *user.PushToken == "" {
		log.Printf("[Push] ⚠️ User %s does NOT have a push token. Skipping.", user.ID)
		return nil
	}

	token := *user.PushToken
	tokenType := getTokenType(token)

	log.Printf("[Push] 📱 Token type: %s | Token: %s...", tokenType, token[:min(30, len(token))])

	switch tokenType {
	case "expo":
		return sendViaExpo(token, title, body, user.ID)
	case "fcm":
		return sendViaFCM(token, title, body, user.ID)
	default:
		log.Printf("[Push] ⚠️ Unknown token type, trying both methods")

		// Coba Expo dulu
		err := sendViaExpo(token, title, body, user.ID)
		if err != nil {
			log.Printf("[Push] ⚠️ Expo failed, trying FCM: %v", err)
			return sendViaFCM(token, title, body, user.ID)
		}
		return nil
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// =========================
// Notify Semua User
// =========================
func NotifyAllUsers(message, link string) {
	log.Println("[NotifyAll] 📢 Sending notification to ALL users")

	var users []models.User
	if err := config.DB.Find(&users).Error; err != nil {
		log.Println("[NotifyAll] ❌ DB error:", err)
		return
	}

	log.Printf("[NotifyAll] 👥 Total users: %d", len(users))

	for _, u := range users {
		log.Printf("[NotifyAll] 🔄 Processing user: %s (%s)", u.ID, u.Name)

		notif := models.Notification{
			UserID:  u.ID,
			Message: message,
			Link:    link,
		}
		if err := config.DB.Create(&notif).Error; err != nil {
			log.Println("[NotifyAll] ❌ Error creating notification:", err)
		}

		EmitNewNotification(u.ID, "new_notification")

		go func(user models.User) {
			log.Printf("[NotifyAll] 🚀 Sending push async for user: %s", user.ID)
			if err := sendPushIfAvailable(user, "Notifikasi Baru", message); err != nil {
				log.Printf("[NotifyAll] ❌ Push error for user %s: %v", user.ID, err)
			}
		}(u)
	}
}

// =========================
// Notify Admins
// =========================
func NotifyAdmins(message, link string) {
	log.Println("[NotifyAdmins] 📢 Sending notification ONLY to admins")

	var users []models.User
	if err := config.DB.Where("role IN ?", []string{"admin", "superadmin"}).Find(&users).Error; err != nil {
		log.Println("[NotifyAdmins] ❌ DB error:", err)
		return
	}

	log.Printf("[NotifyAdmins] 👥 Total admin users: %d", len(users))

	for _, u := range users {
		log.Printf("[NotifyAdmins] 🔄 Processing user: %s (%s)", u.ID, u.Name)

		notif := models.Notification{
			UserID:  u.ID,
			Message: message,
			Link:    link,
		}
		if err := config.DB.Create(&notif).Error; err != nil {
			log.Println("[NotifyAdmins] ❌ Create notification error:", err)
		}

		EmitNewNotification(u.ID, "new_notification")

		go func(user models.User) {
			log.Printf("[NotifyAdmins] 🚀 Sending push async for user: %s", user.ID)
			if err := sendPushIfAvailable(user, "Notifikasi Admin", message); err != nil {
				log.Printf("[NotifyAdmins] ❌ Push admin error for user %s: %v", user.ID, err)
			}
		}(u)
	}
}

// =========================
// Notify User Tertentu
// =========================
func NotifySpecificUser(userID, message, link string) {
	log.Printf("[NotifySpecific] 📢 Sending notification to: %s", userID)

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		log.Printf("[NotifySpecific] ❌ User not found: %s", userID)
		return
	}

	log.Printf("[NotifySpecific] ✅ User found: %s | Token: %v", user.ID, user.PushToken != nil)

	notif := models.Notification{
		UserID:  userID,
		Message: message,
		Link:    link,
	}
	if err := config.DB.Create(&notif).Error; err != nil {
		log.Println("[NotifySpecific] ❌ DB create error:", err)
	}

	EmitNewNotification(userID, "new_notification")

	go func(u models.User) {
		log.Printf("[NotifySpecific] 🚀 Sending push async for user: %s", u.ID)
		if err := sendPushIfAvailable(u, "Notifikasi Baru", message); err != nil {
			log.Printf("[NotifySpecific] ❌ Push error for user %s: %v", u.ID, err)
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

// =========================
// Test Firebase Connection
// =========================
func TestFirebaseConnection() error {
	log.Println("[Test] 🔧 Testing Firebase connection...")

	if err := initFirebaseApp(); err != nil {
		log.Printf("[Test] ❌ Firebase init failed: %v", err)
		return err
	}

	log.Println("[Test] ✅ Firebase connection successful")
	return nil
}
