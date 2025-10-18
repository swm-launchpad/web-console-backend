package state

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidStateFormat = errors.New("invalid state format")
	ErrInvalidSignature   = errors.New("invalid state signature")
	ErrStateExpired       = errors.New("state has expired")
)

// StateValidator handles HMAC-based state validation for OAuth flows
type StateValidator struct {
	secret string
}

// NewStateValidator creates a new state validator with the given secret
func NewStateValidator(secret string) *StateValidator {
	return &StateValidator{
		secret: secret,
	}
}

// GenerateState creates a signed state string for CSRF protection
// Format: base64(random_bytes):timestamp:userID:hmac
// The state is valid for 10 minutes
func (v *StateValidator) GenerateState(userID uint) (string, error) {
	// Generate random bytes
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randomStr := base64.URLEncoding.EncodeToString(randomBytes)

	// Add timestamp for expiration check
	timestamp := time.Now().Unix()

	// Create payload
	payload := fmt.Sprintf("%s:%d:%d", randomStr, timestamp, userID)

	// Generate HMAC signature
	signature := v.sign(payload)

	// Combine payload and signature
	state := fmt.Sprintf("%s:%s", payload, signature)

	return state, nil
}

// ValidateState verifies the state signature and extracts the user ID
// Returns the user ID if valid, or an error if invalid/expired
func (v *StateValidator) ValidateState(state string) (uint, error) {
	// Split state into parts
	parts := strings.Split(state, ":")
	if len(parts) != 4 {
		return 0, ErrInvalidStateFormat
	}

	randomStr := parts[0]
	timestampStr := parts[1]
	userIDStr := parts[2]
	receivedSignature := parts[3]

	// Reconstruct payload
	payload := fmt.Sprintf("%s:%s:%s", randomStr, timestampStr, userIDStr)

	// Verify signature
	expectedSignature := v.sign(payload)
	if !hmac.Equal([]byte(receivedSignature), []byte(expectedSignature)) {
		return 0, ErrInvalidSignature
	}

	// Check expiration (10 minutes)
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return 0, ErrInvalidStateFormat
	}

	expirationTime := time.Unix(timestamp, 0).Add(10 * time.Minute)
	if time.Now().After(expirationTime) {
		return 0, ErrStateExpired
	}

	// Parse user ID
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return 0, ErrInvalidStateFormat
	}

	return uint(userID), nil
}

// sign generates HMAC-SHA256 signature for the given payload
func (v *StateValidator) sign(payload string) string {
	h := hmac.New(sha256.New, []byte(v.secret))
	h.Write([]byte(payload))
	signature := h.Sum(nil)
	return base64.URLEncoding.EncodeToString(signature)
}
