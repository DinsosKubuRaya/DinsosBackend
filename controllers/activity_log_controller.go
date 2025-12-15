package controllers

import (
	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetAllActivityLogs(c *gin.Context) {
	var logs []models.ActivityLog

	page := 1
	limit := 20

	if p := c.Query("page"); p != "" {
		fmt.Sscanf(p, "%d", &page)
	}
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	offset := (page - 1) * limit

	if err := config.DB.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil log aktivitas"})
		return
	}

	var total int64
	config.DB.Model(&models.ActivityLog{}).Count(&total)

	c.JSON(http.StatusOK, gin.H{
		"page":  page,
		"limit": limit,
		"total": total,
		"data":  logs,
	})
}
