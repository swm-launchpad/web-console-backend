package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"path/filepath"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

// Service defines the interface for email operations
type Service interface {
	// SendVerificationEmail sends an email verification link to the user
	SendVerificationEmail(ctx context.Context, email, username, token string) error

	// SendPasswordResetEmail sends a password reset link to the user
	SendPasswordResetEmail(ctx context.Context, email, username, token string) error
}

// service is the concrete implementation of Service
type service struct {
	host        string
	port        int
	username    string
	password    string
	from        string
	frontendURL string
	templates   *template.Template
	logger      logger.Logger
}

// NewService creates a new email service
func NewService(host string, port int, username, password, from, frontendURL string, log logger.Logger) Service {
	// Load email templates
	templatesPath := filepath.Join("templates", "email", "*.html")
	tmpl, err := template.ParseGlob(templatesPath)
	if err != nil {
		// Log error but don't fail - we can fall back to embedded templates
		log.Error(context.Background(), "failed to load email templates",
			zap.String("templates_path", templatesPath),
			zap.Error(err),
		)
	}

	return &service{
		host:        host,
		port:        port,
		username:    username,
		password:    password,
		from:        from,
		frontendURL: frontendURL,
		templates:   tmpl,
		logger:      log,
	}
}

// SendVerificationEmail sends an email verification link
func (s *service) SendVerificationEmail(ctx context.Context, email, username, token string) error {
	s.logger.Info(ctx, "email service send verification email started",
		zap.String("email", email),
		zap.String("username", username),
	)

	subject := "이메일 인증 - Launchpad Web Console"

	// Create email body from template
	body, err := s.renderTemplate("verification.html", map[string]string{
		"Username":    username,
		"Token":       token,
		"FrontendURL": s.frontendURL,
	})
	if err != nil {
		s.logger.Error(ctx, "email service failed to render verification template",
			zap.String("email", email),
			zap.Error(err),
		)
		return fmt.Errorf("failed to render email template: %w", err)
	}

	if err := s.sendEmail(email, subject, body); err != nil {
		s.logger.Error(ctx, "email service failed to send verification email",
			zap.String("email", email),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info(ctx, "email service send verification email completed",
		zap.String("email", email),
	)
	return nil
}

// SendPasswordResetEmail sends a password reset link
func (s *service) SendPasswordResetEmail(ctx context.Context, email, username, token string) error {
	s.logger.Info(ctx, "email service send password reset email started",
		zap.String("email", email),
		zap.String("username", username),
	)

	subject := "비밀번호 재설정 - Launchpad Web Console"

	// Create email body from template
	body, err := s.renderTemplate("password_reset.html", map[string]string{
		"Username":    username,
		"Token":       token,
		"FrontendURL": s.frontendURL,
	})
	if err != nil {
		s.logger.Error(ctx, "email service failed to render password reset template",
			zap.String("email", email),
			zap.Error(err),
		)
		return fmt.Errorf("failed to render email template: %w", err)
	}

	if err := s.sendEmail(email, subject, body); err != nil {
		s.logger.Error(ctx, "email service failed to send password reset email",
			zap.String("email", email),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info(ctx, "email service send password reset email completed",
		zap.String("email", email),
	)
	return nil
}

// sendEmail sends an email using SMTP
func (s *service) sendEmail(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	d := gomail.NewDialer(s.host, s.port, s.username, s.password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// renderTemplate renders an email template with the given data
func (s *service) renderTemplate(templateName string, data map[string]string) (string, error) {
	if s.templates == nil {
		return "", fmt.Errorf("email templates not loaded: template files must be present in templates/email directory")
	}

	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", templateName, err)
	}

	return buf.String(), nil
}
