package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ukuvago/angel-platform/internal/config"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/routes"
	"github.com/ukuvago/angel-platform/internal/services"
)

func main() {
	log.Println("Starting application...")

	// Load configuration
	cfg := config.Load()
	log.Printf("Config loaded. Database Type: %s", cfg.DatabaseType)

	// Initialize database ASYNCHRONOUSLY to prevent Cloud Run timeouts
	go func() {
		log.Println("Background: Initializing database connection...")
		// Initialize database
		if err := database.Initialize(cfg); err != nil {
			log.Fatalf("CRITICAL: Failed to initialize database: %v", err)
		}
		log.Println("Background: Database initialized successfully.")

		// Seed projects (separate from initial seedData which handles static data)
		if err := database.SeedProjects(); err != nil {
			log.Printf("Warning: Failed to seed projects: %v", err)
		}

		// Seed admin user (depends on DB)
		authService := services.NewAuthService(cfg)
		if err := routes.SeedAdminUser(cfg, authService); err != nil {
			log.Printf("Warning: failed to seed admin user: %v", err)
		} else {
			log.Println("Admin user ready (email: " + cfg.AdminEmail + ")")
		}
	}()

	// Debug: Log web directory structure
	log.Println("DEBUG: Listing web directory contents:")
	filepath.Walk("web", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing %s: %v", path, err)
			return nil
		}
		log.Printf("Found: %s (Size: %d)", path, info.Size())
		return nil
	})

	// Create upload directory
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Printf("Warning: failed to create upload directory: %v", err)
	}

	// Setup router
	log.Println("Setting up router...")
	router := routes.SetupRouter(cfg)

	// Start server
	addr := cfg.ServerHost + ":" + cfg.ServerPort
	log.Printf("Server starting on %s", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
