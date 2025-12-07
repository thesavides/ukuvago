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

// SeedCategories populates the database with default categories
func SeedCategories() error {
	categories := []models.Category{
		{Name: "FinTech", Icon: "💳", Description: "Financial technology and services"},
		{Name: "HealthTech", Icon: "🏥", Description: "Healthcare and medical technology"},
		{Name: "SaaS", Icon: "☁️", Description: "Software as a Service platforms"},
		{Name: "AI & ML", Icon: "🤖", Description: "Artificial Intelligence and Machine Learning"},
		{Name: "E-Commerce", Icon: "🛒", Description: "Online retail and marketplaces"},
		{Name: "CleanTech", Icon: "🌍", Description: "Renewable energy and sustainability"},
		{Name: "EdTech", Icon: "🎓", Description: "Education technology"},
		{Name: "AgriTech", Icon: "🌾", Description: "Agricultural technology"},
		{Name: "PropTech", Icon: "🏠", Description: "Real estate technology"},
		{Name: "Logistics", Icon: "🚚", Description: "Supply chain and logistics"},
	}

	for _, c := range categories {
		// Use FirstOrCreate to avoid duplicates but ensure these exist
		if err := DB.Where(models.Category{Name: c.Name}).FirstOrCreate(&c).Error; err != nil {
			return err
		}
	}
	log.Println("Seeded categories (idempotent check complete)")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}
