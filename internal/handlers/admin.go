package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/models"
	"github.com/ukuvago/angel-platform/internal/services"
)

type AdminHandler struct {
	emailService *services.EmailService
	authService  *services.AuthService
}

func NewAdminHandler(emailService *services.EmailService, authService *services.AuthService) *AdminHandler {
	return &AdminHandler{
		emailService: emailService,
		authService:  authService,
	}
}

// GetDashboardStats returns platform statistics
func (h *AdminHandler) GetDashboardStats(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		TotalUsers       int64 `json:"total_users"`
		TotalInvestors   int64 `json:"total_investors"`
		TotalDevelopers  int64 `json:"total_developers"`
		TotalProjects    int64 `json:"total_projects"`
		ApprovedProjects int64 `json:"approved_projects"`
		PendingProjects  int64 `json:"pending_projects"`
		TotalOffers      int64 `json:"total_offers"`
		AcceptedOffers   int64 `json:"accepted_offers"`
		TotalPayments    int64 `json:"total_payments"`
		TotalRevenue     int64 `json:"total_revenue"`
	}

	db.Model(&models.User{}).Count(&stats.TotalUsers)
	db.Model(&models.User{}).Where("role = ?", models.RoleInvestor).Count(&stats.TotalInvestors)
	db.Model(&models.User{}).Where("role = ?", models.RoleDeveloper).Count(&stats.TotalDevelopers)
	db.Model(&models.Project{}).Count(&stats.TotalProjects)
	db.Model(&models.Project{}).Where("status = ?", models.ProjectStatusApproved).Count(&stats.ApprovedProjects)
	db.Model(&models.Project{}).Where("status = ?", models.ProjectStatusPending).Count(&stats.PendingProjects)
	db.Model(&models.InvestmentOffer{}).Count(&stats.TotalOffers)
	db.Model(&models.InvestmentOffer{}).Where("status = ?", models.OfferStatusAccepted).Count(&stats.AcceptedOffers)
	db.Model(&models.Payment{}).Where("status = ?", models.PaymentStatusCompleted).Count(&stats.TotalPayments)

	// Calculate total revenue
	var revenue struct {
		Total int64
	}
	db.Model(&models.Payment{}).
		Where("status = ?", models.PaymentStatusCompleted).
		Select("COALESCE(SUM(amount), 0) as total").
		Scan(&revenue)
	stats.TotalRevenue = revenue.Total

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// ListAllUsers returns all users with pagination
func (h *AdminHandler) ListAllUsers(c *gin.Context) {
	db := database.GetDB()

	role := c.Query("role")

	query := db.Model(&models.User{})
	if role != "" {
		query = query.Where("role = ?", role)
	}

	var users []models.User
	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Convert to response format
	var response []models.UserResponse
	for _, u := range users {
		response = append(response, u.ToResponse())
	}

	c.JSON(http.StatusOK, gin.H{"users": response})
}

// ListAllProjects returns all projects with pagination
func (h *AdminHandler) ListAllProjects(c *gin.Context) {
	db := database.GetDB()

	status := c.Query("status")

	query := db.Model(&models.Project{}).
		Preload("Developer").
		Preload("Category").
		Preload("Images")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var projects []models.Project
	if err := query.Order("created_at DESC").Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

// GetPendingProjects returns projects awaiting approval
func (h *AdminHandler) GetPendingProjects(c *gin.Context) {
	db := database.GetDB()

	var projects []models.Project
	if err := db.Where("status = ?", models.ProjectStatusPending).
		Preload("Developer").
		Preload("Category").
		Preload("Images").
		Order("created_at ASC").
		Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

// ApproveProjectRequest represents project approval input
type ApproveProjectRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

// ApproveProject approves or rejects a project
func (h *AdminHandler) ApproveProject(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var req ApproveProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	var project models.Project
	if err := db.Preload("Developer").First(&project, "id = ?", projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if project.Status != models.ProjectStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project is not pending review"})
		return
	}

	now := time.Now()

	if req.Approved {
		project.Status = models.ProjectStatusApproved
		project.ApprovedAt = &now
		project.RejectionReason = ""
	} else {
		if req.Reason == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rejection reason is required"})
			return
		}
		project.Status = models.ProjectStatusRejected
		project.RejectionReason = req.Reason
	}

	if err := db.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	// Send notification to developer
	if project.Developer != nil {
		go h.emailService.SendProjectApprovalNotification(project.Developer, &project, req.Approved)
	}

	status := "approved"
	if !req.Approved {
		status = "rejected"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Project " + status,
		"project": project,
	})
}

// ListAllOffers returns all investment offers
func (h *AdminHandler) ListAllOffers(c *gin.Context) {
	db := database.GetDB()

	var offers []models.InvestmentOffer
	if err := db.Preload("Project").
		Preload("Investor").
		Preload("TermSheet").
		Order("created_at DESC").
		Find(&offers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch offers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"offers": offers})
}

// ListAllPayments returns all payments
func (h *AdminHandler) ListAllPayments(c *gin.Context) {
	db := database.GetDB()

	var payments []models.Payment
	if err := db.Preload("Investor").
		Order("created_at DESC").
		Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

// CreateCategory creates a new category
type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func (h *AdminHandler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := &models.Category{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
	}

	db := database.GetDB()
	if err := db.Create(category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Category created successfully",
		"category": category,
	})
}

// UpdateCategory updates a category
func (h *AdminHandler) UpdateCategory(c *gin.Context) {
	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()

	var category models.Category
	if err := db.First(&category, "id = ?", categoryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	category.Name = req.Name
	category.Description = req.Description
	category.Icon = req.Icon

	if err := db.Save(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Category updated successfully",
		"category": category,
	})
}

// DeleteCategory deletes a category
func (h *AdminHandler) DeleteCategory(c *gin.Context) {
	categoryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	db := database.GetDB()

	// Check if category has projects
	var count int64
	db.Model(&models.Project{}).Where("category_id = ?", categoryID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete category with existing projects"})
		return
	}

	if err := db.Delete(&models.Category{}, "id = ?", categoryID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}
