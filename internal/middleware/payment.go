package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ukuvago/angel-platform/internal/models"
	"github.com/ukuvago/angel-platform/internal/services"
)

// RequirePayment ensures the investor has an active payment with remaining views
func RequirePayment(paymentService *services.PaymentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		role, _ := GetUserRole(c)
		// Payment only required for investors
		if role != models.RoleInvestor {
			c.Next()
			return
		}

		payment, err := paymentService.GetActivePayment(userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Payment required",
				"code":    "PAYMENT_REQUIRED",
				"message": "You must pay the viewing fee to access full project details",
			})
			c.Abort()
			return
		}

		if !payment.CanViewMore() {
			c.JSON(http.StatusForbidden, gin.H{
				"error":             "No remaining views",
				"code":              "NO_VIEWS_REMAINING",
				"message":           "You have used all your project views. Please make another payment to view more projects",
				"projects_remaining": 0,
			})
			c.Abort()
			return
		}

		c.Set("paymentID", payment.ID)
		c.Set("projectsRemaining", payment.ProjectsRemaining)
		c.Next()
	}
}

// CheckPaymentStatus adds payment status to context without requiring it
func CheckPaymentStatus(paymentService *services.PaymentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			c.Set("hasPaid", false)
			c.Set("projectsRemaining", 0)
			c.Next()
			return
		}

		payment, err := paymentService.GetActivePayment(userID)
		if err != nil || !payment.CanViewMore() {
			c.Set("hasPaid", false)
			c.Set("projectsRemaining", 0)
		} else {
			c.Set("hasPaid", true)
			c.Set("paymentID", payment.ID)
			c.Set("projectsRemaining", payment.ProjectsRemaining)
		}

		c.Next()
	}
}
