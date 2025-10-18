package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure/sqlc"
)

type oauthStateRepository struct {
	queries *sqlc.Queries
}

func NewOAuthStateRepository(db sqlc.DBTX) repository.OAuthStateRepository {
	return &oauthStateRepository{
		queries: sqlc.New(db),
	}
}

func (r *oauthStateRepository) Create(ctx context.Context, state *model.OAuthState) error {
	params := sqlc.CreateOAuthStateParams{
		State:          state.State,
		UserID:         uint32(state.UserID),
		InstallationID: toNullInt64(state.InstallationID),
		ExpiresAt:      state.ExpiresAt,
		CreatedAt:      state.CreatedAt,
		ConsumedAt:     toNullTime(state.ConsumedAt),
	}

	_, err := r.queriesWithContext(ctx).CreateOAuthState(ctx, params)
	if err != nil {
		if isDuplicateError(err) {
			return usererrors.ErrValidationFailed // State already exists
		}
		return usererrors.ErrDatabaseOperation
	}

	return nil
}

func (r *oauthStateRepository) FindByState(ctx context.Context, state string) (*model.OAuthState, error) {
	row, err := r.queriesWithContext(ctx).GetOAuthStateByState(ctx, state)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usererrors.ErrInvalidState
		}
		return nil, usererrors.ErrDatabaseUnavailable
	}

	return &model.OAuthState{
		State:          row.State,
		UserID:         uint(row.UserID),
		InstallationID: nullInt64ToPtr(row.InstallationID),
		ExpiresAt:      row.ExpiresAt,
		CreatedAt:      row.CreatedAt,
		ConsumedAt:     nullTimeToPtr(row.ConsumedAt),
	}, nil
}

func (r *oauthStateRepository) MarkAsConsumed(ctx context.Context, state string, installationID int64) error {
	now := time.Now()
	params := sqlc.MarkOAuthStateAsConsumedParams{
		ConsumedAt:     toNullTime(&now),
		InstallationID: toNullInt64(&installationID),
		State:          state,
	}

	result, err := r.queriesWithContext(ctx).MarkOAuthStateAsConsumed(ctx, params)
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	if rowsAffected == 0 {
		return usererrors.ErrInvalidState // Already consumed or doesn't exist
	}

	return nil
}

func (r *oauthStateRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.queriesWithContext(ctx).DeleteExpiredOAuthStates(ctx)
	if err != nil {
		return 0, usererrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, usererrors.ErrDatabaseOperation
	}

	return rowsAffected, nil
}

func (r *oauthStateRepository) queriesWithContext(ctx context.Context) *sqlc.Queries {
	if tx, ok := db.GetTx(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

// Helper functions
func toNullInt64(ptr *int64) sql.NullInt64 {
	if ptr == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *ptr, Valid: true}
}

func nullInt64ToPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}
