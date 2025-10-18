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

type githubInstallationRepository struct {
	queries *sqlc.Queries
}

func NewGitHubInstallationRepository(db sqlc.DBTX) repository.GitHubInstallationRepository {
	return &githubInstallationRepository{
		queries: sqlc.New(db),
	}
}

func (r *githubInstallationRepository) Create(ctx context.Context, installation *model.GitHubInstallation) error {
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
			return usererrors.ErrInstallationExists
		}
		return usererrors.ErrDatabaseOperation
	}

	return nil
}

func (r *githubInstallationRepository) FindByInstallationID(ctx context.Context, installationID int64) (*model.GitHubInstallation, error) {
	row, err := r.queriesWithContext(ctx).GetGitHubInstallationByID(ctx, uint64(installationID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, usererrors.ErrInstallationNotFound
		}
		return nil, usererrors.ErrDatabaseUnavailable
	}

	return toGitHubInstallationModel(row), nil
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
	params := sqlc.UpdateGitHubInstallationParams{
		CachedToken:    toNullString(installation.CachedToken),
		TokenExpiresAt: toNullTime(installation.TokenExpiresAt),
		UpdatedAt:      toNullTime(installation.UpdatedAt),
		InstallationID: uint64(installation.InstallationID),
	}

	result, err := r.queriesWithContext(ctx).UpdateGitHubInstallation(ctx, params)
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
