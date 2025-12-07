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
	log.Println("Starting application...")

	// Load configuration
	cfg := config.Load()
	log.Printf("Config loaded. Database Type: %s", cfg.DatabaseType)

	// Initialize database
	log.Println("Initializing database connection...")
	if err := database.Initialize(cfg); err != nil {
		log.Printf("CRITICAL: Failed to initialize database: %v", err)
		log.Println("Server will start but will likely fail requests depending on DB.")
		// We deliberately don't Fatalf here so the logs have time to flush and container acts alive long enough to see.
	} else {
		log.Println("Database initialized successfully.")
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
	log.Println("Setting up router...")
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
