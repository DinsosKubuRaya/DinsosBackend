package controllers

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"

	"github.com/gin-gonic/gin"
)

// ======================================================
// CREATE (Semua staff bisa upload)
// ======================================================
func CreateDocumentStaff(c *gin.Context) {
	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userRaw.(models.User)

	sender := c.PostForm("sender")
	subject := c.PostForm("subject")
	letterType := c.PostForm("letter_type")

	if sender == "" || subject == "" || letterType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sender, subject, dan letter_type wajib diisi"})
		return
	}

	if letterType != "masuk" && letterType != "keluar" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "letter_type harus 'masuk' atau 'keluar'"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File tidak ditemukan"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat membuka file"})
		return
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	var resourceType string
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		resourceType = "image"
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		resourceType = "raw"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung"})
		return
	}

	url, publicID, err := config.UploadToCloudinary(src, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
		return
	}

	document := models.DocumentStaff{
		UserID:       user.ID,
		Sender:       sender,
		Subject:      subject,
		LetterType:   letterType,
		FileName:     url,
		PublicID:     publicID,
		ResourceType: resourceType,
	}

	if err := config.DB.Create(&document).Error; err != nil {
		config.DeleteFromCloudinary(publicID, resourceType)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error: " + err.Error()})
		return
	}

	config.DB.Preload("User").Find(&document)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Dokumen berhasil diupload",
		"document": document,
	})
}

// ======================================================
// GET ALL (Semua staff bisa lihat semua dokumen staff)
// ======================================================
func GetDocumentStaffs(c *gin.Context) {
	var documents []models.DocumentStaff

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
	search := c.Query("search")
	letterType := c.Query("letter_type")

	query := config.DB.Model(&models.DocumentStaff{}).
		Joins("LEFT JOIN users AS User ON document_staffs.user_id = User.id")

	if search != "" {
		searchQuery := "%" + search + "%"
		query = query.Where(
			"document_staffs.sender LIKE ? OR document_staffs.subject LIKE ? OR document_staffs.file_name LIKE ? OR User.name LIKE ?",
			searchQuery, searchQuery, searchQuery, searchQuery,
		)
	}

	if letterType != "" && letterType != "all" {
		query = query.Where("document_staffs.letter_type = ?", letterType)
	}

	var total int64
	countQuery := config.DB.Model(&models.DocumentStaff{}).
		Joins("LEFT JOIN users AS User ON document_staffs.user_id = User.id")

	if search != "" {
		searchQuery := "%" + search + "%"
		countQuery = countQuery.Where(
			"document_staffs.sender LIKE ? OR document_staffs.subject LIKE ? OR document_staffs.file_name LIKE ? OR User.name LIKE ?",
			searchQuery, searchQuery, searchQuery, searchQuery,
		)
	}

	if letterType != "" && letterType != "all" {
		countQuery = countQuery.Where("document_staffs.letter_type = ?", letterType)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		total = 0
	}

	offset := (page - 1) * perPage
	err := query.
		Select("document_staffs.*").
		Offset(offset).
		Limit(perPage).
		Order("document_staffs.created_at DESC").
		Preload("User").
		Find(&documents).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil dokumen"})
		return
	}

	lastPage := 0
	if perPage > 0 {
		lastPage = int(total) / perPage
		if int(total)%perPage != 0 {
			lastPage++
		}
	}
	if lastPage == 0 && total > 0 {
		lastPage = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"documents":    documents,
		"total":        total,
		"current_page": page,
		"last_page":    lastPage,
		"per_page":     perPage,
	})
}

// ======================================================
// GET BY ID
// ======================================================
func GetDocumentStaffByID(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

	if err := config.DB.Preload("User").First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"document": document})
}

// ======================================================
// UPDATE (SEMUA STAFF & ADMIN BISA UPDATE)
// ======================================================
func UpdateDocumentStaff(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

	if err := config.DB.First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	// ✅ TIDAK ADA CHECK OWNERSHIP - Semua staff bisa edit

	sender := c.PostForm("sender")
	subject := c.PostForm("subject")
	letterType := c.PostForm("letter_type")

	updates := map[string]interface{}{}

	if sender != "" {
		updates["sender"] = sender
	}
	if subject != "" {
		updates["subject"] = subject
	}
	if letterType != "" {
		if letterType != "masuk" && letterType != "keluar" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "letter_type harus 'masuk' atau 'keluar'"})
			return
		}
		updates["letter_type"] = letterType
	}

	if len(updates) > 0 {
		if err := config.DB.Model(&document).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan"})
			return
		}
	}

	config.DB.Preload("User").Find(&document)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Dokumen berhasil diperbarui",
		"document": document,
	})
}

// ======================================================
// DELETE (SEMUA STAFF & ADMIN BISA HAPUS)
// ======================================================
func DeleteDocumentStaff(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

	if err := config.DB.First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	config.DeleteFromCloudinary(document.PublicID, document.ResourceType)
	config.DB.Delete(&document)

	c.JSON(http.StatusOK, gin.H{"message": "Dokumen berhasil dihapus"})
}

// ======================================================
// DOWNLOAD DOCUMENT STAFF (Redirect ke Cloudinary)
// ======================================================
func DownloadDocumentStaff(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

	if err := config.DB.First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	// Redirect ke URL Cloudinary
	if document.FileName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "File tidak tersedia"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, document.FileName)
}
