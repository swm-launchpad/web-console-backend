package token

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
)

// TokenType represents the type of verification token
type TokenType string

const (
	TokenTypeEmailVerification TokenType = "email_verification"
	TokenTypePasswordReset     TokenType = "password_reset"
)

// VerificationToken represents a verification token entity
type VerificationToken struct {
	TokenID   uint
	UserID    uint
	Token     string
	TokenType TokenType
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// NewEmailVerificationToken creates a new email verification token
// Token is valid for 24 hours
func NewEmailVerificationToken(userID uint) (*VerificationToken, error) {
	token, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	return &VerificationToken{
		UserID:    userID,
		Token:     token,
		TokenType: TokenTypeEmailVerification,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// NewPasswordResetToken creates a new password reset token
// Token is valid for 1 hour
func NewPasswordResetToken(userID uint) (*VerificationToken, error) {
	token, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	return &VerificationToken{
		UserID:    userID,
		Token:     token,
		TokenType: TokenTypePasswordReset,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// IsExpired checks if the token has expired
func (t *VerificationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed checks if the token has been used
func (t *VerificationToken) IsUsed() bool {
	return t.UsedAt != nil
}

// MarkAsUsed marks the token as used
func (t *VerificationToken) MarkAsUsed() error {
	if t.IsUsed() {
		return usererrors.ErrTokenAlreadyUsed
	}

	if t.IsExpired() {
		return usererrors.ErrTokenExpired
	}

	now := time.Now()
	t.UsedAt = &now
	return nil
}

// Validate validates the token
func (t *VerificationToken) Validate() error {
	if t.IsUsed() {
		return usererrors.ErrTokenAlreadyUsed
	}

	if t.IsExpired() {
		return usererrors.ErrTokenExpired
	}

	return nil
}

// generateSecureToken generates a cryptographically secure random token
// Returns a 64-character hex string (32 bytes)
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", usererrors.ErrTokenGenerationFailed
	}
	return hex.EncodeToString(bytes), nil
}
