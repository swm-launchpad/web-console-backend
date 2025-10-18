package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

type GenerateInstallationTokenInput struct {
	InstallationID int64
	UserID         uint
}

type GenerateInstallationTokenOutput struct {
	Token string `json:"token"`
}

type GenerateInstallationTokenUseCase struct {
	installationRepo repository.GitHubInstallationRepository
	githubClient     *github.Client
	txManager        db.TxManager
}

func NewGenerateInstallationTokenUseCase(
	installationRepo repository.GitHubInstallationRepository,
	githubClient *github.Client,
	txManager db.TxManager,
) *GenerateInstallationTokenUseCase {
	return &GenerateInstallationTokenUseCase{
		installationRepo: installationRepo,
		githubClient:     githubClient,
		txManager:        txManager,
	}
}

func (uc *GenerateInstallationTokenUseCase) Execute(ctx context.Context, input GenerateInstallationTokenInput) (*GenerateInstallationTokenOutput, error) {
	var token string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Get installation
		installation, err := uc.installationRepo.FindByInstallationID(txCtx, input.InstallationID)
		if err != nil {
			return err
		}

		// Verify ownership
		if installation.UserID != input.UserID {
			return usererrors.ErrInstallationNotFound
		}

		// Check if cached token is still valid (5 minute buffer)
		if installation.IsTokenValid(5) {
			token = *installation.CachedToken
			return nil
		}

		// Generate new token from GitHub
		installationToken, err := uc.githubClient.CreateInstallationToken(input.InstallationID)
		if err != nil {
			// Check if the error is due to installation being revoked or unauthorized
			if errors.Is(err, github.ErrInstallationNotFound) {
				// Mark installation as revoked in database
				_ = uc.installationRepo.MarkAsRevoked(txCtx, input.InstallationID)
				return usererrors.ErrInstallationRevoked
			}
			if errors.Is(err, github.ErrInstallationUnauthorized) {
				// Mark installation as revoked in database
				_ = uc.installationRepo.MarkAsRevoked(txCtx, input.InstallationID)
				return usererrors.ErrInstallationUnauthorized
			}
			return fmt.Errorf("%w: %v", usererrors.ErrGitHubTokenGenerateFail, err)
		}

		// Update cached token
		installation.UpdateToken(installationToken.Token, installationToken.ExpiresAt)

		// Save to database
		if err := uc.installationRepo.Update(txCtx, installation); err != nil {
			return fmt.Errorf("failed to update installation token: %w", err)
		}

		token = installationToken.Token
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &GenerateInstallationTokenOutput{
		Token: token,
	}, nil
}
