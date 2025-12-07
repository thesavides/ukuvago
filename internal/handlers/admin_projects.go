package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/models"
)

// GetAllProjects traverses all projects for admin management
func (h *AdminHandler) GetAllProjects(c *gin.Context) {
	var projects []models.Project
	db := database.GetDB()

	// Fetch all projects regardless of status, include developer and category info
	if err := db.Preload("Developer").Preload("Category").Order("created_at desc").Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"projects": projects})
}
