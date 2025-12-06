package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleInvestor  UserRole = "investor"
	RoleDeveloper UserRole = "developer"
	RoleAdmin     UserRole = "admin"
)

type User struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Email         string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash  string         `gorm:"not null" json:"-"`
	Role          UserRole       `gorm:"type:varchar(20);not null" json:"role"`
	FirstName     string         `gorm:"not null" json:"first_name"`
	LastName      string         `gorm:"not null" json:"last_name"`
	Phone         string         `json:"phone"`
	CompanyName   string         `json:"company_name"`
	Bio           string         `gorm:"type:text" json:"bio"`
	EmailVerified bool           `gorm:"default:false" json:"email_verified"`
	VerifyToken   string         `gorm:"index" json:"-"`
	ResetToken    string         `gorm:"index" json:"-"`
	ResetExpires  *time.Time     `json:"-"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Projects []Project         `gorm:"foreignKey:DeveloperID" json:"projects,omitempty"`
	NDAs     []NDA             `gorm:"foreignKey:InvestorID" json:"ndas,omitempty"`
	Payments []Payment         `gorm:"foreignKey:InvestorID" json:"payments,omitempty"`
	Offers   []InvestmentOffer `gorm:"foreignKey:InvestorID" json:"offers,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// UserResponse is a safe representation without sensitive fields
type UserResponse struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	Role          UserRole  `json:"role"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Phone         string    `json:"phone"`
	CompanyName   string    `json:"company_name"`
	Bio           string    `json:"bio"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		Role:          u.Role,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Phone:         u.Phone,
		CompanyName:   u.CompanyName,
		Bio:           u.Bio,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	}
}
