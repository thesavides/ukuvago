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

type ProjectHandler struct {
	storageService *services.StorageService
	paymentService *services.PaymentService
}

func NewProjectHandler(storageService *services.StorageService, paymentService *services.PaymentService) *ProjectHandler {
	return &ProjectHandler{
		storageService: storageService,
		paymentService: paymentService,
	}
}

// ListProjects returns a list of approved projects (public info only)
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	db := database.GetDB()

	category := c.Query("category")
	search := c.Query("search")

	query := db.Where("status = ?", models.ProjectStatusApproved).
		Preload("Category").
		Preload("Images")

	if category != "" {
		query = query.Where("category_id = ?", category)
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title ILIKE ? OR tagline ILIKE ?", searchPattern, searchPattern)
	}

	var projects []models.Project
	if err := query.Order("created_at DESC").Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	// Convert to public info
	var publicProjects []models.ProjectPublicInfo
	for _, p := range projects {
		publicProjects = append(publicProjects, p.ToPublicInfo())
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": publicProjects,
		"total":    len(publicProjects),
	})
}

// GetProject returns full project details (requires NDA + payment)
func (h *ProjectHandler) GetProject(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	db := database.GetDB()

	var project models.Project
	if err := db.Preload("Category").
		Preload("Images").
		Preload("Developer").
		First(&project, "id = ?", projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	userID, exists := middleware.GetUserID(c)
	role, _ := middleware.GetUserRole(c)

	// Developers can view their own projects
	if role == models.RoleDeveloper && project.DeveloperID == userID {
		c.JSON(http.StatusOK, gin.H{"project": project})
		return
	}

	// Admins can view all projects
	if role == models.RoleAdmin {
		c.JSON(http.StatusOK, gin.H{"project": project})
		return
	}

	// For investors: check if already viewed
	if exists && role == models.RoleInvestor {
		if h.paymentService.HasViewedProject(userID, projectID) {
			// Already viewed, show full details
			c.JSON(http.StatusOK, gin.H{"project": project})
			return
		}

		// Use a view credit
		if err := h.paymentService.UseViewCredit(userID, projectID); err != nil {
			// Return public info only
			c.JSON(http.StatusOK, gin.H{
				"project":        project.ToPublicInfo(),
				"full_access":    false,
				"payment_needed": true,
				"error":          err.Error(),
			})
			return
		}

		// Increment view count
		db.Model(&project).Update("view_count", project.ViewCount+1)
	}

	// Check if public-only access
	if !exists || project.Status != models.ProjectStatusApproved {
		c.JSON(http.StatusOK, gin.H{
			"project":     project.ToPublicInfo(),
			"full_access": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"project":     project,
		"full_access": true,
	})
}

// GetCategories returns all project categories
func (h *ProjectHandler) GetCategories(c *gin.Context) {
	db := database.GetDB()

	var categories []models.Category
	if err := db.Order("name").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// CreateProjectRequest represents project creation input
type CreateProjectRequest struct {
	Title         string    `json:"title" binding:"required"`
	Tagline       string    `json:"tagline"`
	CategoryID    uuid.UUID `json:"category_id" binding:"required"`
	Description   string    `json:"description" binding:"required"`
	PitchContent  string    `json:"pitch_content"`
	Problem       string    `json:"problem"`
	Solution      string    `json:"solution"`
	TargetMarket  string    `json:"target_market"`
	BusinessModel string    `json:"business_model"`
	Traction      string    `json:"traction"`
	Team          string    `json:"team"`
	MinInvestment float64   `json:"min_investment" binding:"required"`
	MaxInvestment float64   `json:"max_investment"`
	EquityOffered float64   `json:"equity_offered"`
	ValuationCap  float64   `json:"valuation_cap"`
}

// CreateProject creates a new project (developer only)
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := &models.Project{
		DeveloperID:   userID,
		CategoryID:    req.CategoryID,
		Title:         req.Title,
		Tagline:       req.Tagline,
		Description:   req.Description,
		PitchContent:  req.PitchContent,
		Problem:       req.Problem,
		Solution:      req.Solution,
		TargetMarket:  req.TargetMarket,
		BusinessModel: req.BusinessModel,
		Traction:      req.Traction,
		Team:          req.Team,
		MinInvestment: req.MinInvestment,
		MaxInvestment: req.MaxInvestment,
		EquityOffered: req.EquityOffered,
		ValuationCap:  req.ValuationCap,
		Status:        models.ProjectStatusDraft,
	}

	db := database.GetDB()
	if err := db.Create(project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Project created successfully",
		"project": project,
	})
}

// UpdateProject updates a project (developer only, draft/rejected status)
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	db := database.GetDB()

	var project models.Project
	if err := db.First(&project, "id = ? AND developer_id = ?", projectID, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if project.Status != models.ProjectStatusDraft && project.Status != models.ProjectStatusRejected {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot edit approved or pending projects"})
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project.Title = req.Title
	project.Tagline = req.Tagline
	project.CategoryID = req.CategoryID
	project.Description = req.Description
	project.PitchContent = req.PitchContent
	project.Problem = req.Problem
	project.Solution = req.Solution
	project.TargetMarket = req.TargetMarket
	project.BusinessModel = req.BusinessModel
	project.Traction = req.Traction
	project.Team = req.Team
	project.MinInvestment = req.MinInvestment
	project.MaxInvestment = req.MaxInvestment
	project.EquityOffered = req.EquityOffered
	project.ValuationCap = req.ValuationCap

	if err := db.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Project updated successfully",
		"project": project,
	})
}

