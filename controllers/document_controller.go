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
// CREATE DOCUMENT
// ======================================================
func CreateDocument(c *gin.Context) {
	sender := c.PostForm("sender")
	subject := c.PostForm("subject")
	letterType := c.PostForm("letter_type")

	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User tidak terautentikasi"})
		return
	}
	user := userRaw.(models.User)

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "format file tidak didukung."})
		return
	}

	url, publicID, err := config.UploadToCloudinary(src, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
		return
	}

	document := models.Document{
		Sender:       sender,
		Subject:      subject,
		LetterType:   letterType,
		FileName:     url,
		PublicID:     publicID,
		ResourceType: resourceType,
		UserID:       &user.ID,
	}

	if err := config.DB.Create(&document).Error; err != nil {
		config.DeleteFromCloudinary(publicID, resourceType)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error: " + err.Error()})
		return
	}

	config.DB.Preload("User").Find(&document)

	c.JSON(http.StatusCreated, gin.H{"document": document})
}

// ======================================================
// GET ALL WITH SEARCH (FIXED - PostgreSQL Compatible)
// ======================================================
func GetDocuments(c *gin.Context) {
	var documents []models.Document

	// Pagination & Filters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
	letterType := c.Query("letter_type")
	search := c.Query("search")

	// 1. Base query dengan JOIN ke tabel users
	// FIXED: Gunakan syntax yang kompatibel dengan MySQL & PostgreSQL
	query := config.DB.Model(&models.Document{}).
		Joins("LEFT JOIN users AS User ON documents.user_id = User.id")

	// 2. ✅ FILTER SEARCH (FIXED)
	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(documents.sender) LIKE ? OR LOWER(documents.subject) LIKE ? OR LOWER(documents.file_name) LIKE ? OR LOWER(User.name) LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// 3. Filter Tipe Surat
	if letterType != "" && letterType != "all" {
		query = query.Where("documents.letter_type = ?", letterType)
	}

	// 4. Hitung total (sebelum pagination)
	var total int64
	// FIXED: Count harus dilakukan pada subquery untuk akurasi
	countQuery := config.DB.Model(&models.Document{}).
		Joins("LEFT JOIN users AS User ON documents.user_id = User.id")

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		countQuery = countQuery.Where(
			"LOWER(documents.sender) LIKE ? OR LOWER(documents.subject) LIKE ? OR LOWER(documents.file_name) LIKE ? OR LOWER(User.name) LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	if letterType != "" && letterType != "all" {
		countQuery = countQuery.Where("documents.letter_type = ?", letterType)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		total = 0
	}

	// 5. Ambil hasil dengan pagination DAN sorting
	offset := (page - 1) * perPage

	// FIXED: Select distinct untuk menghindari duplikasi dari JOIN
	err := query.
		Select("documents.*").
		Offset(offset).
		Limit(perPage).
		Order("documents.created_at DESC").
		Preload("User").
		Find(&documents).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil dokumen"})
		return
	}

	// Perhitungan Last Page
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
func GetDocumentByID(c *gin.Context) {
	id := c.Param("id")
	var document models.Document

	if err := config.DB.Preload("User").First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"document": document})
}

// ======================================================
// UPDATE DOCUMENT
// ======================================================
func UpdateDocument(c *gin.Context) {
	id := c.Param("id")
	var document models.Document

	if err := config.DB.First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

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
		updates["letter_type"] = letterType
	}

	fileHeader, err := c.FormFile("file")
	if err == nil {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung."})
			return
		}

		url, publicID, err := config.UploadToCloudinary(src, resourceType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
			return
		}

		if document.PublicID != "" {
			config.DeleteFromCloudinary(document.PublicID, document.ResourceType)
		}

		updates["file_name"] = url
		updates["public_id"] = publicID
		updates["resource_type"] = resourceType
	}

	if len(updates) > 0 {
		if err := config.DB.Model(&document).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan: " + err.Error()})
			return
		}
	}

	config.DB.Preload("User").Find(&document)

	c.JSON(http.StatusOK, gin.H{"message": "Dokumen berhasil diperbarui", "document": document})
}

// ======================================================
// DELETE
// ======================================================
func DeleteDocument(c *gin.Context) {
	id := c.Param("id")
	var document models.Document

	if err := config.DB.First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	config.DeleteFromCloudinary(document.PublicID, document.ResourceType)
	config.DB.Delete(&document)

	c.JSON(http.StatusOK, gin.H{"message": "Dokumen berhasil dihapus"})
}

// ======================================================
// DOWNLOAD DOCUMENT (Redirect ke Cloudinary)
// ======================================================
func DownloadDocument(c *gin.Context) {
	id := c.Param("id")
	var document models.Document

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
