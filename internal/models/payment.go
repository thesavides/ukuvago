package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
)

type Payment struct {
	ID                uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	InvestorID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"investor_id"`
	Amount            int64          `gorm:"not null" json:"amount"` // Amount in cents
	Currency          string         `gorm:"not null;default:'usd'" json:"currency"`
	StripePaymentID   string         `gorm:"index" json:"stripe_payment_id,omitempty"`
	StripeClientSecret string        `json:"-"`
	Status            PaymentStatus  `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ProjectsRemaining int            `gorm:"not null" json:"projects_remaining"`
	ProjectsTotal     int            `gorm:"not null" json:"projects_total"`
	Description       string         `json:"description"`
	ReceiptURL        string         `json:"receipt_url,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Investor *User         `gorm:"foreignKey:InvestorID" json:"investor,omitempty"`
	Views    []ProjectView `gorm:"foreignKey:PaymentID" json:"views,omitempty"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (p *Payment) CanViewMore() bool {
	return p.Status == PaymentStatusCompleted && p.ProjectsRemaining > 0
}

func (p *Payment) UseCredit() bool {
	if p.ProjectsRemaining > 0 {
		p.ProjectsRemaining--
		return true
	}
	return false
}

// PaymentResponse is the safe representation for API responses
type PaymentResponse struct {
	ID                uuid.UUID     `json:"id"`
	Amount            int64         `json:"amount"`
	AmountFormatted   string        `json:"amount_formatted"`
	Currency          string        `json:"currency"`
	Status            PaymentStatus `json:"status"`
	ProjectsRemaining int           `json:"projects_remaining"`
	ProjectsTotal     int           `json:"projects_total"`
	ReceiptURL        string        `json:"receipt_url,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	CompletedAt       *time.Time    `json:"completed_at,omitempty"`
}

func (p *Payment) ToResponse() PaymentResponse {
	formatted := formatCurrency(p.Amount, p.Currency)
	return PaymentResponse{
		ID:                p.ID,
		Amount:            p.Amount,
		AmountFormatted:   formatted,
		Currency:          p.Currency,
		Status:            p.Status,
		ProjectsRemaining: p.ProjectsRemaining,
		ProjectsTotal:     p.ProjectsTotal,
		ReceiptURL:        p.ReceiptURL,
		CreatedAt:         p.CreatedAt,
		CompletedAt:       p.CompletedAt,
	}
}

func formatCurrency(amount int64, currency string) string {
	major := float64(amount) / 100
	switch currency {
	case "zar":
		return "R" + formatFloat(major)
	case "eur":
		return "€" + formatFloat(major)
	case "gbp":
		return "£" + formatFloat(major)
	default:
		return "$" + formatFloat(major)
	}
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return string(rune(int64(f)))
	}
	return ""
}
