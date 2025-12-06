package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/middleware"
	"github.com/ukuvago/angel-platform/internal/models"
	"github.com/ukuvago/angel-platform/internal/services"
)

type TermSheetHandler struct {
	documentService *services.DocumentService
	emailService    *services.EmailService
	authService     *services.AuthService
}

func NewTermSheetHandler(documentService *services.DocumentService, emailService *services.EmailService, authService *services.AuthService) *TermSheetHandler {
	return &TermSheetHandler{
		documentService: documentService,
		emailService:    emailService,
		authService:     authService,
	}
}

// GetTermSheet returns a specific term sheet
func (h *TermSheetHandler) GetTermSheet(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	termSheetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid term sheet ID"})
		return
	}

	db := database.GetDB()

	var termSheet models.TermSheet
	if err := db.Preload("Offer").
		Preload("Offer.Project").
		Preload("Offer.Investor").
		First(&termSheet, "id = ?", termSheetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Term sheet not found"})
		return
	}

	// Verify user has access
	role, _ := middleware.GetUserRole(c)
	if role == models.RoleInvestor && termSheet.Offer.InvestorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	if role == models.RoleDeveloper && termSheet.Offer.Project.DeveloperID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"term_sheet": termSheet})
}

// GetMyTermSheets returns term sheets for the current user
func (h *TermSheetHandler) GetMyTermSheets(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	role, _ := middleware.GetUserRole(c)
	db := database.GetDB()

	var termSheets []models.TermSheet

	if role == models.RoleInvestor {
		if err := db.Joins("JOIN investment_offers ON investment_offers.id = term_sheets.offer_id").
			Where("investment_offers.investor_id = ?", userID).
			Preload("Offer").
			Preload("Offer.Project").
			Order("term_sheets.created_at DESC").
			Find(&termSheets).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch term sheets"})
			return
		}
	} else if role == models.RoleDeveloper {
		if err := db.Joins("JOIN investment_offers ON investment_offers.id = term_sheets.offer_id").
			Joins("JOIN projects ON projects.id = investment_offers.project_id").
			Where("projects.developer_id = ?", userID).
			Preload("Offer").
			Preload("Offer.Project").
			Preload("Offer.Investor").
			Order("term_sheets.created_at DESC").
			Find(&termSheets).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch term sheets"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"term_sheets": termSheets})
}

// SignTermSheetRequest represents term sheet signing input
type SignTermSheetRequest struct {
	SignatureData string `json:"signature_data" binding:"required"`
}

// SignTermSheet signs a term sheet
func (h *TermSheetHandler) SignTermSheet(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	termSheetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid term sheet ID"})
		return
	}

	var req SignTermSheetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	termSheet, err := h.documentService.SignTermSheet(termSheetID, userID, req.SignatureData, c.ClientIP())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If fully signed, send notifications and generate PDF
	if termSheet.Status == models.TermSheetStatusCompleted {
		db := database.GetDB()
		var offer models.InvestmentOffer
		db.Preload("Project").Preload("Investor").First(&offer, "id = ?", termSheet.OfferID)

		var developer models.User
		db.First(&developer, "id = ?", offer.Project.DeveloperID)

		// Generate PDF
		pdfPath, _ := h.documentService.GenerateSAFENotePDF(termSheet, &offer, offer.Investor, &developer, offer.Project)
		termSheet.DocumentPath = pdfPath
		db.Save(termSheet)

		// Send notifications
		go h.emailService.SendTermSheetSignedNotification(offer.Investor, offer.Project)
		go h.emailService.SendTermSheetSignedNotification(&developer, offer.Project)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Term sheet signed successfully",
		"term_sheet": termSheet,
	})
}

// DownloadTermSheet downloads the term sheet PDF
func (h *TermSheetHandler) DownloadTermSheet(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	termSheetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid term sheet ID"})
		return
	}

	db := database.GetDB()

	var termSheet models.TermSheet
	if err := db.Preload("Offer").
		Preload("Offer.Project").
		Preload("Offer.Investor").
		First(&termSheet, "id = ?", termSheetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Term sheet not found"})
		return
	}

	// Verify user has access
	role, _ := middleware.GetUserRole(c)
	if role == models.RoleInvestor && termSheet.Offer.InvestorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var developer models.User
	db.First(&developer, "id = ?", termSheet.Offer.Project.DeveloperID)

	if role == models.RoleDeveloper && developer.ID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Generate fresh PDF
	pdfPath, err := h.documentService.GenerateSAFENotePDF(&termSheet, termSheet.Offer, termSheet.Offer.Investor, &developer, termSheet.Offer.Project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	c.File(pdfPath)
}
