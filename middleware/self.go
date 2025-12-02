package middleware

import (
	"net/http"

	"dinsos_kuburaya/models"

	"github.com/gin-gonic/gin"
)

// Hanya superadmin boleh akses user lain
// admin & staff hanya boleh akses dirinya sendiri
func UserSelfOrSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRaw, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			c.Abort()
			return
		}

		user := userRaw.(models.User)
		targetID := c.Param("id")

		// superadmin bebas
		if user.Role == "superadmin" {
			c.Next()
			return
		}

		// admin & staff hanya boleh update dirinya sendiri
		if user.ID != targetID {
			c.JSON(http.StatusForbidden, gin.H{"message": "Tidak boleh mengakses data user lain"})
			c.Abort()
			return
		}

		c.Next()
	}
}
