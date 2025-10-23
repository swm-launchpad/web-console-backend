package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

type ListGitHubBranchesUseCase struct {
	installationRepo repository.GitHubInstallationRepository
	githubClient     *github.Client
}

func NewListGitHubBranchesUseCase(
	installationRepo repository.GitHubInstallationRepository,
	githubClient *github.Client,
) *ListGitHubBranchesUseCase {
	return &ListGitHubBranchesUseCase{
		installationRepo: installationRepo,
		githubClient:     githubClient,
	}
}

type ListGitHubBranchesInput struct {
	UserID         uint
	InstallationID int64
	Repo           string // Repository name only (e.g., "backend")
}

type BranchDTO struct {
	Name      string `json:"name"`
	CommitSHA string `json:"commit_sha"`
	Protected bool   `json:"protected"`
}

type ListGitHubBranchesOutput struct {
	Branches     []BranchDTO `json:"branches"`
	TotalCount   int         `json:"total_count"`
	ShowingCount int         `json:"showing_count"`
	HasMore      bool        `json:"has_more"`
}

func (uc *ListGitHubBranchesUseCase) Execute(ctx context.Context, input ListGitHubBranchesInput) (*ListGitHubBranchesOutput, error) {
	// Validate input
	if input.UserID == 0 {
		return nil, usererrors.ErrUserIDRequired
	}
	if input.InstallationID <= 0 {
		return nil, usererrors.ErrInvalidInstallationID
	}
	if input.Repo == "" {
		return nil, usererrors.ErrRepositoryNameRequired
	}

	// Check if GitHub client is configured
	if uc.githubClient == nil {
		return nil, usererrors.ErrGitHubNotConfigured
	}

	// Verify the installation belongs to the user and get owner from account_login
	installation, err := uc.installationRepo.FindByInstallationID(ctx, input.InstallationID)
	if err != nil {
		// Only convert to ErrInstallationNotFound if it's actually a not found error
		if errors.Is(err, usererrors.ErrInstallationNotFound) {
			return nil, usererrors.ErrInstallationNotFound
		}
		// Wrap infrastructure errors for better debugging
		return nil, fmt.Errorf("failed to find installation: %w", err)
	}

	if installation.UserID != input.UserID {
		return nil, usererrors.ErrInvalidUserID
	}

	// Get owner from installation's account_login
	owner := installation.AccountLogin

	// List branches using GitHub API
	branches, err := uc.githubClient.ListBranches(input.InstallationID, owner, input.Repo)
	if err != nil {
		// Handle installation revocation errors
		if errors.Is(err, github.ErrInstallationRevoked) || errors.Is(err, github.ErrInstallationNotFound) {
			if markErr := uc.installationRepo.MarkAsRevoked(ctx, input.InstallationID); markErr != nil {
				// Log error (TODO: Add logger)
				// For now, we continue with returning the original error
				_ = markErr
			}
			return nil, usererrors.ErrInstallationRevoked
		}
		if errors.Is(err, github.ErrInstallationUnauthorized) {
			if markErr := uc.installationRepo.MarkAsRevoked(ctx, input.InstallationID); markErr != nil {
				// Log error (TODO: Add logger)
				_ = markErr
			}
			return nil, usererrors.ErrInstallationUnauthorized
		}
		// Repository-specific errors
		if errors.Is(err, github.ErrRepositoryNotFound) {
			return nil, usererrors.ErrRepositoryNotFound
		}
		if errors.Is(err, github.ErrRepositoryForbidden) {
			return nil, usererrors.ErrRepositoryForbidden
		}
		return nil, usererrors.ErrGitHubAPIFailed
	}

	// Calculate total count and limit to 100 branches
	totalCount := len(branches)
	if totalCount > 100 {
		branches = branches[:100]
	}
	showingCount := len(branches)

	// Convert to DTO format
	branchDTOs := make([]BranchDTO, showingCount)
	for i, branch := range branches {
		branchDTOs[i] = BranchDTO{
			Name:      branch.Name,
			CommitSHA: branch.Commit.SHA,
			Protected: branch.Protected,
		}
	}

	return &ListGitHubBranchesOutput{
		Branches:     branchDTOs,
		TotalCount:   totalCount,
		ShowingCount: showingCount,
		HasMore:      totalCount > 100,
	}, nil
}
