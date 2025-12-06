package main

import (
	"log"
	"os"

	"github.com/ukuvago/angel-platform/internal/config"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/routes"
	"github.com/ukuvago/angel-platform/internal/services"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create upload directory
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Printf("Warning: failed to create upload directory: %v", err)
	}

	// Seed admin user
	authService := services.NewAuthService(cfg)
	if err := routes.SeedAdminUser(cfg, authService); err != nil {
		log.Printf("Warning: failed to seed admin user: %v", err)
	} else {
		log.Println("Admin user ready (email: " + cfg.AdminEmail + ")")
	}

	// Setup router
	router := routes.SetupRouter(cfg)

	// Start server
	addr := cfg.ServerHost + ":" + cfg.ServerPort
	log.Printf("Server starting on %s", addr)
	log.Printf("Frontend: http://localhost:%s", cfg.ServerPort)
	log.Printf("API: http://localhost:%s/api", cfg.ServerPort)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
