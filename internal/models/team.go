package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamMember struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	ProjectID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"project_id"`
	Name       string         `gorm:"not null" json:"name"`
	Role       string         `gorm:"not null" json:"role"` // e.g., CEO, CTO
	ProfileURL string         `json:"profile_url"`          // LinkedIn, etc.
	IsLead     bool           `gorm:"default:false" json:"is_lead"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *TeamMember) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
