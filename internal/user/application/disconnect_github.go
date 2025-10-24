package application

import (
	"context"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
)

type DisconnectGitHubInput struct {
	UserID         uint
	InstallationID int64
}

type DisconnectGitHubUseCase struct {
	installationRepo repository.GitHubInstallationRepository
	txManager        db.TxManager
	logger           logger.Logger
}

func NewDisconnectGitHubUseCase(
	installationRepo repository.GitHubInstallationRepository,
	txManager db.TxManager,
	log logger.Logger,
) *DisconnectGitHubUseCase {
	return &DisconnectGitHubUseCase{
		installationRepo: installationRepo,
		txManager:        txManager,
		logger:           log,
	}
}

func (uc *DisconnectGitHubUseCase) Execute(ctx context.Context, input DisconnectGitHubInput) error {
	uc.logger.Info(ctx, "disconnect github started",
		zap.Uint("user_id", input.UserID),
		zap.Int64("installation_id", input.InstallationID),
	)

	// Validate input
	if input.UserID == 0 {
		uc.logger.Error(ctx, "user ID is required",
			zap.Uint("user_id", input.UserID),
		)
		return usererrors.ErrUserIDRequired
	}
	if input.InstallationID <= 0 {
		uc.logger.Error(ctx, "invalid installation ID",
			zap.Int64("installation_id", input.InstallationID),
		)
		return usererrors.ErrInvalidInstallationID
	}

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Get installation to verify ownership
		installation, err := uc.installationRepo.FindByInstallationID(txCtx, input.InstallationID)
		if err != nil {
			uc.logger.Error(ctx, "failed to find installation",
				zap.Error(err),
				zap.Int64("installation_id", input.InstallationID),
			)
			return err
		}

		// Verify that the installation belongs to the user
		if installation.UserID != input.UserID {
			uc.logger.Warn(ctx, "installation does not belong to user",
				zap.Uint("user_id", input.UserID),
				zap.Uint("installation_user_id", installation.UserID),
				zap.Int64("installation_id", input.InstallationID),
			)
			return usererrors.ErrInstallationNotFound
		}

		// Soft delete the installation
		if err := uc.installationRepo.Delete(txCtx, input.InstallationID); err != nil {
			uc.logger.Error(ctx, "failed to delete installation",
				zap.Error(err),
				zap.Uint("user_id", input.UserID),
				zap.Int64("installation_id", input.InstallationID),
			)
			return fmt.Errorf("failed to delete installation: %w", err)
		}

		uc.logger.Info(ctx, "github installation disconnected successfully",
			zap.Uint("user_id", input.UserID),
			zap.Int64("installation_id", input.InstallationID),
		)

		return nil
	})

	if err != nil {
		return err
	}

	uc.logger.Info(ctx, "disconnect github completed",
		zap.Uint("user_id", input.UserID),
		zap.Int64("installation_id", input.InstallationID),
	)

	return nil
}
