package controllers

import (
	"net/http"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"
	"dinsos_kuburaya/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var allowedRoles = map[string]bool{
	"superadmin": true,
	"admin":      true,
	"staff":      true,
}

type StorePushTokenRequest struct {
	Token string `json:"token"`
}

func hashPassword(pass string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	return string(hashed), err
}

func CreateUserWithRole(c *gin.Context, role string) {
	if !allowedRoles[role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role tidak valid"})
		return
	}

	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.ID = uuid.NewString()
	input.Role = role

	// Hash Password
	hashed, err := hashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi password"})
		return
	}
	input.Password = hashed

	if err := config.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat user"})
		return
	}

	currentUser := c.MustGet("user").(models.User)
	services.CreateActivity(
		currentUser.ID,
		currentUser.Name,
		"create",
		"Menambahkan user baru: "+input.Name+" sebagai "+role,
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "User berhasil dibuat",
		"user":    input,
	})
}

/*
|--------------------------------------------------------------------------
| WRAPPER: Create Superadmin, Admin, Staff
|--------------------------------------------------------------------------
*/
func CreateSuperAdmin(c *gin.Context) { CreateUserWithRole(c, "superadmin") }
func CreateAdmin(c *gin.Context)      { CreateUserWithRole(c, "admin") }
func CreateStaff(c *gin.Context)      { CreateUserWithRole(c, "staff") }

/*
|--------------------------------------------------------------------------
| READ ALL USER
|--------------------------------------------------------------------------
*/
func GetUsers(c *gin.Context) {
	var users []models.User

	// Gunakan Find dengan kondisi yang lebih spesifik
	result := config.DB.Select("id", "name", "username", "role", "created_at", "updated_at", "photo_url").
		Order("created_at DESC").
		Find(&users)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data users: " + result.Error.Error()})
		return
	}

	// Jika tidak ada data, kembalikan array kosong
	c.JSON(http.StatusOK, users)
}

/*
|--------------------------------------------------------------------------
| READ BY ID
|--------------------------------------------------------------------------
*/
func GetUserByID(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	if err := config.DB.Select("id", "name", "username", "role", "created_at", "updated_at").
		Where("id = ?", id).First(&user).Error; err != nil {

		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, user)
}

/*
|--------------------------------------------------------------------------
| GET USERS FOR FILTER (Khusus untuk filter document staff)
|--------------------------------------------------------------------------
*/
func GetUsersForFilter(c *gin.Context) {
	var users []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	// Hanya ambil id dan name user dengan role staff
	result := config.DB.Model(&models.User{}).
		Select("id", "name").
		Where("role IN ?", []string{"staff", "admin", "superadmin"}). // Sesuaikan dengan role yang diinginkan
		Order("name ASC").
		Find(&users)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data users: " + result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

/*
|--------------------------------------------------------------------------
| GetMe
|--------------------------------------------------------------------------
*/

func GetMe(c *gin.Context) {
	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User tidak terautentikasi",
		})
		return
	}

	user := userRaw.(models.User)

	// Response yang konsisten untuk frontend
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        user.ID,
			"name":      user.Name,
			"username":  user.Username,
			"role":      user.Role,
			"photo_url": user.PhotoURL,
		},
	})
}

/*
|--------------------------------------------------------------------------
| UPDATE USER
|--------------------------------------------------------------------------
*/
func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User

	// Cek apakah user ada
	if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// Ambil semua form-data
	oldPassword := c.PostForm("old_password")
	newPassword := c.PostForm("new_password")

	var input struct {
		Name     string `form:"name"`
		Username string `form:"username"`
		Role     string `form:"role"`
	}

	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid: " + err.Error()})
		return
	}

	updates := map[string]interface{}{}

	/*
		|----------------------------------------
		| Update Nama
		|----------------------------------------
	*/
	if input.Name != "" {
		updates["name"] = input.Name
	}

	/*
		|----------------------------------------
		| Update Username
		|----------------------------------------
	*/
	if input.Username != "" {
		var count int64
		config.DB.Model(&models.User{}).
			Where("username = ? AND id != ?", input.Username, id).
			Count(&count)

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username sudah digunakan"})
			return
		}

		updates["username"] = input.Username
	}

	/*
		|----------------------------------------
		| Update Password Dengan Validasi
		|----------------------------------------
	*/
	if oldPassword != "" || newPassword != "" {

		// Harus isi dua-duanya
		if oldPassword == "" || newPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password lama dan baru harus diisi"})
			return
		}

		// Cek password lama benar atau tidak
		err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Password lama salah"})
			return
		}

		// Validasi panjang minimal password baru
		if len(newPassword) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password baru minimal 6 karakter"})
			return
		}

		// Hash password baru
		hashed, err := hashPassword(newPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi password"})
			return
		}

		updates["password"] = hashed
	}

	/*
		|----------------------------------------
		| Update Foto User
		|----------------------------------------
	*/
	file, err := c.FormFile("photo")
	if err == nil {
		if file.Size > 5<<20 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ukuran file maksimal 5MB"})
			return
		}

		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuka file"})
			return
		}
		defer f.Close()

		uploadRes, err := config.UploadToCloudinary(f, file.Filename, "users", "image")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload foto: " + err.Error()})
			return
		}

		// Hapus foto lama
		if user.PhotoID != nil && *user.PhotoID != "" {
			config.DeleteFromCloudinary(*user.PhotoID, "image")
		}

		updates["photo_url"] = uploadRes.SecureURL
		updates["photo_id"] = uploadRes.PublicID
	}

	/*
		|----------------------------------------
		| Simpan Perubahan
		|----------------------------------------
	*/
	if len(updates) > 0 {
		if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update user"})
			return
		}
	}

	// Ambil data terbaru
	config.DB.Where("id = ?", id).First(&user)

	// ========== ACTIVITY LOG ==========
	currentUser := c.MustGet("user").(models.User)
	services.CreateActivity(
		currentUser.ID,
		currentUser.Name,
		"update",
		"Mengupdate user: "+user.Name,
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "User berhasil diperbarui",
		"user":    user,
	})
}

/*
|--------------------------------------------------------------------------
| DELETE USER
|--------------------------------------------------------------------------
*/
func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// Ambil user yang akan dihapus untuk log
	var user models.User
	if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		return
	}

	// Hapus user
	if err := config.DB.Delete(&models.User{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus user"})
		return
	}

	// ========== ACTIVITY LOG ==========
	currentUser := c.MustGet("user").(models.User)
	services.CreateActivity(
		currentUser.ID,
		currentUser.Name,
		"delete",
		"Menghapus user: "+user.Name,
	)

	c.JSON(http.StatusOK, gin.H{"message": "User berhasil dihapus"})
}

func StorePushToken(c *gin.Context) {
	userRaw, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user := userRaw.(models.User)

	var req StorePushTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}

	// Simpan push token
	config.DB.Model(&models.User{}).
		Where("id = ?", user.ID).
		Update("push_token", req.Token)

	c.JSON(http.StatusOK, gin.H{
		"message": "Push token stored successfully",
	})
}
