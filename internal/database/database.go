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
		&models.TeamMember{},
		&models.NDA{},
		&models.Payment{},
		&models.ProjectView{},
		&models.InvestmentOffer{},
		&models.TermSheet{},
	)
}

func seedData() error {
	// Seed categories if empty
	return SeedCategories()
}

// SeedCategories populates the database with default categories if empty
func SeedCategories() error {
	var count int64
	DB.Model(&models.Category{}).Count(&count)
	if count > 0 {
		return nil
	}

	categories := []models.Category{
		{Name: "FinTech", Icon: "💳", Description: "Financial technology and services"},
		{Name: "HealthTech", Icon: "🏥", Description: "Healthcare and medical technology"},
		{Name: "SaaS / AI", Icon: "🚀", Description: "Software as a Service and Artificial Intelligence"},
		{Name: "E-Commerce", Icon: "🛒", Description: "Online retail and marketplaces"},
		{Name: "CleanTech", Icon: "🌍", Description: "Renewable energy and sustainability"},
		{Name: "EdTech", Icon: "🎓", Description: "Education technology"},
		{Name: "AgriTech", Icon: "🌾", Description: "Agricultural technology"},
		{Name: "Logistics", Icon: "🚚", Description: "Supply chain and logistics"},
		{Name: "PropTech", Icon: "🏠", Description: "Real estate technology"},
	}

	for _, c := range categories {
		if err := DB.Create(&c).Error; err != nil {
			return err
		}
	}
	log.Println("Seeded categories")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}
