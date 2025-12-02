package controllers

import (
	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// API: Mengambil semua data log untuk ditampilkan di dashboard
func GetAllActivityLogs(c *gin.Context) {
	var logs []models.ActivityLog

	// Ambil query params
	page := 1
	limit := 20

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	// Hitung offset
	offset := (page - 1) * limit

	// Ambil data dengan limit + offset
	if err := config.DB.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil log aktivitas"})
		return
	}

	// Hitung total untuk pagination (optional)
	var total int64
	config.DB.Model(&models.ActivityLog{}).Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"page":  page,
		"limit": limit,
		"total": total,
		"data":  logs,
	})
}
