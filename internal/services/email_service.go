package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/ukuvago/angel-platform/internal/config"
	"github.com/ukuvago/angel-platform/internal/models"
)

type EmailService struct {
	config *config.Config
}

func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{config: cfg}
}

// EmailData contains common email template data
type EmailData struct {
	AppName     string
	AppURL      string
	UserName    string
	UserEmail   string
	Subject     string
	Content     template.HTML
	ActionURL   string
	ActionLabel string
}

// BaseEmailTemplate is the base HTML email template
const BaseEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Subject}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .button { display: inline-block; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 12px 30px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
        .footer { text-align: center; color: #888; font-size: 12px; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.AppName}}</h1>
        </div>
        <div class="content">
            <p>Hi {{.UserName}},</p>
            {{.Content}}
            {{if .ActionURL}}
            <p style="text-align: center;">
                <a href="{{.ActionURL}}" class="button">{{.ActionLabel}}</a>
            </p>
            {{end}}
        </div>
        <div class="footer">
            <p>&copy; {{.AppName}}. All rights reserved.</p>
            <p>This is an automated message. Please do not reply.</p>
        </div>
    </div>
</body>
</html>
`

// sendEmail sends an email using SMTP
func (s *EmailService) sendEmail(to, subject, body string) error {
	if s.config.SMTPHost == "" {
		// Log email instead of sending in development
		fmt.Printf("\n=== EMAIL ===\nTo: %s\nSubject: %s\nBody: %s\n=============\n", to, subject, body)
		return nil
	}

	from := s.config.FromEmail
	auth := smtp.PlainAuth("", s.config.SMTPUser, s.config.SMTPPassword, s.config.SMTPHost)

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		from, to, subject)

	msg := []byte(headers + body)

	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

// renderEmail renders an email using the base template
func (s *EmailService) renderEmail(data EmailData) (string, error) {
	data.AppName = s.config.AppName
	data.AppURL = s.config.AppURL

	tmpl, err := template.New("email").Parse(BaseEmailTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// SendVerificationEmail sends an email verification link
func (s *EmailService) SendVerificationEmail(user *models.User) error {
	data := EmailData{
		UserName:    user.FirstName,
		UserEmail:   user.Email,
		Subject:     "Verify your email address",
		Content:     template.HTML("<p>Thank you for registering with " + s.config.AppName + ". Please click the button below to verify your email address.</p>"),
		ActionURL:   fmt.Sprintf("%s/verify-email?token=%s", s.config.AppURL, user.VerifyToken),
		ActionLabel: "Verify Email",
	}

	body, err := s.renderEmail(data)
	if err != nil {
		return err
	}

	return s.sendEmail(user.Email, data.Subject, body)
}

// SendPasswordResetEmail sends a password reset link
func (s *EmailService) SendPasswordResetEmail(user *models.User, token string) error {
	data := EmailData{
		UserName:    user.FirstName,
		UserEmail:   user.Email,
		Subject:     "Reset your password",
		Content:     template.HTML("<p>You requested a password reset. Click the button below to reset your password. This link will expire in 24 hours.</p>"),
		ActionURL:   fmt.Sprintf("%s/reset-password?token=%s", s.config.AppURL, token),
		ActionLabel: "Reset Password",
	}

	body, err := s.renderEmail(data)
	if err != nil {
		return err
	}

	return s.sendEmail(user.Email, data.Subject, body)
}

// SendOfferNotification notifies a developer of a new investment offer
func (s *EmailService) SendOfferNotification(developer *models.User, investor *models.User, offer *models.InvestmentOffer, project *models.Project) error {
	content := fmt.Sprintf(`
		<p>Great news! You have received a new investment offer for your project <strong>%s</strong>.</p>
		<p><strong>Offer Details:</strong></p>
		<ul>
			<li>Investor: %s</li>
			<li>Amount: $%.2f</li>
		</ul>
		<p>Log in to your dashboard to review and respond to this offer.</p>
	`, project.Title, investor.FullName(), offer.OfferAmount)

	data := EmailData{
		UserName:    developer.FirstName,
		UserEmail:   developer.Email,
		Subject:     fmt.Sprintf("New Investment Offer for %s", project.Title),
		Content:     template.HTML(content),
		ActionURL:   fmt.Sprintf("%s/developer/offers", s.config.AppURL),
		ActionLabel: "View Offer",
	}

	body, err := s.renderEmail(data)
	if err != nil {
		return err
	}

	return s.sendEmail(developer.Email, data.Subject, body)
}

// SendOfferResponseNotification notifies an investor of offer response
func (s *EmailService) SendOfferResponseNotification(investor *models.User, offer *models.InvestmentOffer, project *models.Project, accepted bool) error {
	status := "accepted"
	action := "You can now proceed to sign the term sheet."
	if !accepted {
		status = "declined"
		action = "You may continue exploring other investment opportunities on our platform."
	}

	content := fmt.Sprintf(`
		<p>Your investment offer for <strong>%s</strong> has been <strong>%s</strong>.</p>
		<p>%s</p>
	`, project.Title, status, action)

	data := EmailData{
		UserName:    investor.FirstName,
		UserEmail:   investor.Email,
		Subject:     fmt.Sprintf("Your offer for %s has been %s", project.Title, status),
		Content:     template.HTML(content),
		ActionURL:   fmt.Sprintf("%s/investor/offers", s.config.AppURL),
		ActionLabel: "View Details",
	}

	body, err := s.renderEmail(data)
	if err != nil {
		return err
	}

	return s.sendEmail(investor.Email, data.Subject, body)
}

// SendProjectApprovalNotification notifies a developer of project approval
func (s *EmailService) SendProjectApprovalNotification(developer *models.User, project *models.Project, approved bool) error {
	status := "approved"
	content := fmt.Sprintf("<p>Congratulations! Your project <strong>%s</strong> has been approved and is now visible to investors.</p>", project.Title)
	
	if !approved {
		status = "requires changes"
		content = fmt.Sprintf(`
			<p>Your project <strong>%s</strong> requires some changes before it can be published.</p>
			<p><strong>Feedback:</strong></p>
			<p>%s</p>
			<p>Please update your project and resubmit for review.</p>
		`, project.Title, project.RejectionReason)
	}

	data := EmailData{
		UserName:    developer.FirstName,
		UserEmail:   developer.Email,
		Subject:     fmt.Sprintf("Your project has been %s", status),
		Content:     template.HTML(content),
		ActionURL:   fmt.Sprintf("%s/developer/projects", s.config.AppURL),
		ActionLabel: "View Project",
	}

	body, err := s.renderEmail(data)
	if err != nil {
		return err
	}

	return s.sendEmail(developer.Email, data.Subject, body)
}

// SendTermSheetSignedNotification notifies when a term sheet is fully signed
func (s *EmailService) SendTermSheetSignedNotification(recipient *models.User, project *models.Project) error {
	content := fmt.Sprintf(`
		<p>The SAFE term sheet for <strong>%s</strong> has been fully signed by both parties.</p>
		<p>Congratulations on completing this investment agreement!</p>
		<p>You can download the signed document from your dashboard.</p>
	`, project.Title)

	data := EmailData{
		UserName:    recipient.FirstName,
		UserEmail:   recipient.Email,
		Subject:     fmt.Sprintf("SAFE Agreement Completed for %s", project.Title),
		Content:     template.HTML(content),
		ActionURL:   fmt.Sprintf("%s/termsheets", s.config.AppURL),
		ActionLabel: "View Term Sheet",
	}

	body, err := s.renderEmail(data)
	if err != nil {
		return err
	}

	return s.sendEmail(recipient.Email, data.Subject, body)
}
