package services

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"github.com/ukuvago/angel-platform/internal/config"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/models"
)

type DocumentService struct {
	config *config.Config
}

func NewDocumentService(cfg *config.Config) *DocumentService {
	return &DocumentService{config: cfg}
}

// GenerateNDAPDF generates a PDF of the signed NDA
func (s *DocumentService) GenerateNDAPDF(nda *models.NDA, investor *models.User) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(190, 10, "NON-DISCLOSURE AGREEMENT", "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Content
	pdf.SetFont("Arial", "", 10)
	
	content := `This Non-Disclosure Agreement ("Agreement") is entered into as of ` + nda.SignedAt.Format("January 2, 2006") + `.

BETWEEN:
UkuvaGo Platform ("Disclosing Party")
AND
` + investor.FullName() + ` ("Receiving Party")

1. PURPOSE
The Receiving Party wishes to receive access to confidential startup project information for the purpose of evaluating potential investment opportunities.

2. CONFIDENTIAL INFORMATION
"Confidential Information" includes all information disclosed through the UkuvaGo platform, including business plans, financial projections, technical specifications, and intellectual property.

3. OBLIGATIONS
The Receiving Party agrees to:
a) Hold all Confidential Information in strict confidence;
b) Not disclose Confidential Information to any third party without prior written consent;
c) Use Confidential Information solely for evaluating investment opportunities;
d) Protect Confidential Information using reasonable care.

4. TERM
This Agreement shall remain in effect for a period of two (2) years from the date of signing.

5. ELECTRONIC SIGNATURE
The parties agree that electronic signatures shall be legally binding.`

	pdf.MultiCell(190, 5, content, "", "", false)
	pdf.Ln(10)

	// Signature section
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 10, "RECEIVING PARTY SIGNATURE")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(190, 5, "Name: "+nda.SignedName)
	pdf.Ln(5)
	pdf.Cell(190, 5, "Email: "+investor.Email)
	pdf.Ln(5)
	pdf.Cell(190, 5, "Signed: "+nda.SignedAt.Format("January 2, 2006 15:04:05 MST"))
	pdf.Ln(5)
	pdf.Cell(190, 5, "IP Address: "+nda.IPAddress)
	pdf.Ln(5)
	pdf.Cell(190, 5, "Document Version: "+nda.Version)
	pdf.Ln(10)

	// Notice
	pdf.SetFont("Arial", "I", 8)
	pdf.MultiCell(190, 4, "This document was electronically signed via the UkuvaGo platform. The signature data is securely stored and this document serves as proof of agreement.", "", "", false)

	// Save PDF
	docsDir := filepath.Join(s.config.UploadDir, "documents", "ndas")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("nda_%s_%s.pdf", nda.ID.String()[:8], time.Now().Format("20060102"))
	filePath := filepath.Join(docsDir, filename)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

// SAFEData contains data for SAFE note template
type SAFEData struct {
	CompanyName      string
	InvestorName     string
	InvestmentAmount string
	ValuationCap     string
	DiscountRate     string
	ProRataRights    string
	CompanyRepName   string
	CompanyRepTitle  string
	CompanySignDate  string
	InvestorSignDate string
}

