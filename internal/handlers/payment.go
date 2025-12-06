package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ukuvago/angel-platform/internal/middleware"
	"github.com/ukuvago/angel-platform/internal/services"
)

type PaymentHandler struct {
	paymentService *services.PaymentService
}

func NewPaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

// CreatePaymentIntent creates a new payment intent for viewing projects
func (h *PaymentHandler) CreatePaymentIntent(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	payment, clientSecret, err := h.paymentService.CreatePaymentIntent(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"payment_id":    payment.ID,
		"client_secret": clientSecret,
		"amount":        payment.Amount,
		"currency":      payment.Currency,
		"projects":      payment.ProjectsTotal,
	})
}

// ConfirmPaymentRequest represents payment confirmation input
type ConfirmPaymentRequest struct {
	PaymentID       uuid.UUID `json:"payment_id" binding:"required"`
	StripePaymentID string    `json:"stripe_payment_id"`
	DemoMode        bool      `json:"demo_mode"`
}

// ConfirmPayment confirms a completed payment
func (h *PaymentHandler) ConfirmPayment(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req ConfirmPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var payment *interface{}
	var err error

	if req.DemoMode {
		// Demo mode confirmation
		p, e := h.paymentService.DemoConfirmPayment(req.PaymentID)
		if e != nil {
			err = e
		} else {
			var temp interface{} = p
			payment = &temp
		}
	} else {
		p, e := h.paymentService.ConfirmPayment(req.PaymentID, req.StripePaymentID)
		if e != nil {
			err = e
		} else {
			var temp interface{} = p
			payment = &temp
		}
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = userID // Validate payment belongs to user in production

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment confirmed successfully",
		"payment": payment,
	})
}

// GetPaymentStatus returns the current payment status
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	payment, err := h.paymentService.GetActivePayment(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"has_active_payment":  false,
			"projects_remaining":  0,
			"message":             "No active payment. Please make a payment to view projects.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"has_active_payment":  true,
		"payment":             payment.ToResponse(),
		"projects_remaining":  payment.ProjectsRemaining,
		"projects_total":      payment.ProjectsTotal,
	})
}

// GetPaymentHistory returns the user's payment history
func (h *PaymentHandler) GetPaymentHistory(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	payments, err := h.paymentService.GetPaymentHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve payment history"})
		return
	}

	// Convert to response format
	var response []interface{}
	for _, p := range payments {
		response = append(response, p.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{
		"payments": response,
	})
}

// GetViewedProjects returns projects the user has viewed
func (h *PaymentHandler) GetViewedProjects(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	views, err := h.paymentService.GetViewedProjects(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve viewed projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"views": views,
	})
}
