package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
)

type ListBranchesUseCase struct {
	installationRepo repository.GitHubInstallationRepository
	githubClient     *github.Client
	logger           logger.Logger
}

func NewListBranchesUseCase(
	installationRepo repository.GitHubInstallationRepository,
	githubClient *github.Client,
	log logger.Logger,
) *ListBranchesUseCase {
	return &ListBranchesUseCase{
		installationRepo: installationRepo,
		githubClient:     githubClient,
		logger:           log,
	}
}

type ListBranchesInput struct {
	UserID         uint
	InstallationID int64
	Owner          string
	Repo           string
}

type BranchInfo struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	CommitSHA string `json:"commit_sha"`
}

type ListBranchesOutput struct {
	Branches []BranchInfo `json:"branches"`
}

func (uc *ListBranchesUseCase) Execute(ctx context.Context, input ListBranchesInput) (*ListBranchesOutput, error) {
	uc.logger.Info(ctx, "list branches started",
		zap.Uint("user_id", input.UserID),
		zap.Int64("installation_id", input.InstallationID),
		zap.String("owner", input.Owner),
		zap.String("repo", input.Repo),
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
	if strings.TrimSpace(input.Owner) == "" || strings.TrimSpace(input.Repo) == "" {
		uc.logger.Error(ctx, "owner and repo are required",
			zap.String("owner", input.Owner),
			zap.String("repo", input.Repo),
		)
		return nil, usererrors.ErrValidationFailed
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
		if errors.Is(err, usererrors.ErrInstallationNotFound) {
			return nil, usererrors.ErrInstallationNotFound
		}
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

	// List branches using GitHub API
	branches, err := uc.githubClient.ListBranches(input.InstallationID, input.Owner, input.Repo)
	if err != nil {
		// Check if the error is due to installation being revoked or unauthorized
		if errors.Is(err, github.ErrInstallationNotFound) {
			uc.logger.Error(ctx, "installation not found in GitHub, marking as revoked",
				zap.Error(err),
				zap.Int64("installation_id", input.InstallationID),
			)
			_ = uc.installationRepo.MarkAsRevoked(ctx, input.InstallationID)
			return nil, usererrors.ErrInstallationRevoked
		}
		if errors.Is(err, github.ErrInstallationUnauthorized) {
			uc.logger.Error(ctx, "installation unauthorized in GitHub, marking as revoked",
				zap.Error(err),
				zap.Int64("installation_id", input.InstallationID),
			)
			_ = uc.installationRepo.MarkAsRevoked(ctx, input.InstallationID)
			return nil, usererrors.ErrInstallationUnauthorized
		}
		uc.logger.Error(ctx, "failed to list branches from GitHub API",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
			zap.Int64("installation_id", input.InstallationID),
			zap.String("owner", input.Owner),
			zap.String("repo", input.Repo),
		)
		return nil, usererrors.ErrGitHubAPIFailed
	}

	// Convert to output format
	branchInfos := make([]BranchInfo, len(branches))
	for i, branch := range branches {
		branchInfos[i] = BranchInfo{
			Name:      branch.Name,
			Protected: branch.Protected,
			CommitSHA: branch.Commit.SHA,
		}
	}

	uc.logger.Info(ctx, "list branches completed",
		zap.Uint("user_id", input.UserID),
		zap.Int64("installation_id", input.InstallationID),
		zap.String("owner", input.Owner),
		zap.String("repo", input.Repo),
		zap.Int("branch_count", len(branchInfos)),
	)

	return &ListBranchesOutput{
		Branches: branchInfos,
	}, nil
}