// GenerateSAFENotePDF generates a SAFE note term sheet PDF
func (s *DocumentService) GenerateSAFENotePDF(termSheet *models.TermSheet, offer *models.InvestmentOffer, investor *models.User, developer *models.User, project *models.Project) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Header
	pdf.SetFont("Arial", "B", 20)
	pdf.CellFormat(190, 12, "SAFE", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 12)
	pdf.CellFormat(190, 8, "Simple Agreement for Future Equity", "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Parties
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 8, "PARTIES")
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(95, 6, "Company: "+developer.CompanyName)
	pdf.Cell(95, 6, "Investor: "+investor.FullName())
	pdf.Ln(10)

	// Investment Terms
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 8, "INVESTMENT TERMS")
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)

	pdf.Cell(60, 6, "Project:")
	pdf.Cell(130, 6, project.Title)
	pdf.Ln(6)

	pdf.Cell(60, 6, "Investment Amount:")
	pdf.Cell(130, 6, fmt.Sprintf("$%.2f", termSheet.InvestmentAmount))
	pdf.Ln(6)

	if termSheet.ValuationCap > 0 {
		pdf.Cell(60, 6, "Valuation Cap:")
		pdf.Cell(130, 6, fmt.Sprintf("$%.2f", termSheet.ValuationCap))
		pdf.Ln(6)
	}

	if termSheet.DiscountRate > 0 {
		pdf.Cell(60, 6, "Discount Rate:")
		pdf.Cell(130, 6, fmt.Sprintf("%.1f%%", termSheet.DiscountRate))
		pdf.Ln(6)
	}

	proRata := "No"
	if termSheet.ProRataRights {
		proRata = "Yes"
	}
	pdf.Cell(60, 6, "Pro-Rata Rights:")
	pdf.Cell(130, 6, proRata)
	pdf.Ln(10)

	// Terms
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 8, "KEY TERMS")
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 9)

	terms := `1. CONVERSION EVENTS
This SAFE will automatically convert into equity upon:
• Equity Financing: A bona fide transaction with the primary purpose of raising capital
• Liquidity Event: A change of control, IPO, or direct listing
• Dissolution Event: Voluntary or involuntary termination of the Company

2. CONVERSION MECHANICS
Upon an Equity Financing, the Investor will receive the greater of:
• Shares based on the Valuation Cap price, or
• Shares based on the Discount Rate applied to the price per share

3. REPRESENTATIONS
Both parties represent they have the authority to enter into this agreement and that this investment complies with applicable securities laws.`

	pdf.MultiCell(190, 5, terms, "", "", false)
	pdf.Ln(10)

	// Signatures
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 8, "SIGNATURES")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(95, 6, "COMPANY")
	pdf.Cell(95, 6, "INVESTOR")
	pdf.Ln(8)

	// Company signature
	if termSheet.DeveloperSignature != "" {
		pdf.Cell(95, 6, "Signed: "+termSheet.DeveloperSignedAt.Format("Jan 2, 2006"))
	} else {
		pdf.Cell(95, 6, "Pending signature")
	}

	// Investor signature
	if termSheet.InvestorSignature != "" {
		pdf.Cell(95, 6, "Signed: "+termSheet.InvestorSignedAt.Format("Jan 2, 2006"))
	} else {
		pdf.Cell(95, 6, "Pending signature")
	}
	pdf.Ln(6)

	pdf.Cell(95, 6, developer.FullName())
	pdf.Cell(95, 6, investor.FullName())
	pdf.Ln(6)

	pdf.Cell(95, 6, developer.CompanyName)
	pdf.Cell(95, 6, investor.CompanyName)
	pdf.Ln(15)

	// Footer
	pdf.SetFont("Arial", "I", 8)
	pdf.MultiCell(190, 4, "This document was generated via the UkuvaGo platform. Electronic signatures are legally binding under applicable e-signature laws.", "", "", false)

	// Save PDF
	docsDir := filepath.Join(s.config.UploadDir, "documents", "termsheets")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("safe_%s_%s.pdf", termSheet.ID.String()[:8], time.Now().Format("20060102"))
	filePath := filepath.Join(docsDir, filename)

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

// RenderTemplate renders an HTML template with data
func (s *DocumentService) RenderTemplate(templateContent string, data interface{}) (string, error) {
	tmpl, err := template.New("doc").Parse(templateContent)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// CreateTermSheet creates a new term sheet for an accepted offer
func (s *DocumentService) CreateTermSheet(offer *models.InvestmentOffer, project *models.Project) (*models.TermSheet, error) {
	db := database.GetDB()

	termSheet := &models.TermSheet{
		OfferID:          offer.ID,
		InvestmentAmount: offer.OfferAmount,
		ValuationCap:     project.ValuationCap,
		DiscountRate:     20.0, // Default 20% discount
		ProRataRights:    true,
		Status:           models.TermSheetStatusDraft,
	}

	if err := db.Create(termSheet).Error; err != nil {
		return nil, err
	}

	return termSheet, nil
}

// SignTermSheet records a signature on a term sheet
func (s *DocumentService) SignTermSheet(termSheetID uuid.UUID, userID uuid.UUID, signatureData, ipAddress string) (*models.TermSheet, error) {
	db := database.GetDB()

	var termSheet models.TermSheet
	if err := db.Preload("Offer").First(&termSheet, "id = ?", termSheetID).Error; err != nil {
		return nil, err
	}

	// Get the offer to determine user role
	var offer models.InvestmentOffer
	if err := db.Preload("Project").First(&offer, "id = ?", termSheet.OfferID).Error; err != nil {
		return nil, err
	}

	now := time.Now()

	if offer.InvestorID == userID {
		// Investor signing
		termSheet.InvestorSignature = signatureData
		termSheet.InvestorSignedAt = &now
		termSheet.InvestorIP = ipAddress

		if termSheet.DeveloperSignature != "" {
			termSheet.Status = models.TermSheetStatusCompleted
		} else {
			termSheet.Status = models.TermSheetStatusInvestorSigned
		}
	} else if offer.Project.DeveloperID == userID {
		// Developer signing
		termSheet.DeveloperSignature = signatureData
		termSheet.DeveloperSignedAt = &now
		termSheet.DeveloperIP = ipAddress

		if termSheet.InvestorSignature != "" {
			termSheet.Status = models.TermSheetStatusCompleted
		}
	} else {
		return nil, fmt.Errorf("user not authorized to sign this term sheet")
	}

	if err := db.Save(&termSheet).Error; err != nil {
		return nil, err
	}

	return &termSheet, nil
}
