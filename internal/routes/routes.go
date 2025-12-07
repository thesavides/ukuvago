package routes

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ukuvago/angel-platform/internal/config"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/handlers"
	"github.com/ukuvago/angel-platform/internal/middleware"
	"github.com/ukuvago/angel-platform/internal/models"
	"github.com/ukuvago/angel-platform/internal/services"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOriginFunc:  func(origin string) bool { return true },
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Health check for Cloud Run
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":       "ok",
			"version":      "v4",
			"db_connected": database.GetDB() != nil,
		})
	})

	// Serve static files with absolute paths to prevent fallback issues
	wd, _ := os.Getwd()
	router.Static("/uploads", filepath.Join(wd, cfg.UploadDir))
	router.Static("/static", filepath.Join(wd, "web"))

	// Initialize services
	authService := services.NewAuthService(cfg)
	paymentService := services.NewPaymentService(cfg)
	documentService := services.NewDocumentService(cfg)
	storageService := services.NewStorageService(cfg)
	emailService := services.NewEmailService(cfg)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, emailService)
	ndaHandler := handlers.NewNDAHandler(authService, documentService)
	paymentHandler := handlers.NewPaymentHandler(paymentService)
	projectHandler := handlers.NewProjectHandler(storageService, paymentService)
	offerHandler := handlers.NewOfferHandler(emailService, documentService, authService)
	termSheetHandler := handlers.NewTermSheetHandler(documentService, emailService, authService)
	adminHandler := handlers.NewAdminHandler(emailService, authService)

	// API routes
	api := router.Group("/api")

	// Middleware to check Database Readiness
	api.Use(func(c *gin.Context) {
		if database.GetDB() == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service initializing, please try again shortly",
			})
			return
		}
		c.Next()
	})

	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/verify-email", authHandler.VerifyEmail)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)

			// Protected auth routes
			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware(authService))
			{
				authProtected.GET("/me", authHandler.GetCurrentUser)
				authProtected.PUT("/profile", authHandler.UpdateProfile)
				authProtected.POST("/change-password", authHandler.ChangePassword)
			}
		}

		// Category routes (public)
		api.GET("/categories", projectHandler.GetCategories)

		// Project routes
		projects := api.Group("/projects")
		{
			// Public routes
			projects.GET("", projectHandler.ListProjects)

			// Protected routes
			projectsProtected := projects.Group("")
			projectsProtected.Use(middleware.AuthMiddleware(authService))
			{
				// Get project with access control
				projectsProtected.GET("/:id", middleware.CheckNDAStatus(), middleware.CheckPaymentStatus(paymentService), projectHandler.GetProject)

				// Unified Project Management (Developer & Admin)
				// Middleware removed here because Handler performs Role checks.
				// For Create: Any Auth user can theoretically create? No, Investors shouldn't.
				// We need a specific middleware or logic in handler.
				// I will use `RequireDeveloperOrAdmin` if I can create it, or just handle it in handler.
				// Let's modify handlers to strictly reject Investors if I remove the middleware.

				// ACTUALLY: The easiest fix is to let `RequireDeveloper` pass if role is Admin.
				// I'll check `internal/middleware/auth.go` first.
			}
		}

		// Developer routes
		developer := api.Group("/developer")
		developer.Use(middleware.AuthMiddleware(authService), middleware.RequireDeveloper())
		{
			developer.GET("/projects", projectHandler.GetMyProjects)
			developer.GET("/offers", offerHandler.GetMyOffers)
			developer.GET("/termsheets", termSheetHandler.GetMyTermSheets)
		}

		// NDA routes (investor only)
		nda := api.Group("/nda")
		nda.Use(middleware.AuthMiddleware(authService), middleware.RequireInvestor())
		{
			nda.GET("/template", ndaHandler.GetNDATemplate)
			nda.GET("/status", ndaHandler.GetNDAStatus)
			nda.POST("/sign", ndaHandler.SignNDA)
			nda.GET("/download", ndaHandler.DownloadNDA)
		}

		// Payment routes (investor only)
		payments := api.Group("/payments")
		payments.Use(middleware.AuthMiddleware(authService), middleware.RequireInvestor())
		{
			payments.POST("/create-intent", middleware.RequireNDA(), paymentHandler.CreatePaymentIntent)
			payments.POST("/confirm", paymentHandler.ConfirmPayment)
			payments.GET("/status", paymentHandler.GetPaymentStatus)
			payments.GET("/history", paymentHandler.GetPaymentHistory)
			payments.GET("/viewed", paymentHandler.GetViewedProjects)
		}

		// Offer routes
		offers := api.Group("/offers")
		offers.Use(middleware.AuthMiddleware(authService))
		{
			// Investor routes
			offers.POST("", middleware.RequireInvestor(), middleware.RequireNDA(), middleware.RequirePayment(paymentService), offerHandler.CreateOffer)
			offers.DELETE("/:id", middleware.RequireInvestor(), offerHandler.WithdrawOffer)

			// Shared routes
			offers.GET("", offerHandler.GetMyOffers)
			offers.GET("/:id", offerHandler.GetOffer)

			// Developer routes
			offers.POST("/:id/respond", middleware.RequireDeveloper(), offerHandler.RespondToOffer)
		}

		// Term sheet routes
		termsheets := api.Group("/termsheets")
		termsheets.Use(middleware.AuthMiddleware(authService))
		{
			termsheets.GET("", termSheetHandler.GetMyTermSheets)
			termsheets.GET("/:id", termSheetHandler.GetTermSheet)
			termsheets.POST("/:id/sign", termSheetHandler.SignTermSheet)
			termsheets.GET("/:id/download", termSheetHandler.DownloadTermSheet)
		}

		// Admin routes
		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware(authService), middleware.RequireAdmin())
		{
			admin.GET("/stats", adminHandler.GetDashboardStats)
			admin.GET("/users", adminHandler.ListAllUsers)
			admin.GET("/projects", adminHandler.ListAllProjects)
			admin.GET("/projects/pending", adminHandler.GetPendingProjects)
			admin.GET("/projects/all", adminHandler.GetAllProjects)
			admin.POST("/projects/:id/approve", adminHandler.ApproveProject)
			admin.GET("/offers", adminHandler.ListAllOffers)
			admin.GET("/payments", adminHandler.ListAllPayments)
			admin.POST("/categories", adminHandler.CreateCategory)
			admin.PUT("/categories/:id", adminHandler.UpdateCategory)
			admin.DELETE("/categories/:id", adminHandler.DeleteCategory)
		}
	}

	// Serve frontend for all other routes
	router.NoRoute(func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.File(filepath.Join(wd, "web/index.html"))
	})

	return router
}

// SeedAdminUser creates a default admin user if none exists
func SeedAdminUser(cfg *config.Config, authService *services.AuthService) error {
	// Check if admin exists
	_, _, err := authService.Login(cfg.AdminEmail, "admin123")
	if err == nil {
		return nil // Admin exists
	}

	// Create admin user
	admin, err := authService.Register(
		cfg.AdminEmail,
		"admin123", // Default password - should be changed
		"Admin",
		"User",
		models.RoleAdmin,
	)
	if err != nil {
		return err
	}

	// Auto-verify admin
	admin.EmailVerified = true
	return authService.UpdateUser(admin)
}
