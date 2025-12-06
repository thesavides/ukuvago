package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/middleware"
	"github.com/ukuvago/angel-platform/internal/models"
	"github.com/ukuvago/angel-platform/internal/services"
)

type NDAHandler struct {
	authService     *services.AuthService
	documentService *services.DocumentService
}

func NewNDAHandler(authService *services.AuthService, documentService *services.DocumentService) *NDAHandler {
	return &NDAHandler{
		authService:     authService,
		documentService: documentService,
	}
}

// GetNDATemplate returns the NDA template content
func (h *NDAHandler) GetNDATemplate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"template": models.NDATemplateContent,
		"version":  "1.0",
	})
}

// GetNDAStatus returns the current user's NDA status
func (h *NDAHandler) GetNDAStatus(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	db := database.GetDB()
	var nda models.NDA
	err := db.Where("investor_id = ?", userID).
		Order("signed_at DESC").
		First(&nda).Error

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"signed":  false,
			"message": "NDA not signed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"signed":     true,
		"valid":      nda.IsValid(),
		"signed_at":  nda.SignedAt,
		"expires_at": nda.ExpiresAt,
		"version":    nda.Version,
	})
}

// SignNDARequest represents NDA signing input
type SignNDARequest struct {
	SignatureData string `json:"signature_data" binding:"required"` // Base64 encoded signature image
	SignedName    string `json:"signed_name" binding:"required"`
	Agreed        bool   `json:"agreed" binding:"required"`
}

// SignNDA handles NDA signing
func (h *NDAHandler) SignNDA(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req SignNDARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.Agreed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You must agree to the NDA terms"})
		return
	}

	// Get user
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if NDA already signed
	db := database.GetDB()
	var existingNDA models.NDA
	err = db.Where("investor_id = ?", userID).First(&existingNDA).Error
	if err == nil && existingNDA.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already signed a valid NDA"})
		return
	}

	// Create hash of NDA content for legal purposes
	hash := sha256.Sum256([]byte(models.NDATemplateContent))
	documentHash := hex.EncodeToString(hash[:])

	// Set expiration to 2 years from now
	expiresAt := time.Now().AddDate(2, 0, 0)

	nda := &models.NDA{
		InvestorID:    userID,
		SignatureData: req.SignatureData,
		SignedName:    req.SignedName,
		IPAddress:     c.ClientIP(),
		UserAgent:     c.GetHeader("User-Agent"),
		SignedAt:      time.Now(),
		ExpiresAt:     &expiresAt,
		Version:       "1.0",
		DocumentHash:  documentHash,
	}

	if err := db.Create(nda).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save NDA"})
		return
	}

	// Generate PDF
	pdfPath, err := h.documentService.GenerateNDAPDF(nda, user)
	if err != nil {
		// Log error but don't fail the request
		// The NDA is still valid even without the PDF
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "NDA signed successfully",
		"signed_at":  nda.SignedAt,
		"expires_at": nda.ExpiresAt,
		"pdf_path":   pdfPath,
	})
}

// DownloadNDA downloads the signed NDA PDF
func (h *NDAHandler) DownloadNDA(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	db := database.GetDB()
	var nda models.NDA
	err := db.Where("investor_id = ?", userID).
		Order("signed_at DESC").
		First(&nda).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No signed NDA found"})
		return
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Generate fresh PDF
	pdfPath, err := h.documentService.GenerateNDAPDF(&nda, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	c.File(pdfPath)
}
