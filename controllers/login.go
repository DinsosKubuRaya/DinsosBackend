package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var input LoginRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Input tidak valid"})
		return
	}

	db := config.DB

	var user models.User
	if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Username atau password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Username atau password salah"})
		return
	}

	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "default_secret"
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(14 * 24 * time.Hour).Unix(),
		"role":    user.Role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal membuat token"})
		return
	}

	// Hash token
	hash := sha256.Sum256([]byte(signedToken))
	hashedToken := hex.EncodeToString(hash[:])

	device := c.GetHeader("X-Device")
	if device == "" {
		device = "unknown"
	}

	// Clean old token
	db.Where("expires_at < ?", time.Now()).Delete(&models.SecretToken{})
	db.Where("user_id = ? AND device = ?", user.ID, device).Delete(&models.SecretToken{})

	// Limit 2 tokens
	var userTokens []models.SecretToken
	db.Where("user_id = ?", user.ID).Order("created_at DESC").Find(&userTokens)
	if len(userTokens) >= 2 {
		oldest := userTokens[len(userTokens)-1]
		db.Delete(&oldest)
	}

	secretToken := models.SecretToken{
		JwtToken:  hashedToken,
		UserID:    user.ID,
		Device:    device,
		ExpiresAt: time.Now().Add(14 * 24 * time.Hour),
	}

	if err := db.Create(&secretToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal menyimpan token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Login berhasil",
		"token":    signedToken,
		"token_id": secretToken.ID,
		"user": gin.H{
			"id":        user.ID,
			"name":      user.Name,
			"username":  user.Username,
			"role":      user.Role,
			"photo_url": user.PhotoURL,
		},
	})
}
