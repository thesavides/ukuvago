package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Server
	ServerPort string
	ServerHost string

	// Database
	DatabaseURL  string
	DatabaseType string // "postgres" or "sqlite"

	// JWT
	JWTSecret     string
	JWTExpiration int // hours

	// Stripe
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string

	// Payment
	ViewFeeAmount   int64  // in cents
	ViewFeeCurrency string // e.g., "usd", "zar"
	MaxProjectViews int    // max projects per payment

	// Storage
	UploadDir string

	// Email
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	FromEmail    string

	// App
	AppURL   string
	AppName  string
	AdminEmail string
}

func Load() *Config {
	return &Config{
		// Server
		ServerPort: getEnv("SERVER_PORT", "8080"),
		ServerHost: getEnv("SERVER_HOST", "0.0.0.0"),

		// Database
		DatabaseURL:  getEnv("DATABASE_URL", "ukuvago.db"),
		DatabaseType: getEnv("DATABASE_TYPE", "sqlite"),

		// JWT
		JWTSecret:     getEnv("JWT_SECRET", "your-super-secret-key-change-in-production"),
		JWTExpiration: getEnvInt("JWT_EXPIRATION", 72),

		// Stripe
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),

		// Payment
		ViewFeeAmount:   getEnvInt64("VIEW_FEE_AMOUNT", 50000), // $500 in cents
		ViewFeeCurrency: getEnv("VIEW_FEE_CURRENCY", "usd"),
		MaxProjectViews: getEnvInt("MAX_PROJECT_VIEWS", 4),

		// Storage
		UploadDir: getEnv("UPLOAD_DIR", "./uploads"),

		// Email
		SMTPHost:     getEnv("SMTP_HOST", "localhost"),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		FromEmail:    getEnv("FROM_EMAIL", "noreply@ukuvago.com"),

		// App
		AppURL:     getEnv("APP_URL", "http://localhost:8080"),
		AppName:    getEnv("APP_NAME", "UkuvaGo"),
		AdminEmail: getEnv("ADMIN_EMAIL", "admin@ukuvago.com"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}
