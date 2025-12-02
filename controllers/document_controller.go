package controllers

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"dinsos_kuburaya/services"

	"github.com/gin-gonic/gin"
)

// =======================
// CREATE DOCUMENT
// =======================
func CreateDocument(c *gin.Context) {
	sender := c.PostForm("sender")
	subject := c.PostForm("subject")
	letterType := c.PostForm("letter_type")

	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file"})
		return
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca data file"})
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	var resourceType, folder string

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		resourceType = "image"
		folder = "gambar"
	case ".pdf":
		resourceType = "raw"
		folder = "arsip"
	default:
		resourceType = "raw"
		folder = "arsip"
	}

	reader := bytes.NewReader(fileBytes)
	uploadResult, err := config.UploadToCloudinary(reader, fileHeader.Filename, folder, resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cloudinary upload gagal: " + err.Error()})
		return
	}

	userID := user.ID
	document := models.Document{
		Sender:       sender,
		FileName:     fileHeader.Filename,
		FileURL:      uploadResult.SecureURL,
		Subject:      subject,
		LetterType:   letterType,
		UserID:       &userID,
		PublicID:     uploadResult.PublicID,
		ResourceType: uploadResult.ResourceType,
	}

	if err := config.DB.Create(&document).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan dokumen di database"})
		return
	}

	// LOG + NOTIF
	services.CreateActivity(user.ID, user.Name, "create", "Mengunggah dokumen: "+document.FileName)
	services.NotifyAllUsers("Dokumen baru diunggah: "+document.FileName, "/documents/"+document.ID)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Dokumen berhasil diupload",
		"document": document,
		"file_url": uploadResult.SecureURL,
	})
}

// =======================
// GET ALL DOCUMENTS
// =======================
func GetDocuments(c *gin.Context) {
	var documents []models.Document

	search := c.Query("search")
	letterType := c.Query("letter_type")

	query := config.DB.Preload("User")
	if letterType != "" && letterType != "all" {
		query = query.Where("letter_type = ?", letterType)
	}
	if search != "" {
		s := "%" + search + "%"
		query = query.Where("sender LIKE ? OR subject LIKE ? OR file_name LIKE ?", s, s, s)
	}

	if err := query.Order("created_at DESC").Find(&documents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data dokumen"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": documents,
		"total":     len(documents),
	})
}

// =======================
// GET DOCUMENT BY ID
// =======================
func GetDocumentByID(c *gin.Context) {
	id := c.Param("id")
	var document models.Document

	if err := config.DB.Preload("User").First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"document": document})
}

// =======================
// UPDATE DOCUMENT
// =======================
func UpdateDocument(c *gin.Context) {
	id := c.Param("id")
	var document models.Document

	if err := config.DB.First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	var payload struct {
		Sender     string `json:"sender"`
		Subject    string `json:"subject"`
		LetterType string `json:"letter_type"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	document.Sender = payload.Sender
	document.Subject = payload.Subject
	document.LetterType = payload.LetterType

	if err := config.DB.Save(&document).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui dokumen"})
		return
	}

	if userRaw, exists := c.Get("user"); exists {
		user := userRaw.(models.User)
		services.CreateActivity(user.ID, user.Name, "update", "Memperbarui dokumen: "+document.FileName)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Dokumen berhasil diperbarui",
		"document": document,
	})
}

// =======================
// DELETE DOCUMENT
// =======================
func DeleteDocument(c *gin.Context) {
	id := c.Param("id")
	var document models.Document

	if err := config.DB.First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	if document.PublicID != "" && document.ResourceType != "" {
		_ = config.DeleteFromCloudinary(document.PublicID, document.ResourceType)
	}

	if err := config.DB.Delete(&document).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus dokumen"})
		return
	}

	if userRaw, exists := c.Get("user"); exists {
		user := userRaw.(models.User)
		services.CreateActivity(user.ID, user.Name, "delete", "Menghapus dokumen: "+document.FileName)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Dokumen berhasil dihapus"})
}

// =======================
// DOWNLOAD DOCUMENT
// =======================
func DownloadDocument(c *gin.Context) {
	id := c.Param("id")
	var document models.Document

	if err := config.DB.First(&document, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	if document.FileURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link file tidak tersedia"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, document.FileURL)
}
