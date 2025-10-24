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

type githubInstallationRepository struct {
	queries *sqlc.Queries
	logger  logger.Logger
}

func NewGitHubInstallationRepository(db sqlc.DBTX, log logger.Logger) repository.GitHubInstallationRepository {
	return &githubInstallationRepository{
		queries: sqlc.New(db),
		logger:  log,
	}
}

func (r *githubInstallationRepository) Create(ctx context.Context, installation *model.GitHubInstallation) error {
	r.logger.Info(ctx, "github installation repository create started",
		zap.Int64("installation_id", installation.InstallationID),
		zap.Uint("user_id", installation.UserID),
		zap.String("account_login", installation.AccountLogin),
	)

	params := sqlc.CreateGitHubInstallationParams{
		InstallationID: uint64(installation.InstallationID),
		UserID:         uint32(installation.UserID),
		AccountLogin:   installation.AccountLogin,
		AccountType:    sqlc.GithubInstallationsAccountType(installation.AccountType),
		Status:         sqlc.GithubInstallationsStatus(installation.Status),
		CachedToken:    toNullString(installation.CachedToken),
		TokenExpiresAt: toNullTime(installation.TokenExpiresAt),
		IsDeleted:      installation.IsDeleted,
		DeletedAt:      toNullTime(installation.DeletedAt),
		CreatedAt:      installation.CreatedAt,
		UpdatedAt:      toNullTime(installation.UpdatedAt),
	}

	_, err := r.queriesWithContext(ctx).CreateGitHubInstallation(ctx, params)
	if err != nil {
		if isDuplicateError(err) {
			r.logger.Error(ctx, "github installation already exists",
				zap.Int64("installation_id", installation.InstallationID),
				zap.Error(usererrors.ErrInstallationExists),
			)
			return usererrors.ErrInstallationExists
		}
		r.logger.Error(ctx, "github installation repository create failed",
			zap.Int64("installation_id", installation.InstallationID),
			zap.Error(err),
		)
		return usererrors.ErrDatabaseOperation
	}

	r.logger.Info(ctx, "github installation repository create completed",
		zap.Int64("installation_id", installation.InstallationID),
		zap.Uint("user_id", installation.UserID),
	)
	return nil
}

func (r *githubInstallationRepository) FindByInstallationID(ctx context.Context, installationID int64) (*model.GitHubInstallation, error) {
	r.logger.Info(ctx, "github installation repository find by installation id started",
		zap.Int64("installation_id", installationID),
	)

	row, err := r.queriesWithContext(ctx).GetGitHubInstallationByID(ctx, uint64(installationID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Error(ctx, "github installation not found",
				zap.Int64("installation_id", installationID),
				zap.Error(usererrors.ErrInstallationNotFound),
			)
			return nil, usererrors.ErrInstallationNotFound
		}
		r.logger.Error(ctx, "github installation repository find failed",
			zap.Int64("installation_id", installationID),
			zap.Error(err),
		)
		return nil, usererrors.ErrDatabaseUnavailable
	}

	installation := toGitHubInstallationModel(row)
	r.logger.Info(ctx, "github installation repository find by installation id completed",
		zap.Int64("installation_id", installationID),
		zap.Uint("user_id", installation.UserID),
	)
	return installation, nil
}

func (r *githubInstallationRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.GitHubInstallation, error) {
	rows, err := r.queriesWithContext(ctx).GetGitHubInstallationsByUserID(ctx, uint32(userID))
	if err != nil {
		return nil, usererrors.ErrDatabaseUnavailable
	}

	installations := make([]*model.GitHubInstallation, 0, len(rows))
	for _, row := range rows {
		installations = append(installations, toGitHubInstallationListModel(row))
	}

	return installations, nil
}

func (r *githubInstallationRepository) Update(ctx context.Context, installation *model.GitHubInstallation) error {
	r.logger.Info(ctx, "github installation repository update started",
		zap.Int64("installation_id", installation.InstallationID),
	)

	params := sqlc.UpdateGitHubInstallationParams{
		CachedToken:    toNullString(installation.CachedToken),
		TokenExpiresAt: toNullTime(installation.TokenExpiresAt),
		UpdatedAt:      toNullTime(installation.UpdatedAt),
		InstallationID: uint64(installation.InstallationID),
	}

	result, err := r.queriesWithContext(ctx).UpdateGitHubInstallation(ctx, params)
	if err != nil {
		r.logger.Error(ctx, "github installation repository update failed",
			zap.Int64("installation_id", installation.InstallationID),
			zap.Error(err),
		)
		return usererrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error(ctx, "github installation repository update rows affected check failed",
			zap.Int64("installation_id", installation.InstallationID),
			zap.Error(err),
		)
		return usererrors.ErrDatabaseOperation
	}

	if rowsAffected == 0 {
		r.logger.Error(ctx, "github installation not found for update",
			zap.Int64("installation_id", installation.InstallationID),
			zap.Error(usererrors.ErrInstallationNotFound),
		)
		return usererrors.ErrInstallationNotFound
	}

	r.logger.Info(ctx, "github installation repository update completed",
		zap.Int64("installation_id", installation.InstallationID),
	)
	return nil
}

