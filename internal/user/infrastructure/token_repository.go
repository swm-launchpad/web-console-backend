package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure/sqlc"
)

type tokenRepository struct {
	queries *sqlc.Queries
}

// NewTokenRepository creates a new token repository instance
func NewTokenRepository(db sqlc.DBTX) repository.TokenRepository {
	return &tokenRepository{
		queries: sqlc.New(db),
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
		return usererrors.ErrDatabaseOperation
	}

	return nil
}

// FindByToken retrieves a token by its token string
func (r *tokenRepository) FindByToken(ctx context.Context, tokenStr string) (*token.VerificationToken, error) {
	queries := r.getQueries(ctx)
	sqlcToken, err := queries.FindTokenByToken(ctx, tokenStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usererrors.ErrTokenNotFound
		}
		return nil, usererrors.ErrDatabaseOperation
	}

	return mapSQLCTokenToDomainToken(sqlcToken), nil
}

// MarkAsUsed marks a token as used
func (r *tokenRepository) MarkAsUsed(ctx context.Context, tokenID uint) error {
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
		return usererrors.ErrDatabaseOperation
	}

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
