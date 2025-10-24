package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
)

type ListRepositoriesUseCase struct {
	installationRepo repository.GitHubInstallationRepository
	githubClient     *github.Client
	logger           logger.Logger
}

func NewListRepositoriesUseCase(
	installationRepo repository.GitHubInstallationRepository,
	githubClient *github.Client,
	log logger.Logger,
) *ListRepositoriesUseCase {
	return &ListRepositoriesUseCase{
		installationRepo: installationRepo,
		githubClient:     githubClient,
		logger:           log,
	}
}

type ListRepositoriesInput struct {
	UserID         uint
	InstallationID int64
}

type RepositoryInfo struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	HTMLURL  string `json:"html_url"`
	CloneURL string `json:"clone_url"`
}

type ListRepositoriesOutput struct {
	Repositories []RepositoryInfo `json:"repositories"`
}

func (uc *ListRepositoriesUseCase) Execute(ctx context.Context, input ListRepositoriesInput) (*ListRepositoriesOutput, error) {
	uc.logger.Info(ctx, "list repositories started",
		zap.Uint("user_id", input.UserID),
		zap.Int64("installation_id", input.InstallationID),
	)

	// Validate input
	if input.UserID == 0 {
		uc.logger.Error(ctx, "user ID is required",
			zap.Uint("user_id", input.UserID),
		)
		return nil, usererrors.ErrUserIDRequired
	}
	if input.InstallationID <= 0 {
		uc.logger.Error(ctx, "invalid installation ID",
			zap.Int64("installation_id", input.InstallationID),
		)
		return nil, usererrors.ErrInvalidInstallationID
	}

	// Check if GitHub client is configured
	if uc.githubClient == nil {
		uc.logger.Error(ctx, "github client not configured",
			zap.Uint("user_id", input.UserID),
		)
		return nil, usererrors.ErrGitHubNotConfigured
	}

	// Verify the installation belongs to the user
	installation, err := uc.installationRepo.FindByInstallationID(ctx, input.InstallationID)
	if err != nil {
		uc.logger.Error(ctx, "failed to find installation",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
			zap.Int64("installation_id", input.InstallationID),
		)
		// Only convert to ErrInstallationNotFound if it's actually a not found error
		// Preserve other errors (database unavailable, etc.)
		if errors.Is(err, usererrors.ErrInstallationNotFound) {
			return nil, usererrors.ErrInstallationNotFound
		}
		// Wrap infrastructure errors for better debugging
		return nil, fmt.Errorf("failed to find installation: %w", err)
	}

	if installation.UserID != input.UserID {
		uc.logger.Warn(ctx, "installation does not belong to user",
			zap.Uint("user_id", input.UserID),
			zap.Uint("installation_user_id", installation.UserID),
			zap.Int64("installation_id", input.InstallationID),
		)
		return nil, usererrors.ErrInvalidUserID
	}

	// List repositories using GitHub API
	repos, err := uc.githubClient.ListRepositories(input.InstallationID)
	if err != nil {
		// Check if the error is due to installation being revoked or unauthorized
		if errors.Is(err, github.ErrInstallationNotFound) {
			uc.logger.Error(ctx, "installation not found in GitHub, marking as revoked",
				zap.Error(err),
				zap.Int64("installation_id", input.InstallationID),
			)
			// Mark installation as revoked in database
			_ = uc.installationRepo.MarkAsRevoked(ctx, input.InstallationID)
			return nil, usererrors.ErrInstallationRevoked
		}
		if errors.Is(err, github.ErrInstallationUnauthorized) {
			uc.logger.Error(ctx, "installation unauthorized in GitHub, marking as revoked",
				zap.Error(err),
				zap.Int64("installation_id", input.InstallationID),
			)
			// Mark installation as revoked in database
			_ = uc.installationRepo.MarkAsRevoked(ctx, input.InstallationID)
			return nil, usererrors.ErrInstallationUnauthorized
		}
		uc.logger.Error(ctx, "failed to list repositories from GitHub API",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
			zap.Int64("installation_id", input.InstallationID),
		)
		return nil, usererrors.ErrGitHubAPIFailed
	}

	// Convert to output format
	repositories := make([]RepositoryInfo, len(repos))
	for i, repo := range repos {
		repositories[i] = RepositoryInfo{
			ID:       repo.ID,
			Name:     repo.Name,
			FullName: repo.FullName,
			Private:  repo.Private,
			HTMLURL:  repo.HTMLURL,
			CloneURL: repo.CloneURL,
		}
	}

	uc.logger.Info(ctx, "list repositories completed",
		zap.Uint("user_id", input.UserID),
		zap.Int64("installation_id", input.InstallationID),
		zap.Int("repository_count", len(repositories)),
	)

	return &ListRepositoriesOutput{
		Repositories: repositories,
	}, nil
}
