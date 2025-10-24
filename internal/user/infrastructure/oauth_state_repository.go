package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure/sqlc"
	"go.uber.org/zap"
)

type oauthStateRepository struct {
	queries *sqlc.Queries
	logger  logger.Logger
}

func NewOAuthStateRepository(db sqlc.DBTX, log logger.Logger) repository.OAuthStateRepository {
	return &oauthStateRepository{
		queries: sqlc.New(db),
		logger:  log,
	}
}

func (r *oauthStateRepository) Create(ctx context.Context, state *model.OAuthState) error {
	r.logger.Info(ctx, "oauth state repository create started",
		zap.Uint("user_id", state.UserID),
	)

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
			r.logger.Error(ctx, "oauth state already exists",
				zap.Error(usererrors.ErrValidationFailed),
			)
			return usererrors.ErrValidationFailed // State already exists
		}
		r.logger.Error(ctx, "oauth state repository create failed",
			zap.Error(err),
		)
		return usererrors.ErrDatabaseOperation
	}

	r.logger.Info(ctx, "oauth state repository create completed",
		zap.Uint("user_id", state.UserID),
	)
	return nil
}

func (r *oauthStateRepository) FindByState(ctx context.Context, state string) (*model.OAuthState, error) {
	r.logger.Info(ctx, "oauth state repository find by state started")

	row, err := r.queriesWithContext(ctx).GetOAuthStateByState(ctx, state)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "oauth state not found",
				zap.Error(usererrors.ErrInvalidState),
			)
			return nil, usererrors.ErrInvalidState
		}
		r.logger.Error(ctx, "oauth state repository find by state failed",
			zap.Error(err),
		)
		return nil, usererrors.ErrDatabaseUnavailable
	}

	oauthState := &model.OAuthState{
		State:          row.State,
		UserID:         uint(row.UserID),
		InstallationID: nullInt64ToPtr(row.InstallationID),
		ExpiresAt:      row.ExpiresAt,
		CreatedAt:      row.CreatedAt,
		ConsumedAt:     nullTimeToPtr(row.ConsumedAt),
	}

	r.logger.Info(ctx, "oauth state repository find by state completed",
		zap.Uint("user_id", oauthState.UserID),
	)
	return oauthState, nil
}

func (r *oauthStateRepository) MarkAsConsumed(ctx context.Context, state string, installationID int64) error {
	r.logger.Info(ctx, "oauth state repository mark as consumed started",
		zap.Int64("installation_id", installationID),
	)

	now := time.Now()
	params := sqlc.MarkOAuthStateAsConsumedParams{
		ConsumedAt:     toNullTime(&now),
		InstallationID: toNullInt64(&installationID),
		State:          state,
	}

	result, err := r.queriesWithContext(ctx).MarkOAuthStateAsConsumed(ctx, params)
	if err != nil {
		r.logger.Error(ctx, "oauth state repository mark as consumed failed",
			zap.Error(err),
		)
		return usererrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error(ctx, "oauth state repository mark as consumed rows affected check failed",
			zap.Error(err),
		)
		return usererrors.ErrDatabaseOperation
	}

	if rowsAffected == 0 {
		r.logger.Error(ctx, "oauth state already consumed or not found",
			zap.Error(usererrors.ErrInvalidState),
		)
		return usererrors.ErrInvalidState // Already consumed or doesn't exist
	}

	r.logger.Info(ctx, "oauth state repository mark as consumed completed",
		zap.Int64("installation_id", installationID),
	)
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