// SubmitProject submits a project for review
func (h *ProjectHandler) SubmitProject(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	db := database.GetDB()

	var project models.Project
	if err := db.First(&project, "id = ? AND developer_id = ?", projectID, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if project.Status != models.ProjectStatusDraft && project.Status != models.ProjectStatusRejected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project already submitted or approved"})
		return
	}

	project.Status = models.ProjectStatusPending
	if err := db.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Project submitted for review",
		"project": project,
	})
}

// UploadProjectImage uploads an image for a project
func (h *ProjectHandler) UploadProjectImage(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	db := database.GetDB()

	var project models.Project
	if err := db.First(&project, "id = ? AND developer_id = ?", projectID, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	caption := c.PostForm("caption")
	isPrimary := c.PostForm("is_primary") == "true"

	filePath, fileName, err := h.storageService.SaveProjectImage(projectID, file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Count existing images for display order
	var count int64
	db.Model(&models.ProjectImage{}).Where("project_id = ?", projectID).Count(&count)

	// If this is the first image or marked as primary, update others
	if isPrimary {
		db.Model(&models.ProjectImage{}).Where("project_id = ?", projectID).Update("is_primary", false)
	} else if count == 0 {
		isPrimary = true // First image is always primary
	}

	image := &models.ProjectImage{
		ProjectID:    projectID,
		FilePath:     filePath,
		FileName:     fileName,
		Caption:      caption,
		DisplayOrder: int(count),
		IsPrimary:    isPrimary,
	}

	if err := db.Create(image).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Image uploaded successfully",
		"image":   image,
	})
}

// DeleteProjectImage deletes an image from a project
func (h *ProjectHandler) DeleteProjectImage(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	imageID, err := uuid.Parse(c.Param("imageId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	db := database.GetDB()

	// Verify project ownership
	var project models.Project
	if err := db.First(&project, "id = ? AND developer_id = ?", projectID, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	var image models.ProjectImage
	if err := db.First(&image, "id = ? AND project_id = ?", imageID, projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Delete file
	h.storageService.DeleteProjectImage(image.FilePath)

	// Delete record
	if err := db.Delete(&image).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}

// GetMyProjects returns the developer's projects
func (h *ProjectHandler) GetMyProjects(c *gin.Context) {
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	db := database.GetDB()

	var projects []models.Project
	if err := db.Where("developer_id = ?", userID).
		Preload("Category").
		Preload("Images").
		Order("created_at DESC").
		Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	// Count offers for each project
	type ProjectWithOffers struct {
		models.Project
		PendingOffers int `json:"pending_offers"`
	}

	var result []ProjectWithOffers
	for _, p := range projects {
		var count int64
		db.Model(&models.InvestmentOffer{}).
			Where("project_id = ? AND status = ?", p.ID, models.OfferStatusPending).
			Count(&count)

		result = append(result, ProjectWithOffers{
			Project:       p,
			PendingOffers: int(count),
		})
	}

	c.JSON(http.StatusOK, gin.H{"projects": result})
}
