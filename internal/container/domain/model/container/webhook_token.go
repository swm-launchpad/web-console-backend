package model

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"

	"github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// WebhookToken is a value object representing a webhook authentication token
// Token format: 64 character hexadecimal string (32 bytes of random data)
type WebhookToken struct {
	value string
}

const (
	WebhookTokenLength = 64 // 32 bytes = 64 hex characters
)

var webhookTokenRegex = regexp.MustCompile(`^[a-f0-9]{64}$`)

// NewWebhookToken creates a new WebhookToken from a string value
func NewWebhookToken(value string) (*WebhookToken, error) {
	if !webhookTokenRegex.MatchString(value) {
		return nil, errors.ErrInvalidWebhookToken
	}

	return &WebhookToken{value: value}, nil
}

// GenerateWebhookToken generates a new random webhook token
func GenerateWebhookToken() (*WebhookToken, error) {
	// Generate 32 random bytes
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, err
	}

	// Convert to hexadecimal string (64 characters)
	tokenValue := hex.EncodeToString(randomBytes)

	return &WebhookToken{value: tokenValue}, nil
}

// Value returns the string value of the webhook token
func (t *WebhookToken) Value() string {
	return t.value
}

// String returns the string representation of the webhook token
func (t *WebhookToken) String() string {
	return t.value
}

// Equals checks if two webhook tokens are equal
func (t *WebhookToken) Equals(other *WebhookToken) bool {
	if other == nil {
		return false
	}
	return t.value == other.value
}
