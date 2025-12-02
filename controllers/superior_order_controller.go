package controllers

import (
	"net/http"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"dinsos_kuburaya/services"

	"github.com/gin-gonic/gin"
)

// ======================================================
// CREATE SuperiorOrder
// ======================================================
func CreateSuperiorOrder(c *gin.Context) {
	var input struct {
		DocumentID string   `json:"document_id" binding:"required"`
		UserIDs    []string `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Ambil dokumennya dulu
	var doc models.Document
	if err := config.DB.First(&doc, "id = ?", input.DocumentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	var created []models.SuperiorOrder

	// Loop setiap user yang ditugaskan
	for _, userID := range input.UserIDs {

		order := models.SuperiorOrder{
			DocumentID: input.DocumentID,
			UserID:     userID,
		}

		if err := config.DB.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create record: " + err.Error()})
			return
		}

		created = append(created, order)

		// 🔥 Kirim notifikasi khusus user tersebut
		message := "Dokumen masuk kepada Anda: " + doc.FileName
		link := "/documents/" + doc.ID

		services.NotifySpecificUser(userID, message, link)
	}

	if userRaw, exists := c.Get("user"); exists {
		user := userRaw.(models.User)
		services.CreateActivity(
			user.ID,
			user.Name,
			"create",
			"Menambahkan penugasan: "+doc.FileName,
		)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Superior orders created",
		"data":    created,
	})
}

// ======================================================
// GET ALL SuperiorOrders (grouped by document_id)
// ======================================================
func GetSuperiorOrders(c *gin.Context) {
	var orders []models.SuperiorOrder
	if err := config.DB.Preload("User").Preload("Document").Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch records: " + err.Error()})
		return
	}

	// Map untuk menampung hasil akhir
	type UserInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	type DocumentInfo struct {
		DocumentID string     `json:"document_id"`
		Sender     string     `json:"sender"`
		Subject    string     `json:"subject"`
		Users      []UserInfo `json:"users"`
	}

	grouped := make(map[string]*DocumentInfo)

	for _, o := range orders {
		if _, exists := grouped[o.DocumentID]; !exists {
			grouped[o.DocumentID] = &DocumentInfo{
				DocumentID: o.DocumentID,
				Sender:     o.Document.Sender,
				Subject:    o.Document.Subject,
				Users:      []UserInfo{},
			}
		}
		grouped[o.DocumentID].Users = append(
			grouped[o.DocumentID].Users,
			UserInfo{ID: o.UserID, Name: o.User.Name},
		)
	}

	// Ubah map menjadi slice untuk response
	var result []DocumentInfo
	for _, v := range grouped {
		result = append(result, *v)
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// ======================================================
// GET SuperiorOrders by document_id
// ======================================================
func GetSuperiorOrdersByDocument(c *gin.Context) {
	documentID := c.Param("document_id")
	var orders []models.SuperiorOrder
	if err := config.DB.Preload("User").Preload("Document").Where("document_id = ?", documentID).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch records: " + err.Error()})
		return
	}

	if len(orders) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No records found for this document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"document_id": documentID, "user_ids": orders})
}

// ======================================================
// UPDATE SuperiorOrder by document_id
// ======================================================
func UpdateSuperiorOrder(c *gin.Context) {
	documentID := c.Param("id")

	var input struct {
		UserIDs []string `json:"user_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document ID is required"})
		return
	}

	// 🔥 Ambil dokumen untuk nama file
	var doc models.Document
	if err := config.DB.First(&doc, "id = ?", documentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen tidak ditemukan"})
		return
	}

	// 🗑 Hapus semua user lama
	if err := config.DB.Where("document_id = ?", documentID).Delete(&models.SuperiorOrder{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete old records: " + err.Error()})
		return
	}

	// ➕ Tambah user baru + kirim notifikasi
	var created []models.SuperiorOrder

	for _, userID := range input.UserIDs {
		if userID == "" {
			continue
		}

		order := models.SuperiorOrder{
			DocumentID: documentID,
			UserID:     userID,
		}

		if err := config.DB.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create record: " + err.Error()})
			return
		}

		created = append(created, order)

		// 🔔 KIRIM NOTIFIKASI PER USER
		message := "Dokumen masuk kepada Anda: " + doc.FileName
		link := "/documents/" + doc.ID
		services.NotifySpecificUser(userID, message, link)
	}

	if userRaw, exists := c.Get("user"); exists {
		user := userRaw.(models.User)
		services.CreateActivity(
			user.ID,
			user.Name,
			"update",
			"Memperbarui penugasan: "+doc.FileName,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "SuperiorOrder updated",
		"data":    created,
	})
}

// ======================================================
// DELETE SuperiorOrder by document_id
// ======================================================
func DeleteSuperiorOrder(c *gin.Context) {
	documentID := c.Param("id") // Diubah dari "document_id" menjadi "id"

	// Validasi document_id tidak kosong
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Document ID is required"})
		return
	}

	// Cek dulu apakah ada data yang akan dihapus
	var count int64
	if err := config.DB.Model(&models.SuperiorOrder{}).Where("document_id = ?", documentID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check records: " + err.Error()})
		return
	}

	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No records found for this document"})
		return
	}

	if err := config.DB.Where("document_id = ?", documentID).Delete(&models.SuperiorOrder{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete records: " + err.Error()})
		return
	}

	var doc models.Document
	config.DB.First(&doc, "id = ?", documentID)

	if userRaw, exists := c.Get("user"); exists {
		user := userRaw.(models.User)
		services.CreateActivity(
			user.ID,
			user.Name,
			"delete",
			"Menghapus penugasan: "+doc.FileName,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "All SuperiorOrders for document deleted",
		"document_id":   documentID,
		"deleted_count": count,
	})
}
