package email

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
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
func (s *service) sendEmail(to, subject, body string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(s.host, s.port, s.username, s.password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// renderTemplate renders an email template with the given data
func (s *service) renderTemplate(templateName string, data map[string]string) (string, error) {
	// Try to render from loaded templates first
	if s.templates != nil {
		var buf bytes.Buffer
		if err := s.templates.ExecuteTemplate(&buf, templateName, data); err == nil {
			return buf.String(), nil
		}
	}

	// Fallback to embedded templates if file-based templates are not available
	var tmplStr string
	switch templateName {
	case "verification.html":
		tmplStr = getVerificationTemplateEmbedded()
	case "password_reset.html":
		tmplStr = getPasswordResetTemplateEmbedded()
	default:
		return "", fmt.Errorf("unknown template: %s", templateName)
	}

	tmpl, err := template.New(templateName).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse embedded template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute embedded template: %w", err)
	}

	return buf.String(), nil
}

// getVerificationTemplateEmbedded returns the embedded verification email template
func getVerificationTemplateEmbedded() string {
	// Read from file if available
	content, err := os.ReadFile(filepath.Join("templates", "email", "verification.html"))
	if err == nil {
		return string(content)
	}

	// Simple fallback template
	return `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
	<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
		<h2>이메일 인증 - Launchpad Web Console</h2>
		<p>안녕하세요, {{.Username}}님!</p>
		<p>Launchpad Web Console에 가입해 주셔서 감사합니다.</p>
		<p>아래 링크를 클릭하여 이메일 주소를 인증해주세요:</p>
		<p><a href="{{.FrontendURL}}/verify-email?token={{.Token}}" style="display: inline-block; padding: 10px 20px; background-color: #667eea; color: white; text-decoration: none; border-radius: 5px;">이메일 인증하기</a></p>
		<p>이 링크는 24시간 동안 유효합니다.</p>
		<hr>
		<p style="font-size: 12px; color: #666;">© 2025 Launchpad Web Console</p>
	</div>
</body>
</html>
`
}

// getPasswordResetTemplateEmbedded returns the embedded password reset email template
func getPasswordResetTemplateEmbedded() string {
	// Read from file if available
	content, err := os.ReadFile(filepath.Join("templates", "email", "password_reset.html"))
	if err == nil {
		return string(content)
	}

	// Simple fallback template
	return `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
	<div style="max-width: 600px; margin: 0 auto; padding: 20px;">
		<h2>비밀번호 재설정 - Launchpad Web Console</h2>
		<p>안녕하세요, {{.Username}}님!</p>
		<p>비밀번호 재설정 요청을 받았습니다.</p>
		<p>아래 링크를 클릭하여 새로운 비밀번호를 설정해주세요:</p>
		<p><a href="{{.FrontendURL}}/reset-password?token={{.Token}}" style="display: inline-block; padding: 10px 20px; background-color: #f5576c; color: white; text-decoration: none; border-radius: 5px;">비밀번호 재설정하기</a></p>
		<p>이 링크는 1시간 동안 유효합니다.</p>
		<p><strong>본인이 요청하지 않은 경우</strong>, 이 이메일을 무시하셔도 됩니다.</p>
		<hr>
		<p style="font-size: 12px; color: #666;">© 2025 Launchpad Web Console</p>
	</div>
</body>
</html>
`
}
