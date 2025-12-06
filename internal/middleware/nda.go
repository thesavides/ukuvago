package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/models"
)

// RequireNDA ensures the investor has signed an NDA
func RequireNDA() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		role, _ := GetUserRole(c)
		// NDA only required for investors
		if role != models.RoleInvestor {
			c.Next()
			return
		}

		db := database.GetDB()
		var nda models.NDA
		err := db.Where("investor_id = ?", userID).
			Order("signed_at DESC").
			First(&nda).Error

		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "NDA signature required",
				"code":    "NDA_REQUIRED",
				"message": "You must sign the Non-Disclosure Agreement before accessing project details",
			})
			c.Abort()
			return
		}

		if !nda.IsValid() {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "NDA has expired",
				"code":    "NDA_EXPIRED",
				"message": "Your NDA has expired. Please sign a new one to continue",
			})
			c.Abort()
			return
		}

		c.Set("ndaID", nda.ID)
		c.Next()
	}
}

// CheckNDAStatus adds NDA status to context without requiring it
func CheckNDAStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			c.Set("hasNDA", false)
			c.Next()
			return
		}

		db := database.GetDB()
		var nda models.NDA
		err := db.Where("investor_id = ?", userID).
			Order("signed_at DESC").
			First(&nda).Error

		if err != nil || !nda.IsValid() {
			c.Set("hasNDA", false)
		} else {
			c.Set("hasNDA", true)
			c.Set("ndaID", nda.ID)
		}

		c.Next()
	}
}
