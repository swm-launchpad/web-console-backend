package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
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
	logger           logger.Logger
}

func NewGenerateInstallationTokenUseCase(
	installationRepo repository.GitHubInstallationRepository,
	githubClient *github.Client,
	txManager db.TxManager,
	log logger.Logger,
) *GenerateInstallationTokenUseCase {
	return &GenerateInstallationTokenUseCase{
		installationRepo: installationRepo,
		githubClient:     githubClient,
		txManager:        txManager,
		logger:           log,
	}
}

func (uc *GenerateInstallationTokenUseCase) Execute(ctx context.Context, input GenerateInstallationTokenInput) (*GenerateInstallationTokenOutput, error) {
	uc.logger.Info(ctx, "generate installation token started",
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

	var token string
	var usedCachedToken bool

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Get installation
		installation, err := uc.installationRepo.FindByInstallationID(txCtx, input.InstallationID)
		if err != nil {
			uc.logger.Error(ctx, "failed to find installation",
				zap.Error(err),
				zap.Int64("installation_id", input.InstallationID),
			)
			return err
		}

		// Verify ownership
		if installation.UserID != input.UserID {
			uc.logger.Warn(ctx, "installation does not belong to user",
				zap.Uint("user_id", input.UserID),
				zap.Uint("installation_user_id", installation.UserID),
				zap.Int64("installation_id", input.InstallationID),
			)
			return usererrors.ErrInstallationNotFound
		}

		// Check if cached token is still valid (5 minute buffer)
		if installation.IsTokenValid(5) {
			token = *installation.CachedToken
			usedCachedToken = true
			uc.logger.Info(ctx, "using cached installation token",
				zap.Uint("user_id", input.UserID),
				zap.Int64("installation_id", input.InstallationID),
			)
			return nil
		}

		// Generate new token from GitHub
		uc.logger.Info(ctx, "generating new installation token from GitHub",
			zap.Uint("user_id", input.UserID),
			zap.Int64("installation_id", input.InstallationID),
		)

		installationToken, err := uc.githubClient.CreateInstallationToken(input.InstallationID)
		if err != nil {
			// Check if the error is due to installation being revoked or unauthorized
			if errors.Is(err, github.ErrInstallationNotFound) {
				uc.logger.Error(ctx, "installation not found in GitHub, marking as revoked",
					zap.Error(err),
					zap.Int64("installation_id", input.InstallationID),
				)
				// Mark installation as revoked in database
				_ = uc.installationRepo.MarkAsRevoked(txCtx, input.InstallationID)
				return usererrors.ErrInstallationRevoked
			}
			if errors.Is(err, github.ErrInstallationUnauthorized) {
				uc.logger.Error(ctx, "installation unauthorized in GitHub, marking as revoked",
					zap.Error(err),
					zap.Int64("installation_id", input.InstallationID),
				)
				// Mark installation as revoked in database
				_ = uc.installationRepo.MarkAsRevoked(txCtx, input.InstallationID)
				return usererrors.ErrInstallationUnauthorized
			}
			uc.logger.Error(ctx, "failed to create installation token from GitHub",
				zap.Error(err),
				zap.Int64("installation_id", input.InstallationID),
			)
			return fmt.Errorf("%w: %v", usererrors.ErrGitHubTokenGenerateFail, err)
		}

		// Update cached token
		installation.UpdateToken(installationToken.Token, installationToken.ExpiresAt)

		// Save to database
		if err := uc.installationRepo.Update(txCtx, installation); err != nil {
			uc.logger.Error(ctx, "failed to update installation token in database",
				zap.Error(err),
				zap.Uint("user_id", input.UserID),
				zap.Int64("installation_id", input.InstallationID),
			)
			return fmt.Errorf("failed to update installation token: %w", err)
		}

		token = installationToken.Token
		uc.logger.Info(ctx, "installation token generated and cached successfully",
			zap.Uint("user_id", input.UserID),
			zap.Int64("installation_id", input.InstallationID),
		)
		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "generate installation token completed",
		zap.Uint("user_id", input.UserID),
		zap.Int64("installation_id", input.InstallationID),
		zap.Bool("used_cached_token", usedCachedToken),
	)

	return &GenerateInstallationTokenOutput{
		Token: token,
	}, nil
}
