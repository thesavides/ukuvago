package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NDA struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	InvestorID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"investor_id"`
	SignatureData string         `gorm:"type:text;not null" json:"signature_data"` // Base64 encoded signature image
	SignedName    string         `gorm:"not null" json:"signed_name"`
	IPAddress     string         `gorm:"not null" json:"ip_address"`
	UserAgent     string         `json:"user_agent"`
	SignedAt      time.Time      `gorm:"not null" json:"signed_at"`
	ExpiresAt     *time.Time     `json:"expires_at,omitempty"`
	Version       string         `gorm:"default:'1.0'" json:"version"`
	DocumentHash  string         `json:"document_hash"` // Hash of NDA content at time of signing
	CreatedAt     time.Time      `json:"created_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Investor *User `gorm:"foreignKey:InvestorID" json:"investor,omitempty"`
}

func (n *NDA) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	if n.SignedAt.IsZero() {
		n.SignedAt = time.Now()
	}
	return nil
}

func (n *NDA) IsValid() bool {
	if n.ExpiresAt == nil {
		return true
	}
	return time.Now().Before(*n.ExpiresAt)
}

// NDATemplate represents the NDA document content
const NDATemplateContent = `
NON-DISCLOSURE AGREEMENT

This Non-Disclosure Agreement ("Agreement") is entered into as of the date of electronic signature below.

BETWEEN:
UkuvaGo Platform ("Disclosing Party")
AND
The undersigned individual or entity ("Receiving Party")

1. PURPOSE
The Receiving Party wishes to receive access to confidential startup project information, including but not limited to business plans, financial projections, technical specifications, and intellectual property ("Confidential Information") for the purpose of evaluating potential investment opportunities.

2. CONFIDENTIAL INFORMATION
"Confidential Information" includes all information disclosed by the Disclosing Party or project developers through the UkuvaGo platform, whether oral, written, or in any other form, that is designated as confidential or that reasonably should be understood to be confidential.

3. OBLIGATIONS
The Receiving Party agrees to:
a) Hold all Confidential Information in strict confidence;
b) Not disclose Confidential Information to any third party without prior written consent;
c) Use Confidential Information solely for evaluating investment opportunities;
d) Not copy, reproduce, or distribute Confidential Information except as necessary for evaluation;
e) Protect Confidential Information using the same degree of care used to protect their own confidential information, but no less than reasonable care.

4. EXCLUSIONS
This Agreement does not apply to information that:
a) Is or becomes publicly available through no fault of the Receiving Party;
b) Was known to the Receiving Party prior to disclosure;
c) Is independently developed by the Receiving Party without use of Confidential Information;
d) Is disclosed with the written approval of the Disclosing Party;
e) Is required to be disclosed by law or court order.

5. TERM
This Agreement shall remain in effect for a period of two (2) years from the date of signing.

6. NO LICENSE
Nothing in this Agreement grants the Receiving Party any license or rights to any intellectual property of the Disclosing Party or project developers.

7. RETURN OF INFORMATION
Upon request, the Receiving Party shall promptly return or destroy all Confidential Information and any copies thereof.

8. REMEDIES
The Receiving Party acknowledges that any breach of this Agreement may cause irreparable harm, and the Disclosing Party shall be entitled to seek equitable relief, including injunction, in addition to any other remedies available at law.

9. GOVERNING LAW
This Agreement shall be governed by and construed in accordance with applicable laws.

10. ELECTRONIC SIGNATURE
The parties agree that electronic signatures shall be legally binding and have the same force and effect as handwritten signatures.

BY SIGNING BELOW, THE RECEIVING PARTY ACKNOWLEDGES THAT THEY HAVE READ, UNDERSTAND, AND AGREE TO BE BOUND BY THE TERMS OF THIS AGREEMENT.
`
