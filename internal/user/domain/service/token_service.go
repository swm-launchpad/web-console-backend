package service

import (
	"context"
	"time"

	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

// TokenService defines the interface for token-related business logic
type TokenService interface {
	// CreateEmailVerificationToken creates a new email verification token
	CreateEmailVerificationToken(ctx context.Context, userID uint) (*token.VerificationToken, error)

	// CreatePasswordResetToken creates a new password reset token
	CreatePasswordResetToken(ctx context.Context, userID uint) (*token.VerificationToken, error)

	// ValidateAndUseToken validates a token and marks it as used
	ValidateAndUseToken(ctx context.Context, tokenStr string, expectedType token.TokenType) (*token.VerificationToken, error)

	// CanResendVerificationEmail checks if a verification email can be resent (rate limiting)
	CanResendVerificationEmail(ctx context.Context, userID uint) (bool, time.Duration, error)

	// InvalidateUserTokens invalidates all tokens of a specific type for a user
	InvalidateUserTokens(ctx context.Context, userID uint, tokenType token.TokenType) error
}

// tokenService is the concrete implementation of TokenService
type tokenService struct {
	tokenRepo repository.TokenRepository
}

// NewTokenService creates a new instance of TokenService
func NewTokenService(tokenRepo repository.TokenRepository) TokenService {
	return &tokenService{
		tokenRepo: tokenRepo,
	}
}

// CreateEmailVerificationToken creates a new email verification token
func (s *tokenService) CreateEmailVerificationToken(ctx context.Context, userID uint) (*token.VerificationToken, error) {
	if userID == 0 {
		return nil, usererrors.ErrInvalidUserID
	}

	// Create new token
	verificationToken, err := token.NewEmailVerificationToken(userID)
	if err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.tokenRepo.Create(ctx, verificationToken); err != nil {
		return nil, err
	}

	return verificationToken, nil
}

// CreatePasswordResetToken creates a new password reset token
func (s *tokenService) CreatePasswordResetToken(ctx context.Context, userID uint) (*token.VerificationToken, error) {
	if userID == 0 {
		return nil, usererrors.ErrInvalidUserID
	}

	// Delete old password reset tokens for this user
	if err := s.tokenRepo.DeleteUserTokens(ctx, userID, token.TokenTypePasswordReset); err != nil {
		return nil, err
	}

	// Create new token
	resetToken, err := token.NewPasswordResetToken(userID)
	if err != nil {
		return nil, err
	}

	// Save to repository
	if err := s.tokenRepo.Create(ctx, resetToken); err != nil {
		return nil, err
	}

	return resetToken, nil
}

// ValidateAndUseToken validates a token and marks it as used
func (s *tokenService) ValidateAndUseToken(ctx context.Context, tokenStr string, expectedType token.TokenType) (*token.VerificationToken, error) {
	if tokenStr == "" {
		return nil, usererrors.ErrTokenInvalid
	}

	// Find token
	verificationToken, err := s.tokenRepo.FindByToken(ctx, tokenStr)
	if err != nil {
		return nil, err
	}

	// Check token type
	if verificationToken.TokenType != expectedType {
		return nil, usererrors.ErrTokenInvalid
	}

	// Validate token (checks expiry and usage)
	if err := verificationToken.Validate(); err != nil {
		return nil, err
	}

	// Mark as used
	if err := verificationToken.MarkAsUsed(); err != nil {
		return nil, err
	}

	// Update in repository
	if err := s.tokenRepo.MarkAsUsed(ctx, verificationToken.TokenID); err != nil {
		return nil, err
	}

	return verificationToken, nil
}

// CanResendVerificationEmail checks if a verification email can be resent
// Returns: (canResend, waitDuration, error)
func (s *tokenService) CanResendVerificationEmail(ctx context.Context, userID uint) (bool, time.Duration, error) {
	if userID == 0 {
		return false, 0, usererrors.ErrInvalidUserID
	}

	// Find latest email verification token for this user
	latestToken, err := s.tokenRepo.FindLatestByUserAndType(ctx, userID, token.TokenTypeEmailVerification)
	if err != nil {
		// If no token found, can resend
		if err == usererrors.ErrTokenNotFound {
			return true, 0, nil
		}
		return false, 0, err
	}

	// Check if 5 minutes have passed since last token creation
	const resendInterval = 5 * time.Minute
	timeSinceCreation := time.Since(latestToken.CreatedAt)

	if timeSinceCreation < resendInterval {
		waitDuration := resendInterval - timeSinceCreation
		return false, waitDuration, nil
	}

	return true, 0, nil
}

// InvalidateUserTokens invalidates all tokens of a specific type for a user
func (s *tokenService) InvalidateUserTokens(ctx context.Context, userID uint, tokenType token.TokenType) error {
	if userID == 0 {
		return usererrors.ErrInvalidUserID
	}

	return s.tokenRepo.DeleteUserTokens(ctx, userID, tokenType)
}
