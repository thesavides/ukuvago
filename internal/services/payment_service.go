package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/ukuvago/angel-platform/internal/config"
	"github.com/ukuvago/angel-platform/internal/database"
	"github.com/ukuvago/angel-platform/internal/models"
)

type PaymentService struct {
	config *config.Config
}

func NewPaymentService(cfg *config.Config) *PaymentService {
	if cfg.StripeSecretKey != "" {
		stripe.Key = cfg.StripeSecretKey
	}
	return &PaymentService{config: cfg}
}

// CreatePaymentIntent creates a Stripe payment intent for the view fee
func (s *PaymentService) CreatePaymentIntent(investorID uuid.UUID) (*models.Payment, string, error) {
	db := database.GetDB()

	// Check if investor has an active payment with remaining views
	var existingPayment models.Payment
	err := db.Where("investor_id = ? AND status = ? AND projects_remaining > 0",
		investorID, models.PaymentStatusCompleted).First(&existingPayment).Error
	if err == nil {
		return nil, "", errors.New("you already have an active payment with remaining project views")
	}

	// Create payment record
	payment := &models.Payment{
		InvestorID:        investorID,
		Amount:            s.config.ViewFeeAmount,
		Currency:          s.config.ViewFeeCurrency,
		Status:            models.PaymentStatusPending,
		ProjectsRemaining: s.config.MaxProjectViews,
		ProjectsTotal:     s.config.MaxProjectViews,
		Description:       "Project viewing fee - access to view up to 4 projects",
	}

	if err := db.Create(payment).Error; err != nil {
		return nil, "", err
	}

	// Create Stripe payment intent if configured
	var clientSecret string
	if s.config.StripeSecretKey != "" {
		params := &stripe.PaymentIntentParams{
			Amount:   stripe.Int64(payment.Amount),
			Currency: stripe.String(payment.Currency),
			Metadata: map[string]string{
				"payment_id":  payment.ID.String(),
				"investor_id": investorID.String(),
			},
			AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
				Enabled: stripe.Bool(true),
			},
		}

		pi, err := paymentintent.New(params)
		if err != nil {
			// Rollback payment creation
			db.Delete(payment)
			return nil, "", err
		}

		payment.StripePaymentID = pi.ID
		payment.StripeClientSecret = pi.ClientSecret
		clientSecret = pi.ClientSecret

		if err := db.Save(payment).Error; err != nil {
			return nil, "", err
		}
	} else {
		// Demo mode - no Stripe configured
		clientSecret = "demo_mode"
	}

	return payment, clientSecret, nil
}

// ConfirmPayment confirms a payment has been completed
func (s *PaymentService) ConfirmPayment(paymentID uuid.UUID, stripePaymentID string) (*models.Payment, error) {
	db := database.GetDB()

	var payment models.Payment
	if err := db.First(&payment, "id = ?", paymentID).Error; err != nil {
		return nil, errors.New("payment not found")
	}

	if payment.Status != models.PaymentStatusPending {
		return nil, errors.New("payment already processed")
	}

	// Verify with Stripe if configured
	if s.config.StripeSecretKey != "" && stripePaymentID != "" {
		pi, err := paymentintent.Get(stripePaymentID, nil)
		if err != nil {
			return nil, err
		}

		if pi.Status != stripe.PaymentIntentStatusSucceeded {
			return nil, errors.New("payment not successful")
		}

		payment.ReceiptURL = string(pi.LatestCharge.ReceiptURL)
	}

	now := time.Now()
	payment.Status = models.PaymentStatusCompleted
	payment.CompletedAt = &now

	if err := db.Save(&payment).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

// DemoConfirmPayment confirms payment in demo mode (no Stripe)
func (s *PaymentService) DemoConfirmPayment(paymentID uuid.UUID) (*models.Payment, error) {
	db := database.GetDB()

	var payment models.Payment
	if err := db.First(&payment, "id = ?", paymentID).Error; err != nil {
		return nil, errors.New("payment not found")
	}

	if payment.Status != models.PaymentStatusPending {
		return nil, errors.New("payment already processed")
	}

	now := time.Now()
	payment.Status = models.PaymentStatusCompleted
	payment.CompletedAt = &now

	if err := db.Save(&payment).Error; err != nil {
		return nil, err
	}

	return &payment, nil
}

// GetActivePayment gets an investor's active payment with remaining views
func (s *PaymentService) GetActivePayment(investorID uuid.UUID) (*models.Payment, error) {
	db := database.GetDB()

	var payment models.Payment
	err := db.Where("investor_id = ? AND status = ? AND projects_remaining > 0",
		investorID, models.PaymentStatusCompleted).
		Order("created_at DESC").
		First(&payment).Error

	if err != nil {
		return nil, err
	}

	return &payment, nil
}

// UseViewCredit decrements the view credit and records the view
func (s *PaymentService) UseViewCredit(investorID, projectID uuid.UUID) error {
	db := database.GetDB()

	// Check if already viewed
	var existingView models.ProjectView
	if err := db.Where("investor_id = ? AND project_id = ?", investorID, projectID).
		First(&existingView).Error; err == nil {
		// Already viewed, no credit needed
		return nil
	}

	// Get active payment
	payment, err := s.GetActivePayment(investorID)
	if err != nil {
		return errors.New("no active payment with available views")
	}

	if !payment.CanViewMore() {
		return errors.New("no remaining project views")
	}

	// Create view record
	view := &models.ProjectView{
		InvestorID: investorID,
		ProjectID:  projectID,
		PaymentID:  payment.ID,
		ViewedAt:   time.Now(),
	}

	if err := db.Create(view).Error; err != nil {
		return err
	}

	// Decrement credit
	payment.ProjectsRemaining--
	return db.Save(payment).Error
}

// HasViewedProject checks if an investor has already viewed a project
func (s *PaymentService) HasViewedProject(investorID, projectID uuid.UUID) bool {
	db := database.GetDB()

	var view models.ProjectView
	err := db.Where("investor_id = ? AND project_id = ?", investorID, projectID).First(&view).Error
	return err == nil
}

// GetPaymentHistory retrieves payment history for an investor
func (s *PaymentService) GetPaymentHistory(investorID uuid.UUID) ([]models.Payment, error) {
	db := database.GetDB()

	var payments []models.Payment
	err := db.Where("investor_id = ?", investorID).
		Order("created_at DESC").
		Find(&payments).Error

	return payments, err
}

// GetViewedProjects retrieves projects an investor has viewed
func (s *PaymentService) GetViewedProjects(investorID uuid.UUID) ([]models.ProjectView, error) {
	db := database.GetDB()

	var views []models.ProjectView
	err := db.Where("investor_id = ?", investorID).
		Preload("Project").
		Preload("Project.Category").
		Order("viewed_at DESC").
		Find(&views).Error

	return views, err
}
