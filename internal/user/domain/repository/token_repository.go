package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
)

// TokenRepository defines the interface for token data access
type TokenRepository interface {
	// Create saves a new verification token
	Create(ctx context.Context, t *token.VerificationToken) error

	// FindByToken retrieves a token by its token string
	FindByToken(ctx context.Context, tokenStr string) (*token.VerificationToken, error)

	// MarkAsUsed marks a token as used
	MarkAsUsed(ctx context.Context, tokenID uint) error

	// DeleteUserTokens deletes all tokens of a specific type for a user
	DeleteUserTokens(ctx context.Context, userID uint, tokenType token.TokenType) error

	// DeleteExpiredTokens deletes all expired tokens (for cleanup)
	DeleteExpiredTokens(ctx context.Context) error

	// FindLatestByUserAndType finds the latest token for a user of a specific type
	FindLatestByUserAndType(ctx context.Context, userID uint, tokenType token.TokenType) (*token.VerificationToken, error)
}
