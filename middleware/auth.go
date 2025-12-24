package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"

	"dinsos_kuburaya/config"
	"dinsos_kuburaya/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 1. Ambil header Authorization
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		// 2. HASH token yang diterima untuk dicocokkan dengan database
		hash := sha256.Sum256([]byte(tokenString))
		hashedToken := hex.EncodeToString(hash[:])

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "default_secret"
		}

		// 3. Parse token untuk validasi JWT
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil {
			config.DB.Where("jwt_token = ?", hashedToken).Delete(&models.SecretToken{})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token invalid"})
			c.Abort()
			return
		}

		if !token.Valid {
			config.DB.Where("jwt_token = ?", hashedToken).Delete(&models.SecretToken{})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token invalid"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			config.DB.Where("jwt_token = ?", hashedToken).Delete(&models.SecretToken{})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		// 4. Validasi exp claim
		exp, ok := claims["exp"].(float64)
		if !ok {
			config.DB.Where("jwt_token = ?", hashedToken).Delete(&models.SecretToken{})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid exp claim"})
			c.Abort()
			return
		}

		if time.Now().Unix() > int64(exp) {
			config.DB.Where("jwt_token = ?", hashedToken).Delete(&models.SecretToken{})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			c.Abort()
			return
		}

		// 5. Validasi user_id claim
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			config.DB.Where("jwt_token = ?", hashedToken).Delete(&models.SecretToken{})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id claim"})
			c.Abort()
			return
		}

		// 6. Cek hashed token di database
		var st models.SecretToken
		if err := config.DB.Preload("User").Where("jwt_token = ?", hashedToken).First(&st).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			c.Abort()
			return
		}

		// 7. Cek expiration di database
		if time.Now().After(st.ExpiresAt) {
			config.DB.Where("jwt_token = ?", hashedToken).Delete(&models.SecretToken{})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			c.Abort()
			return
		}

		// 8. Cek user masih ada
		if st.User.ID == "" {
			config.DB.Where("jwt_token = ?", hashedToken).Delete(&models.SecretToken{})
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			c.Abort()
			return
		}

		c.Set("user", st.User)
		c.Next()
	}
}
