package service

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
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
	logger    logger.Logger
}

// NewTokenService creates a new instance of TokenService
func NewTokenService(tokenRepo repository.TokenRepository, log logger.Logger) TokenService {
	return &tokenService{
		tokenRepo: tokenRepo,
		logger:    log,
	}
}

// CreateEmailVerificationToken creates a new email verification token
func (s *tokenService) CreateEmailVerificationToken(ctx context.Context, userID uint) (*token.VerificationToken, error) {
	s.logger.Info(ctx, "create email verification token started",
		zap.Uint("user_id", userID),
	)

	if userID == 0 {
		s.logger.Error(ctx, "invalid user id",
			zap.Uint("user_id", userID),
		)
		return nil, usererrors.ErrInvalidUserID
	}

	// Create new token
	verificationToken, err := token.NewEmailVerificationToken(userID)
	if err != nil {
		s.logger.Error(ctx, "failed to create email verification token",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		return nil, err
	}

	// Save to repository
	if err := s.tokenRepo.Create(ctx, verificationToken); err != nil {
		s.logger.Error(ctx, "failed to save email verification token",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		return nil, err
	}

	s.logger.Info(ctx, "create email verification token completed",
		zap.Uint("user_id", userID),
		zap.Uint("token_id", verificationToken.TokenID),
	)

	return verificationToken, nil
}

// CreatePasswordResetToken creates a new password reset token
func (s *tokenService) CreatePasswordResetToken(ctx context.Context, userID uint) (*token.VerificationToken, error) {
	s.logger.Info(ctx, "create password reset token started",
		zap.Uint("user_id", userID),
	)

	if userID == 0 {
		s.logger.Error(ctx, "invalid user id",
			zap.Uint("user_id", userID),
		)
		return nil, usererrors.ErrInvalidUserID
	}

	// Delete old password reset tokens for this user
	if err := s.tokenRepo.DeleteUserTokens(ctx, userID, token.TokenTypePasswordReset); err != nil {
		s.logger.Error(ctx, "failed to delete old password reset tokens",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		return nil, err
	}

	// Create new token
	resetToken, err := token.NewPasswordResetToken(userID)
	if err != nil {
		s.logger.Error(ctx, "failed to create password reset token",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		return nil, err
	}

	// Save to repository
	if err := s.tokenRepo.Create(ctx, resetToken); err != nil {
		s.logger.Error(ctx, "failed to save password reset token",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		return nil, err
	}

	s.logger.Info(ctx, "create password reset token completed",
		zap.Uint("user_id", userID),
		zap.Uint("token_id", resetToken.TokenID),
	)

	return resetToken, nil
}

// ValidateAndUseToken validates a token and marks it as used
func (s *tokenService) ValidateAndUseToken(ctx context.Context, tokenStr string, expectedType token.TokenType) (*token.VerificationToken, error) {
	s.logger.Info(ctx, "validate and use token started",
		zap.String("token_type", string(expectedType)),
	)

	if tokenStr == "" {
		s.logger.Error(ctx, "empty token string")
		return nil, usererrors.ErrTokenInvalid
	}

	// Find token
	verificationToken, err := s.tokenRepo.FindByToken(ctx, tokenStr)
	if err != nil {
		s.logger.Error(ctx, "token not found",
			zap.Error(err),
		)
		return nil, err
	}

	// Check token type
	if verificationToken.TokenType != expectedType {
		s.logger.Error(ctx, "token type mismatch",
			zap.String("expected_type", string(expectedType)),
			zap.String("actual_type", string(verificationToken.TokenType)),
		)
		return nil, usererrors.ErrTokenInvalid
	}

	// Validate token (checks expiry and usage)
	if err := verificationToken.Validate(); err != nil {
		s.logger.Error(ctx, "token validation failed",
			zap.Error(err),
			zap.Uint("token_id", verificationToken.TokenID),
		)
		return nil, err
	}

	// Mark as used
	if err := verificationToken.MarkAsUsed(); err != nil {
		s.logger.Error(ctx, "failed to mark token as used",
			zap.Error(err),
			zap.Uint("token_id", verificationToken.TokenID),
		)
		return nil, err
	}

	// Update in repository
	if err := s.tokenRepo.MarkAsUsed(ctx, verificationToken.TokenID); err != nil {
		s.logger.Error(ctx, "failed to update token in repository",
			zap.Error(err),
			zap.Uint("token_id", verificationToken.TokenID),
		)
		return nil, err
	}

	s.logger.Info(ctx, "validate and use token completed",
		zap.Uint("token_id", verificationToken.TokenID),
		zap.Uint("user_id", verificationToken.UserID),
	)

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