func (r *githubInstallationRepository) Delete(ctx context.Context, installationID int64) error {
	now := time.Now()
	params := sqlc.DeleteGitHubInstallationParams{
		DeletedAt:      toNullTime(&now),
		UpdatedAt:      toNullTime(&now),
		InstallationID: uint64(installationID),
	}

	result, err := r.queriesWithContext(ctx).DeleteGitHubInstallation(ctx, params)
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	if rowsAffected == 0 {
		return usererrors.ErrInstallationNotFound
	}

	return nil
}

func (r *githubInstallationRepository) MarkAsRevoked(ctx context.Context, installationID int64) error {
	now := time.Now()
	params := sqlc.MarkInstallationAsRevokedParams{
		UpdatedAt:      toNullTime(&now),
		InstallationID: uint64(installationID),
	}

	result, err := r.queriesWithContext(ctx).MarkInstallationAsRevoked(ctx, params)
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	if rowsAffected == 0 {
		return usererrors.ErrInstallationNotFound
	}

	return nil
}

func (r *githubInstallationRepository) ExistsByInstallationID(ctx context.Context, installationID int64) (bool, error) {
	exists, err := r.queriesWithContext(ctx).ExistsByInstallationID(ctx, uint64(installationID))
	if err != nil {
		return false, usererrors.ErrDatabaseOperation
	}
	return exists, nil
}

func (r *githubInstallationRepository) FindByInstallationIDIncludingRevoked(ctx context.Context, installationID int64) (*model.GitHubInstallation, error) {
	row, err := r.queriesWithContext(ctx).FindInstallationByIDIncludingRevoked(ctx, uint64(installationID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usererrors.ErrInstallationNotFound
		}
		return nil, usererrors.ErrDatabaseUnavailable
	}

	return &model.GitHubInstallation{
		InstallationID: int64(row.InstallationID),
		UserID:         uint(row.UserID),
		AccountLogin:   row.AccountLogin,
		AccountType:    model.AccountType(row.AccountType),
		Status:         model.InstallationStatus(row.Status),
		CachedToken:    nullStringToPtr(row.CachedToken),
		TokenExpiresAt: nullTimeToPtr(row.TokenExpiresAt),
		IsDeleted:      row.IsDeleted,
		DeletedAt:      nullTimeToPtr(row.DeletedAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      nullTimeToPtr(row.UpdatedAt),
	}, nil
}

func (r *githubInstallationRepository) Reactivate(ctx context.Context, installationID int64, accountLogin string, accountType model.AccountType) error {
	now := time.Now()
	params := sqlc.ReactivateInstallationParams{
		AccountLogin:   accountLogin,
		AccountType:    sqlc.GithubInstallationsAccountType(accountType),
		UpdatedAt:      toNullTime(&now),
		InstallationID: uint64(installationID),
	}

	result, err := r.queriesWithContext(ctx).ReactivateInstallation(ctx, params)
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	if rowsAffected == 0 {
		return usererrors.ErrInstallationNotFound
	}

	return nil
}

func (r *githubInstallationRepository) ValidateUserOwnership(ctx context.Context, installationID int64, userID uint) error {
	params := sqlc.ValidateInstallationOwnershipParams{
		InstallationID: uint64(installationID),
		UserID:         uint32(userID),
	}

	isValid, err := r.queriesWithContext(ctx).ValidateInstallationOwnership(ctx, params)
	if err != nil {
		return usererrors.ErrDatabaseOperation
	}

	if !isValid {
		return usererrors.ErrInstallationUnauthorized
	}

	return nil
}

func (r *githubInstallationRepository) queriesWithContext(ctx context.Context) *sqlc.Queries {
	if tx, ok := db.GetTx(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
}

// Helper functions to convert between SQLC types and domain models

func toGitHubInstallationModel(row sqlc.GetGitHubInstallationByIDRow) *model.GitHubInstallation {
	return &model.GitHubInstallation{
		InstallationID: int64(row.InstallationID),
		UserID:         uint(row.UserID),
		AccountLogin:   row.AccountLogin,
		AccountType:    model.AccountType(row.AccountType),
		Status:         model.InstallationStatus(row.Status),
		CachedToken:    nullStringToPtr(row.CachedToken),
		TokenExpiresAt: nullTimeToPtr(row.TokenExpiresAt),
		IsDeleted:      row.IsDeleted,
		DeletedAt:      nullTimeToPtr(row.DeletedAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      nullTimeToPtr(row.UpdatedAt),
	}
}

func toGitHubInstallationListModel(row sqlc.GetGitHubInstallationsByUserIDRow) *model.GitHubInstallation {
	return &model.GitHubInstallation{
		InstallationID: int64(row.InstallationID),
		UserID:         uint(row.UserID),
		AccountLogin:   row.AccountLogin,
		AccountType:    model.AccountType(row.AccountType),
		Status:         model.InstallationStatus(row.Status),
		CachedToken:    nullStringToPtr(row.CachedToken),
		TokenExpiresAt: nullTimeToPtr(row.TokenExpiresAt),
		IsDeleted:      row.IsDeleted,
		DeletedAt:      nullTimeToPtr(row.DeletedAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      nullTimeToPtr(row.UpdatedAt),
	}
}
