package database

import (
	"log"

	"github.com/glebarez/sqlite"
	"github.com/ukuvago/angel-platform/internal/config"
	"github.com/ukuvago/angel-platform/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Initialize(cfg *config.Config) error {
	var dialector gorm.Dialector

	switch cfg.DatabaseType {
	case "postgres":
		dialector = postgres.Open(cfg.DatabaseURL)
	default:
		dialector = sqlite.Open(cfg.DatabaseURL)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	DB = db
	log.Println("Database connected successfully")

	// Auto-migrate models
	if err := autoMigrate(); err != nil {
		return err
	}

	// Seed initial data
	if err := seedData(); err != nil {
		log.Printf("Warning: seed data error: %v", err)
	}

	return nil
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Project{},
		&models.ProjectImage{},
		&models.NDA{},
		&models.Payment{},
		&models.ProjectView{},
		&models.InvestmentOffer{},
		&models.TermSheet{},
	)
}

func seedData() error {
	// Seed categories if empty
	var count int64
	DB.Model(&models.Category{}).Count(&count)
	if count == 0 {
		categories := []models.Category{
			{Name: "FinTech", Description: "Financial technology and banking innovations", Icon: "💰"},
			{Name: "HealthTech", Description: "Healthcare and medical technology", Icon: "🏥"},
			{Name: "EdTech", Description: "Education technology and e-learning", Icon: "📚"},
			{Name: "AgriTech", Description: "Agricultural technology and farming innovations", Icon: "🌾"},
			{Name: "CleanTech", Description: "Environmental and sustainability solutions", Icon: "🌱"},
			{Name: "PropTech", Description: "Real estate and property technology", Icon: "🏠"},
			{Name: "E-Commerce", Description: "Online retail and marketplace solutions", Icon: "🛒"},
			{Name: "SaaS", Description: "Software as a Service platforms", Icon: "☁️"},
			{Name: "AI/ML", Description: "Artificial intelligence and machine learning", Icon: "🤖"},
			{Name: "IoT", Description: "Internet of Things and connected devices", Icon: "📡"},
			{Name: "Cybersecurity", Description: "Security and data protection", Icon: "🔒"},
			{Name: "Logistics", Description: "Supply chain and delivery solutions", Icon: "🚚"},
		}
		for _, cat := range categories {
			DB.Create(&cat)
		}
		log.Println("Seeded categories")
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}
