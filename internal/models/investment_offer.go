package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OfferStatus string

const (
	OfferStatusPending   OfferStatus = "pending"
	OfferStatusAccepted  OfferStatus = "accepted"
	OfferStatusRejected  OfferStatus = "rejected"
	OfferStatusWithdrawn OfferStatus = "withdrawn"
	OfferStatusExpired   OfferStatus = "expired"
)

type InvestmentOffer struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	InvestorID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"investor_id"`
	ProjectID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"project_id"`
	OfferAmount   float64        `gorm:"not null" json:"offer_amount"`
	EquityRequest float64        `json:"equity_request"` // Percentage if applicable
	TermsNotes    string         `gorm:"type:text" json:"terms_notes"`
	Status        OfferStatus    `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ResponseNotes string         `gorm:"type:text" json:"response_notes,omitempty"`
	ExpiresAt     *time.Time     `json:"expires_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	RespondedAt   *time.Time     `json:"responded_at,omitempty"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Investor  *User      `gorm:"foreignKey:InvestorID" json:"investor,omitempty"`
	Project   *Project   `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	TermSheet *TermSheet `gorm:"foreignKey:OfferID" json:"term_sheet,omitempty"`
}

func (o *InvestmentOffer) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	// Set default expiration to 30 days
	if o.ExpiresAt == nil {
		expires := time.Now().AddDate(0, 0, 30)
		o.ExpiresAt = &expires
	}
	return nil
}

func (o *InvestmentOffer) IsExpired() bool {
	if o.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*o.ExpiresAt)
}

func (o *InvestmentOffer) CanRespond() bool {
	return o.Status == OfferStatusPending && !o.IsExpired()
}

type TermSheetStatus string

const (
	TermSheetStatusDraft          TermSheetStatus = "draft"
	TermSheetStatusInvestorSigned TermSheetStatus = "investor_signed"
	TermSheetStatusCompleted      TermSheetStatus = "completed"
	TermSheetStatusVoided         TermSheetStatus = "voided"
)

type TermSheet struct {
	ID                  uuid.UUID       `gorm:"type:uuid;primary_key" json:"id"`
	OfferID             uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex" json:"offer_id"`
	DocumentPath        string          `json:"document_path,omitempty"`
	InvestorSignature   string          `gorm:"type:text" json:"investor_signature,omitempty"`
	DeveloperSignature  string          `gorm:"type:text" json:"developer_signature,omitempty"`
	InvestorSignedAt    *time.Time      `json:"investor_signed_at,omitempty"`
	DeveloperSignedAt   *time.Time      `json:"developer_signed_at,omitempty"`
	InvestorIP          string          `json:"investor_ip,omitempty"`
	DeveloperIP         string          `json:"developer_ip,omitempty"`
	Status              TermSheetStatus `gorm:"type:varchar(20);default:'draft'" json:"status"`
	
	// SAFE Note Terms
	InvestmentAmount    float64    `json:"investment_amount"`
	ValuationCap        float64    `json:"valuation_cap"`
	DiscountRate        float64    `json:"discount_rate"` // Percentage
	ProRataRights       bool       `json:"pro_rata_rights"`
	MFNClause           bool       `json:"mfn_clause"` // Most Favored Nation
	
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Offer *InvestmentOffer `gorm:"foreignKey:OfferID" json:"offer,omitempty"`
}

func (t *TermSheet) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (t *TermSheet) IsFullySigned() bool {
	return t.InvestorSignature != "" && t.DeveloperSignature != ""
}

// SAFETemplateContent is the SAFE note template
const SAFETemplateContent = `
SIMPLE AGREEMENT FOR FUTURE EQUITY (SAFE)

THIS AGREEMENT is entered into as of the Effective Date between:

COMPANY: {{.CompanyName}}
("Company")

AND

INVESTOR: {{.InvestorName}}
("Investor")

1. INVESTMENT
The Investor agrees to invest {{.InvestmentAmount}} (the "Purchase Amount") in the Company.

2. SAFE TERMS
   a) Valuation Cap: {{.ValuationCap}}
   b) Discount Rate: {{.DiscountRate}}%
   c) Pro-Rata Rights: {{.ProRataRights}}

3. CONVERSION EVENTS
This SAFE will convert into equity upon:
   a) Equity Financing: Next priced equity round of at least $1,000,000
   b) Liquidity Event: IPO, acquisition, or sale of the Company
   c) Dissolution Event: Winding up of the Company

4. CONVERSION MECHANICS
Upon an Equity Financing:
   - If using Valuation Cap: Shares = Purchase Amount / (Valuation Cap / Company Capitalization)
   - If using Discount: Shares = Purchase Amount / (Price Per Share × (1 - Discount Rate))
   - Investor receives the more favorable calculation

5. REPRESENTATIONS
The Company represents that:
   a) It is duly organized and validly existing
   b) This SAFE has been duly authorized
   c) The shares issued upon conversion will be validly issued

The Investor represents that:
   a) They are an accredited investor
   b) They have reviewed Company materials
   c) They understand the risks of this investment

6. MISCELLANEOUS
   a) This SAFE is governed by applicable law
   b) This SAFE may not be assigned without consent
   c) This constitutes the entire agreement between the parties

IN WITNESS WHEREOF, the parties have executed this SAFE as of the date first written above.

COMPANY SIGNATURE:
_________________________________
Name: {{.CompanyRepName}}
Title: {{.CompanyRepTitle}}
Date: {{.CompanySignDate}}

INVESTOR SIGNATURE:
_________________________________
Name: {{.InvestorName}}
Date: {{.InvestorSignDate}}
`
