package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/middleware"
	"github.com/ukuvago/angel-platform/internal/models"
	"github.com/ukuvago/angel-platform/internal/services"
)

type OfferHandler struct {
	emailService    *services.EmailService
	documentService *services.DocumentService
	authService     *services.AuthService
}

func NewOfferHandler(emailService *services.EmailService, documentService *services.DocumentService, authService *services.AuthService) *OfferHandler {
	return &OfferHandler{
		emailService:    emailService,
		documentService: documentService,
		authService:     authService,
	}
}

// CreateOfferRequest represents offer creation input
type CreateOfferRequest struct {
	ProjectID     uuid.UUID `json:"project_id" binding:"required"`
	OfferAmount   float64   `json:"offer_amount" binding:"required,gt=0"`
	EquityRequest float64   `json:"equity_request"`
	TermsNotes    string    `json:"terms_notes"`
}

// CreateOffer creates a new investment offer (investor only)
func (h *OfferHandler) CreateOffer(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req CreateOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	// Verify project exists and is approved
	var project models.Project
	if err := db.Preload("Developer").First(&project, "id = ? AND status = ?", req.ProjectID, models.ProjectStatusApproved).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not available"})
		return
	}

	// Check minimum investment
	if req.OfferAmount < project.MinInvestment {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Offer below minimum investment",
			"min_investment": project.MinInvestment,
		})
		return
	}

	// Check for existing pending offer
	var existingOffer models.InvestmentOffer
	err := db.Where("investor_id = ? AND project_id = ? AND status = ?",
		userID, req.ProjectID, models.OfferStatusPending).First(&existingOffer).Error
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You already have a pending offer for this project"})
		return
	}

	expiresAt := time.Now().AddDate(0, 0, 30)
	offer := &models.InvestmentOffer{
		InvestorID:    userID,
		ProjectID:     req.ProjectID,
		OfferAmount:   req.OfferAmount,
		EquityRequest: req.EquityRequest,
		TermsNotes:    req.TermsNotes,
		Status:        models.OfferStatusPending,
		ExpiresAt:     &expiresAt,
	}

	if err := db.Create(offer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create offer"})
		return
	}

	// Send notification to developer
	investor, _ := h.authService.GetUserByID(userID)
	if project.Developer != nil && investor != nil {
		go h.emailService.SendOfferNotification(project.Developer, investor, offer, &project)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Offer submitted successfully",
		"offer":   offer,
	})
}

// GetMyOffers returns offers made by the investor or received by the developer
func (h *OfferHandler) GetMyOffers(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	role, _ := middleware.GetUserRole(c)
	db := database.GetDB()

	var offers []models.InvestmentOffer

	if role == models.RoleInvestor {
		// Get offers made by investor
		if err := db.Where("investor_id = ?", userID).
			Preload("Project").
			Preload("Project.Category").
			Preload("TermSheet").
			Order("created_at DESC").
			Find(&offers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch offers"})
			return
		}
	} else if role == models.RoleDeveloper {
		// Get offers received on developer's projects
		if err := db.Joins("JOIN projects ON projects.id = investment_offers.project_id").
			Where("projects.developer_id = ?", userID).
			Preload("Project").
			Preload("Investor").
			Preload("TermSheet").
			Order("investment_offers.created_at DESC").
			Find(&offers).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch offers"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"offers": offers})
}

// GetOffer returns a specific offer
func (h *OfferHandler) GetOffer(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	offerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offer ID"})
		return
	}

	db := database.GetDB()

	var offer models.InvestmentOffer
	if err := db.Preload("Project").
		Preload("Investor").
		Preload("TermSheet").
		First(&offer, "id = ?", offerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Offer not found"})
		return
	}

	// Verify user has access
	role, _ := middleware.GetUserRole(c)
	if role == models.RoleInvestor && offer.InvestorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	if role == models.RoleDeveloper && offer.Project.DeveloperID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"offer": offer})
}

// RespondOfferRequest represents offer response input
type RespondOfferRequest struct {
	Action        string  `json:"action" binding:"required,oneof=accept reject"`
	ResponseNotes string  `json:"response_notes"`
	ValuationCap  float64 `json:"valuation_cap"`
	DiscountRate  float64 `json:"discount_rate"`
}

// RespondToOffer accepts or rejects an offer (developer only)
func (h *OfferHandler) RespondToOffer(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	offerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offer ID"})
		return
	}

	var req RespondOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	var offer models.InvestmentOffer
	if err := db.Preload("Project").Preload("Investor").First(&offer, "id = ?", offerID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Offer not found"})
		return
	}

	// Verify developer owns the project
	if offer.Project.DeveloperID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if !offer.CanRespond() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot respond to this offer"})
		return
	}

	now := time.Now()
	offer.RespondedAt = &now
	offer.ResponseNotes = req.ResponseNotes

	accepted := req.Action == "accept"
	if accepted {
		offer.Status = models.OfferStatusAccepted

		// Create term sheet
		termSheet := &models.TermSheet{
			OfferID:          offer.ID,
			InvestmentAmount: offer.OfferAmount,
			ValuationCap:     req.ValuationCap,
			DiscountRate:     req.DiscountRate,
			ProRataRights:    true,
			Status:           models.TermSheetStatusDraft,
		}

		if termSheet.ValuationCap == 0 {
			termSheet.ValuationCap = offer.Project.ValuationCap
		}
		if termSheet.DiscountRate == 0 {
			termSheet.DiscountRate = 20.0 // Default 20%
		}

		if err := db.Create(termSheet).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create term sheet"})
			return
		}

		offer.TermSheet = termSheet
	} else {
		offer.Status = models.OfferStatusRejected
	}

	if err := db.Save(&offer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update offer"})
		return
	}

	// Send notification to investor
	if offer.Investor != nil {
		go h.emailService.SendOfferResponseNotification(offer.Investor, &offer, offer.Project, accepted)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Offer " + req.Action + "ed successfully",
		"offer":   offer,
	})
}

// WithdrawOffer allows an investor to withdraw their offer
func (h *OfferHandler) WithdrawOffer(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	offerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offer ID"})
		return
	}

	db := database.GetDB()

	var offer models.InvestmentOffer
	if err := db.First(&offer, "id = ? AND investor_id = ?", offerID, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Offer not found"})
		return
	}

	if offer.Status != models.OfferStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only withdraw pending offers"})
		return
	}

	offer.Status = models.OfferStatusWithdrawn
	if err := db.Save(&offer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to withdraw offer"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Offer withdrawn successfully",
		"offer":   offer,
	})
}
