package controllers

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"

	"github.com/gin-gonic/gin"
)

// ======================================================
// CREATE STAFF DOCUMENT
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

	// 1. BUKA FILE
	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat membuka file"})
		return
	}
	defer src.Close()

	// 2. BACA KE BUFFER (Fix File Corrupt)
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file buffer"})
		return
	}
	reader := bytes.NewReader(fileBytes)

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	var resourceType string
	var folder string // <-- 1. Variable folder ditambahkan

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		resourceType = "image"
		folder = "gambar" // <-- Set folder gambar
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		resourceType = "raw"
		folder = "arsip" // <-- Set folder arsip
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung"})
		return
	}

	// 3. UPLOAD KE CLOUDINARY (Kirim 4 Parameter: Reader, Filename, Folder, Type)
	uploadResult, err := config.UploadToCloudinary(reader, fileHeader.Filename, folder, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
		return
	}

	// 4. SIMPAN DB
	document := models.DocumentStaff{
		UserID:     user.ID,
		Sender:     sender,
		Subject:    subject,
		LetterType: letterType,
		// Simpan URL ke FileName
		FileName:     uploadResult.SecureURL,
		PublicID:     uploadResult.PublicID,
		ResourceType: resourceType,
	}

	if err := config.DB.Create(&document).Error; err != nil {
		config.DeleteFromCloudinary(uploadResult.PublicID, resourceType)
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
// GET ALL STAFF DOCUMENTS
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
// UPDATE STAFF DOCUMENT
// ======================================================
func UpdateDocumentStaff(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

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
		if letterType != "masuk" && letterType != "keluar" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "letter_type harus 'masuk' atau 'keluar'"})
			return
		}
		updates["letter_type"] = letterType
	}

	// HANDLE UPLOAD JIKA ADA FILE BARU
	fileHeader, err := c.FormFile("file")
	if err == nil {
		src, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat membuka file"})
			return
		}
		defer src.Close()

		// Buffer
		fileBytes, err := io.ReadAll(src)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file buffer"})
			return
		}
		reader := bytes.NewReader(fileBytes)

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		var resourceType string
		var folder string // <-- 1. Variable folder ditambahkan

		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
			resourceType = "image"
			folder = "gambar" // <-- Set folder gambar
		case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
			resourceType = "raw"
			folder = "arsip" // <-- Set folder arsip
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format file tidak didukung"})
			return
		}

		// 3. UPLOAD KE CLOUDINARY (Dengan Folder)
		uploadResult, err := config.UploadToCloudinary(reader, fileHeader.Filename, folder, resourceType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload gagal: " + err.Error()})
			return
		}

		// Hapus file lama
		if document.PublicID != "" {
			config.DeleteFromCloudinary(document.PublicID, document.ResourceType)
		}

		updates["file_name"] = uploadResult.SecureURL
		updates["public_id"] = uploadResult.PublicID
		updates["resource_type"] = resourceType
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
// DELETE
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
// DOWNLOAD (Redirect)
// ======================================================
func DownloadDocumentStaff(c *gin.Context) {
	id := c.Param("id")
	var document models.DocumentStaff

	if err := config.DB.First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	if document.FileName == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "File tidak tersedia"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, document.FileName)
}
