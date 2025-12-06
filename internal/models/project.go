package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null" json:"name"`
	Description string         `json:"description"`
	Icon        string         `json:"icon"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Projects []Project `gorm:"foreignKey:CategoryID" json:"projects,omitempty"`
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

type ProjectStatus string

const (
	ProjectStatusDraft    ProjectStatus = "draft"
	ProjectStatusPending  ProjectStatus = "pending"
	ProjectStatusApproved ProjectStatus = "approved"
	ProjectStatusRejected ProjectStatus = "rejected"
)

type Project struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	DeveloperID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"developer_id"`
	CategoryID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"category_id"`
	Title           string         `gorm:"not null" json:"title"`
	Tagline         string         `gorm:"size:200" json:"tagline"`
	Description     string         `gorm:"type:text;not null" json:"description"`
	PitchContent    string         `gorm:"type:text" json:"pitch_content"`
	Problem         string         `gorm:"type:text" json:"problem"`
	Solution        string         `gorm:"type:text" json:"solution"`
	TargetMarket    string         `gorm:"type:text" json:"target_market"`
	BusinessModel   string         `gorm:"type:text" json:"business_model"`
	Traction        string         `gorm:"type:text" json:"traction"`
	Team            string         `gorm:"type:text" json:"team"`
	MinInvestment   float64        `gorm:"not null" json:"min_investment"`
	MaxInvestment   float64        `json:"max_investment"`
	EquityOffered   float64        `json:"equity_offered"` // percentage
	ValuationCap    float64        `json:"valuation_cap"`
	Status          ProjectStatus  `gorm:"type:varchar(20);default:'draft'" json:"status"`
	RejectionReason string         `gorm:"type:text" json:"rejection_reason,omitempty"`
	ApprovedAt      *time.Time     `json:"approved_at,omitempty"`
	ApprovedBy      *uuid.UUID     `gorm:"type:uuid" json:"approved_by,omitempty"`
	ViewCount       int            `gorm:"default:0" json:"view_count"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Developer *User           `gorm:"foreignKey:DeveloperID" json:"developer,omitempty"`
	Category  *Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Images    []ProjectImage  `gorm:"foreignKey:ProjectID" json:"images,omitempty"`
	Views     []ProjectView   `gorm:"foreignKey:ProjectID" json:"views,omitempty"`
	Offers    []InvestmentOffer `gorm:"foreignKey:ProjectID" json:"offers,omitempty"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// ProjectPublicInfo is the limited info shown before payment
type ProjectPublicInfo struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	Tagline       string    `json:"tagline"`
	CategoryID    uuid.UUID `json:"category_id"`
	Category      *Category `json:"category,omitempty"`
	MinInvestment float64   `json:"min_investment"`
	PrimaryImage  string    `json:"primary_image,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (p *Project) ToPublicInfo() ProjectPublicInfo {
	info := ProjectPublicInfo{
		ID:            p.ID,
		Title:         p.Title,
		Tagline:       p.Tagline,
		CategoryID:    p.CategoryID,
		Category:      p.Category,
		MinInvestment: p.MinInvestment,
		CreatedAt:     p.CreatedAt,
	}
	// Get primary image if available
	for _, img := range p.Images {
		if img.IsPrimary {
			info.PrimaryImage = img.FilePath
			break
		}
	}
	if info.PrimaryImage == "" && len(p.Images) > 0 {
		info.PrimaryImage = p.Images[0].FilePath
	}
	return info
}

type ProjectImage struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	ProjectID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"project_id"`
	FilePath     string         `gorm:"not null" json:"file_path"`
	FileName     string         `json:"file_name"`
	Caption      string         `json:"caption"`
	DisplayOrder int            `gorm:"default:0" json:"display_order"`
	IsPrimary    bool           `gorm:"default:false" json:"is_primary"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (pi *ProjectImage) BeforeCreate(tx *gorm.DB) error {
	if pi.ID == uuid.Nil {
		pi.ID = uuid.New()
	}
	return nil
}

type ProjectView struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	InvestorID uuid.UUID `gorm:"type:uuid;not null;index" json:"investor_id"`
	ProjectID  uuid.UUID `gorm:"type:uuid;not null;index" json:"project_id"`
	PaymentID  uuid.UUID `gorm:"type:uuid;not null;index" json:"payment_id"`
	ViewedAt   time.Time `gorm:"not null" json:"viewed_at"`

	// Relations
	Investor *User    `gorm:"foreignKey:InvestorID" json:"investor,omitempty"`
	Project  *Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Payment  *Payment `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
}

func (pv *ProjectView) BeforeCreate(tx *gorm.DB) error {
	if pv.ID == uuid.Nil {
		pv.ID = uuid.New()
	}
	if pv.ViewedAt.IsZero() {
		pv.ViewedAt = time.Now()
	}
	return nil
}
