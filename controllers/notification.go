package controllers

import (
	"net/http"
	"strconv"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"

	"github.com/gin-gonic/gin"
)

// GetNotifications - Ambil semua notifikasi user yang login
func GetNotifications(c *gin.Context) {
	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User tidak terautentikasi",
		})
		return
	}

	user := userRaw.(models.User)
	userID := user.ID

	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 15
	}

	offset := (page - 1) * limit

	var notifications []models.Notification

	if err := config.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil notifikasi",
		})
		return
	}

	var unreadCount int64
	config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&unreadCount)

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"unread_count":  unreadCount,
		"page":          page,
		"limit":         limit,
		"has_more":      len(notifications) == limit,
	})
}

// MarkNotificationAsRead - Tandai notifikasi sebagai sudah dibaca
func MarkNotificationAsRead(c *gin.Context) {
	notificationID := c.Param("id")

	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User tidak terautentikasi",
		})
		return
	}

	user, ok := userRaw.(models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Tipe data user di context tidak valid",
		})
		return
	}
	userIDStr := user.ID

	var notification models.Notification

	if err := config.DB.Where("id = ? AND user_id = ?", notificationID, userIDStr).
		First(&notification).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Notifikasi tidak ditemukan",
		})
		return
	}

	if !notification.IsRead {
		notification.IsRead = true
		if err := config.DB.Save(&notification).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Gagal memperbarui notifikasi",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notifikasi berhasil ditandai sebagai dibaca",
	})
}

func MarkAllAsRead(c *gin.Context) {
	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userRaw.(models.User)

	result := config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", user.ID, false).
		Update("is_read", true)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "All notifications marked as read",
		"updated_count": result.RowsAffected,
	})
}
