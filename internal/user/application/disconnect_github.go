package application

import (
	"context"
	"fmt"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

type DisconnectGitHubInput struct {
	UserID         uint
	InstallationID int64
}

type DisconnectGitHubUseCase struct {
	installationRepo repository.GitHubInstallationRepository
	txManager        db.TxManager
}

func NewDisconnectGitHubUseCase(
	installationRepo repository.GitHubInstallationRepository,
	txManager db.TxManager,
) *DisconnectGitHubUseCase {
	return &DisconnectGitHubUseCase{
		installationRepo: installationRepo,
		txManager:        txManager,
	}
}

func (uc *DisconnectGitHubUseCase) Execute(ctx context.Context, input DisconnectGitHubInput) error {
	// Validate input
	if input.UserID == 0 {
		return usererrors.ErrUserIDRequired
	}
	if input.InstallationID <= 0 {
		return usererrors.ErrInvalidInstallationID
	}

	return uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Get installation to verify ownership
		installation, err := uc.installationRepo.FindByInstallationID(txCtx, input.InstallationID)
		if err != nil {
			return err
		}

		// Verify that the installation belongs to the user
		if installation.UserID != input.UserID {
			return usererrors.ErrInstallationNotFound
		}

		// Soft delete the installation
		if err := uc.installationRepo.Delete(txCtx, input.InstallationID); err != nil {
			return fmt.Errorf("failed to delete installation: %w", err)
		}

		return nil
	})
}
