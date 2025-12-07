package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ukuvago/angel-platform/internal/config"
)

type StorageService struct {
	config *config.Config
}

func NewStorageService(cfg *config.Config) *StorageService {
	// Ensure upload directory exists
	os.MkdirAll(cfg.UploadDir, 0755)
	os.MkdirAll(filepath.Join(cfg.UploadDir, "projects"), 0755)
	os.MkdirAll(filepath.Join(cfg.UploadDir, "documents"), 0755)

	return &StorageService{config: cfg}
}

// AllowedImageExtensions lists valid image extensions
var AllowedImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

// MaxImageSize is the maximum allowed image size (5MB)
const MaxImageSize = 5 * 1024 * 1024

// SaveProjectImage saves an uploaded project image
func (s *StorageService) SaveProjectImage(projectID uuid.UUID, file *multipart.FileHeader) (string, string, error) {
	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !AllowedImageExtensions[ext] {
		return "", "", fmt.Errorf("invalid file type: %s. Allowed: jpg, jpeg, png, gif, webp", ext)
	}

	// Validate file size
	if file.Size > MaxImageSize {
		return "", "", fmt.Errorf("file too large. Maximum size is 5MB")
	}

	// Create project directory
	projectDir := filepath.Join(s.config.UploadDir, "projects", projectID.String())
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", "", err
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s_%d%s", uuid.New().String()[:8], time.Now().Unix(), ext)
	filePath := filepath.Join(projectDir, filename)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return "", "", err
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", "", err
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		return "", "", err
	}

	// Return relative path for storage in database
	relativePath := filepath.Join("projects", projectID.String(), filename)
	return relativePath, file.Filename, nil
}

// SavePitchDeck saves an uploaded PDF pitch deck
func (s *StorageService) SavePitchDeck(projectID uuid.UUID, file *multipart.FileHeader) (string, error) {
	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".pdf" {
		return "", fmt.Errorf("invalid file type: %s. Only PDF allowed", ext)
	}

	// Validate file size (10MB limit for PDFs)
	if file.Size > 10*1024*1024 {
		return "", fmt.Errorf("file too large. Maximum size is 10MB")
	}

	// Create project directory
	projectDir := filepath.Join(s.config.UploadDir, "projects", projectID.String())
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return "", err
	}

	// Generate unique filename
	filename := fmt.Sprintf("deck_%s_%d%s", uuid.New().String()[:8], time.Now().Unix(), ext)
	filePath := filepath.Join(projectDir, filename)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	// Return relative path
	return filepath.Join("projects", projectID.String(), filename), nil
}

// DeleteProjectImage deletes a project image
func (s *StorageService) DeleteProjectImage(relativePath string) error {
	fullPath := filepath.Join(s.config.UploadDir, relativePath)
	return os.Remove(fullPath)
}

// GetImagePath returns the full path for serving an image
func (s *StorageService) GetImagePath(relativePath string) string {
	return filepath.Join(s.config.UploadDir, relativePath)
}

// SaveDocument saves a generated document
func (s *StorageService) SaveDocument(docType, content string, userID uuid.UUID) (string, error) {
	docDir := filepath.Join(s.config.UploadDir, "documents", docType)
	if err := os.MkdirAll(docDir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s_%s_%d.txt", docType, userID.String()[:8], time.Now().Unix())
	filePath := filepath.Join(docDir, filename)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

// DeleteAllProjectImages deletes all images for a project
func (s *StorageService) DeleteAllProjectImages(projectID uuid.UUID) error {
	projectDir := filepath.Join(s.config.UploadDir, "projects", projectID.String())
	return os.RemoveAll(projectDir)
}

// GetUploadURL returns the base URL for uploaded files
func (s *StorageService) GetUploadURL() string {
	return s.config.AppURL + "/uploads/"
}
