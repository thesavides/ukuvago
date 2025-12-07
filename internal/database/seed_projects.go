package database

import (
	"log"
	"math/rand"

	"github.com/ukuvago/angel-platform/internal/models"
)

// SeedProjects creates sample startup projects for demonstration
func SeedProjects() error {
	var count int64
	DB.Model(&models.Project{}).Count(&count)
	if count > 0 {
		return nil // Already seeded
	}

	// Get a developer user or create one
	var developer models.User
	if err := DB.Where("role = ?", models.RoleDeveloper).First(&developer).Error; err != nil {
		log.Println("No developer users found, skipping project seeding")
		return nil
	}

	// Get categories
	var categories []models.Category
	DB.Find(&categories)
	if len(categories) == 0 {
		return nil
	}

	projects := []struct {
		Title         string
		Tagline       string
		Category      string
		Description   string
		Problem       string
		Solution      string
		MinInvestment float64
		ValuationCap  float64
	}{
		{
			Title:         "PayFlow Africa",
			Tagline:       "Cross-border payments made simple for African SMEs",
			Category:      "FinTech",
			Description:   "PayFlow Africa enables seamless cross-border payments for small and medium businesses across the African continent, reducing transaction costs by up to 70%.",
			Problem:       "African SMEs lose billions annually to high cross-border transaction fees and slow payment processing times.",
			Solution:      "Our blockchain-based payment infrastructure provides instant, low-cost transfers between African nations.",
			MinInvestment: 25000,
			ValuationCap:  2000000,
		},
		{
			Title:         "MediConnect",
			Tagline:       "AI-powered telemedicine for rural communities",
			Category:      "HealthTech",
			Description:   "MediConnect brings quality healthcare to underserved rural areas through AI diagnostics and video consultations with certified doctors.",
			Problem:       "60% of rural populations lack access to qualified healthcare professionals.",
			Solution:      "Mobile-first platform with AI triage, connecting patients with specialists via video calls.",
			MinInvestment: 50000,
			ValuationCap:  3000000,
		},
		{
			Title:         "LearnPath",
			Tagline:       "Personalized skills training for the future workforce",
			Category:      "EdTech",
			Description:   "AI-driven learning platform that creates personalized upskilling paths based on career goals and market demand.",
			Problem:       "Skills gap costing global economy $8.5 trillion in lost productivity.",
			Solution:      "Adaptive learning algorithms that match learners with in-demand skills and job opportunities.",
			MinInvestment: 30000,
			ValuationCap:  2500000,
		},
		{
			Title:         "FarmSense",
			Tagline:       "IoT precision farming for smallholder farmers",
			Category:      "AgriTech",
			Description:   "Affordable IoT sensors and AI analytics helping small-scale farmers optimize crop yields and reduce water usage.",
			Problem:       "Smallholder farmers lose 40% of crops due to inefficient farming practices.",
			Solution:      "Low-cost sensor network with SMS-based insights for farmers without smartphones.",
			MinInvestment: 20000,
			ValuationCap:  1500000,
		},
		{
			Title:         "SolarShare",
			Tagline:       "Community solar microgrids for energy independence",
			Category:      "CleanTech",
			Description:   "Enabling communities to build, own, and trade solar energy through tokenized microgrids.",
			Problem:       "600 million Africans lack reliable electricity access.",
			Solution:      "Peer-to-peer energy trading platform with community-owned solar installations.",
			MinInvestment: 75000,
			ValuationCap:  5000000,
		},
		{
			Title:         "PropChain",
			Tagline:       "Fractional real estate investment on blockchain",
			Category:      "PropTech",
			Description:   "Democratizing real estate investment by enabling fractional ownership of premium properties.",
			Problem:       "Real estate investment requires significant capital, excluding most investors.",
			Solution:      "Tokenized property shares starting from $100, with automated rental income distribution.",
			MinInvestment: 100000,
			ValuationCap:  8000000,
		},
		{
			Title:         "QuickMart",
			Tagline:       "15-minute grocery delivery for urban areas",
			Category:      "E-Commerce",
			Description:   "Dark store network enabling ultra-fast grocery delivery to urban consumers.",
			Problem:       "Traditional e-commerce takes days; consumers want instant gratification.",
			Solution:      "Network of micro-fulfillment centers within 2km of customers.",
			MinInvestment: 150000,
			ValuationCap:  10000000,
		},
		{
			Title:         "TeamSync",
			Tagline:       "All-in-one remote team management platform",
			Category:      "SaaS",
			Description:   "Unified workspace combining project management, communication, and HR tools for distributed teams.",
			Problem:       "Remote teams use 10+ different tools, causing productivity loss.",
			Solution:      "Integrated platform with async video, task management, and performance tracking.",
			MinInvestment: 40000,
			ValuationCap:  3500000,
		},
		{
			Title:         "VisionAI",
			Tagline:       "Computer vision for retail analytics",
			Category:      "AI/ML",
			Description:   "AI-powered cameras that provide real-time customer behavior analytics for retail stores.",
			Problem:       "Retailers lack insight into in-store customer behavior and preferences.",
			Solution:      "Privacy-preserving computer vision that tracks traffic patterns and engagement.",
			MinInvestment: 80000,
			ValuationCap:  6000000,
		},
		{
			Title:         "SmartFactory",
			Tagline:       "Industrial IoT for manufacturing efficiency",
			Category:      "IoT",
			Description:   "End-to-end IoT platform for predictive maintenance and production optimization.",
			Problem:       "Unplanned downtime costs manufacturers $50 billion annually.",
			Solution:      "Sensor-based monitoring with ML-driven failure prediction.",
			MinInvestment: 120000,
			ValuationCap:  9000000,
		},
		{
			Title:         "CyberShield",
			Tagline:       "AI-powered threat detection for SMBs",
			Category:      "Cybersecurity",
			Description:   "Enterprise-grade cybersecurity made affordable for small businesses.",
			Problem:       "43% of cyberattacks target small businesses; most can't afford protection.",
			Solution:      "Automated threat detection and response at 1/10th the cost of enterprise solutions.",
			MinInvestment: 60000,
			ValuationCap:  4500000,
		},
		{
			Title:         "FleetTrack",
			Tagline:       "Last-mile delivery optimization platform",
			Category:      "Logistics",
			Description:   "AI route optimization and real-time tracking for delivery fleets.",
			Problem:       "Inefficient routes cost logistics companies 30% more in fuel and time.",
			Solution:      "Dynamic routing algorithms that adapt to traffic and delivery windows.",
			MinInvestment: 45000,
			ValuationCap:  3000000,
		},
	}

	for _, p := range projects {
		var category models.Category
		for _, c := range categories {
			if c.Name == p.Category {
				category = c
				break
			}
		}

		project := &models.Project{
			DeveloperID:   developer.ID,
			CategoryID:    category.ID,
			Title:         p.Title,
			Tagline:       p.Tagline,
			Description:   p.Description,
			Problem:       p.Problem,
			Solution:      p.Solution,
			MinInvestment: p.MinInvestment,
			MaxInvestment: p.MinInvestment * 10,
			ValuationCap:  p.ValuationCap,
			EquityOffered: float64(5 + rand.Intn(15)),
			Status:        models.ProjectStatusApproved,
			PitchContent:  generatePitch(p.Title, p.Problem, p.Solution),
		}

		DB.Create(project)
	}

	log.Printf("Seeded %d sample projects", len(projects))
	return nil
}

func generatePitch(title, problem, solution string) string {
	return `# ` + title + ` - Investor Pitch

## The Problem
` + problem + `

## Our Solution
` + solution + `

## Market Opportunity
- Total Addressable Market: $10B+
- Growing at 25% annually
- First-mover advantage in key markets

## Traction
- 1,000+ beta users
- 15% month-over-month growth
- Key partnerships in development

## The Team
Experienced founders with background in technology and the target industry.

## The Ask
We're raising angel funding to:
- Scale our technology platform
- Expand market presence
- Build out the core team

Join us in transforming this industry!`
}
