package application

import (
	"context"
	"errors"

	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

type ListRepositoriesUseCase struct {
	installationRepo repository.GitHubInstallationRepository
	githubClient     *github.Client
}

func NewListRepositoriesUseCase(
	installationRepo repository.GitHubInstallationRepository,
	githubClient *github.Client,
) *ListRepositoriesUseCase {
	return &ListRepositoriesUseCase{
		installationRepo: installationRepo,
		githubClient:     githubClient,
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
	// Validate input
	if input.UserID == 0 {
		return nil, usererrors.ErrUserIDRequired
	}
	if input.InstallationID <= 0 {
		return nil, usererrors.ErrInvalidInstallationID
	}

	// Check if GitHub client is configured
	if uc.githubClient == nil {
		return nil, usererrors.ErrGitHubNotConfigured
	}

	// Verify the installation belongs to the user
	installation, err := uc.installationRepo.FindByInstallationID(ctx, input.InstallationID)
	if err != nil {
		return nil, usererrors.ErrInstallationNotFound
	}

	if installation.UserID != input.UserID {
		return nil, usererrors.ErrInvalidUserID
	}

	// List repositories using GitHub API
	repos, err := uc.githubClient.ListRepositories(input.InstallationID)
	if err != nil {
		// Check if the error is due to installation being revoked or unauthorized
		if errors.Is(err, github.ErrInstallationNotFound) {
			// Mark installation as revoked in database
			_ = uc.installationRepo.MarkAsRevoked(ctx, input.InstallationID)
			return nil, usererrors.ErrInstallationRevoked
		}
		if errors.Is(err, github.ErrInstallationUnauthorized) {
			// Mark installation as revoked in database
			_ = uc.installationRepo.MarkAsRevoked(ctx, input.InstallationID)
			return nil, usererrors.ErrInstallationUnauthorized
		}
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

	return &ListRepositoriesOutput{
		Repositories: repositories,
	}, nil
}
