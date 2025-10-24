package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure/sqlc"
	"go.uber.org/zap"
)

type tokenRepository struct {
	queries *sqlc.Queries
	logger  logger.Logger
}

// NewTokenRepository creates a new token repository instance
func NewTokenRepository(db sqlc.DBTX, log logger.Logger) repository.TokenRepository {
	return &tokenRepository{
		queries: sqlc.New(db),
		logger:  log,
	}
}

// getQueries returns the appropriate Queries instance for the context
// If there's a transaction in the context, it returns queries with that transaction
func (r *tokenRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, ok := db.GetTx(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

// Create saves a new verification token
func (r *tokenRepository) Create(ctx context.Context, t *token.VerificationToken) error {
	r.logger.Info(ctx, "token repository create started",
		zap.Uint("user_id", t.UserID),
		zap.String("token_type", string(t.TokenType)),
	)

	params := sqlc.CreateVerificationTokenParams{
		UserID:    uint32(t.UserID),
		Token:     t.Token,
		TokenType: mapTokenTypeToSQLCTokenType(t.TokenType),
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
	}

	queries := r.getQueries(ctx)
	err := queries.CreateVerificationToken(ctx, params)
	if err != nil {
		r.logger.Error(ctx, "token repository create failed",
			zap.Uint("user_id", t.UserID),
			zap.Error(err),
		)
		return usererrors.ErrDatabaseOperation
	}

	r.logger.Info(ctx, "token repository create completed",
		zap.Uint("user_id", t.UserID),
		zap.String("token_type", string(t.TokenType)),
	)
	return nil
}

// FindByToken retrieves a token by its token string
func (r *tokenRepository) FindByToken(ctx context.Context, tokenStr string) (*token.VerificationToken, error) {
	r.logger.Info(ctx, "token repository find by token started")

	queries := r.getQueries(ctx)
	sqlcToken, err := queries.FindTokenByToken(ctx, tokenStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "token not found", zap.Error(usererrors.ErrTokenNotFound))
			return nil, usererrors.ErrTokenNotFound
		}
		r.logger.Error(ctx, "token repository find by token failed", zap.Error(err))
		return nil, usererrors.ErrDatabaseOperation
	}

	domainToken := mapSQLCTokenToDomainToken(sqlcToken)
	r.logger.Info(ctx, "token repository find by token completed",
		zap.Uint("token_id", domainToken.TokenID),
		zap.Uint("user_id", domainToken.UserID),
	)
	return domainToken, nil
}

// MarkAsUsed marks a token as used
func (r *tokenRepository) MarkAsUsed(ctx context.Context, tokenID uint) error {
	r.logger.Info(ctx, "token repository mark as used started",
		zap.Uint("token_id", tokenID),
	)

	// Set current time
	now := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	params := sqlc.MarkTokenAsUsedParams{
		TokenID: uint32(tokenID),
		UsedAt:  now,
	}

	queries := r.getQueries(ctx)
	err := queries.MarkTokenAsUsed(ctx, params)
	if err != nil {
		r.logger.Error(ctx, "token repository mark as used failed",
			zap.Uint("token_id", tokenID),
			zap.Error(err),
		)
		return usererrors.ErrDatabaseOperation
	}

	r.logger.Info(ctx, "token repository mark as used completed",
		zap.Uint("token_id", tokenID),
	)
	return nil
}

// DeleteUserTokens deletes all tokens of a specific type for a user
func (r *tokenRepository) DeleteUserTokens(ctx context.Context, userID uint, tokenType token.TokenType) error {
	params := sqlc.DeleteUserTokensByTypeParams{
		UserID:    uint32(userID),
		TokenType: mapTokenTypeToSQLCTokenType(tokenType),
	}

	queries := r.getQueries(ctx)
	err := queries.DeleteUserTokensByType(ctx, params)
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	return nil
}

// DeleteExpiredTokens deletes all expired tokens
func (r *tokenRepository) DeleteExpiredTokens(ctx context.Context) error {
	queries := r.getQueries(ctx)
	err := queries.DeleteExpiredTokens(ctx)
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	return nil
}

// FindLatestByUserAndType finds the latest token for a user of a specific type
func (r *tokenRepository) FindLatestByUserAndType(ctx context.Context, userID uint, tokenType token.TokenType) (*token.VerificationToken, error) {
	params := sqlc.FindLatestTokenByUserAndTypeParams{
		UserID:    uint32(userID),
		TokenType: mapTokenTypeToSQLCTokenType(tokenType),
	}

	queries := r.getQueries(ctx)
	sqlcToken, err := queries.FindLatestTokenByUserAndType(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usererrors.ErrTokenNotFound
		}
		return nil, usererrors.ErrDatabaseOperation
	}

	return mapSQLCTokenToDomainToken(sqlcToken), nil
}

// Helper functions to map between domain and SQLC types

func mapTokenTypeToSQLCTokenType(t token.TokenType) sqlc.VerificationTokensTokenType {
	switch t {
	case token.TokenTypeEmailVerification:
		return sqlc.VerificationTokensTokenTypeEmailVerification
	case token.TokenTypePasswordReset:
		return sqlc.VerificationTokensTokenTypePasswordReset
	default:
		return sqlc.VerificationTokensTokenTypeEmailVerification
	}
}

func mapSQLCTokenTypeToDomainTokenType(t sqlc.VerificationTokensTokenType) token.TokenType {
	switch t {
	case sqlc.VerificationTokensTokenTypeEmailVerification:
		return token.TokenTypeEmailVerification
	case sqlc.VerificationTokensTokenTypePasswordReset:
		return token.TokenTypePasswordReset
	default:
		return token.TokenTypeEmailVerification
	}
}

func mapSQLCTokenToDomainToken(sqlcToken sqlc.VerificationToken) *token.VerificationToken {
	var usedAt *time.Time
	if sqlcToken.UsedAt.Valid {
		usedAt = &sqlcToken.UsedAt.Time
	}

	return &token.VerificationToken{
		TokenID:   uint(sqlcToken.TokenID),
		UserID:    uint(sqlcToken.UserID),
		Token:     sqlcToken.Token,
		TokenType: mapSQLCTokenTypeToDomainTokenType(sqlcToken.TokenType),
		ExpiresAt: sqlcToken.ExpiresAt,
		UsedAt:    usedAt,
		CreatedAt: sqlcToken.CreatedAt,
	}
}
