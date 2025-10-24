package application

import (
	"context"
	"fmt"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
)

type ConnectGitHubInput struct {
	UserID         uint
	InstallationID int64
}

type ConnectGitHubOutput struct {
	InstallationID int64     `json:"installation_id"`
	AccountLogin   string    `json:"account_login"`
	AccountType    string    `json:"account_type"`
	CreatedAt      time.Time `json:"created_at"`
}

type ConnectGitHubUseCase struct {
	installationRepo repository.GitHubInstallationRepository
	githubClient     *github.Client
	txManager        db.TxManager
	logger           logger.Logger
}

func NewConnectGitHubUseCase(
	installationRepo repository.GitHubInstallationRepository,
	githubClient *github.Client,
	txManager db.TxManager,
	log logger.Logger,
) *ConnectGitHubUseCase {
	return &ConnectGitHubUseCase{
		installationRepo: installationRepo,
		githubClient:     githubClient,
		txManager:        txManager,
		logger:           log,
	}
}

func (uc *ConnectGitHubUseCase) Execute(ctx context.Context, input ConnectGitHubInput) (*ConnectGitHubOutput, error) {
	uc.logger.Info(ctx, "connect github started",
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

	var output *ConnectGitHubOutput

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check if installation already exists
		exists, err := uc.installationRepo.ExistsByInstallationID(txCtx, input.InstallationID)
		if err != nil {
			uc.logger.Error(ctx, "failed to check installation existence",
				zap.Error(err),
				zap.Int64("installation_id", input.InstallationID),
			)
			return fmt.Errorf("failed to check installation existence: %w", err)
		}
		if exists {
			uc.logger.Warn(ctx, "installation already exists",
				zap.Int64("installation_id", input.InstallationID),
			)
			return usererrors.ErrInstallationExists
		}

		// Get installation info from GitHub API
		installationInfo, err := uc.githubClient.GetInstallationInfo(input.InstallationID)
		if err != nil {
			uc.logger.Error(ctx, "failed to get installation info from GitHub API",
				zap.Error(err),
				zap.Int64("installation_id", input.InstallationID),
			)
			return fmt.Errorf("%w: %v", usererrors.ErrGitHubAPIFailed, err)
		}

		// Create GitHub installation entity
		accountType := model.AccountTypeUser
		if installationInfo.Account.Type == "Organization" {
			accountType = model.AccountTypeOrganization
		}

		installation, err := model.NewGitHubInstallation(
			input.InstallationID,
			input.UserID,
			installationInfo.Account.Login,
			accountType,
		)
		if err != nil {
			uc.logger.Error(ctx, "failed to create github installation entity",
				zap.Error(err),
				zap.Uint("user_id", input.UserID),
				zap.Int64("installation_id", input.InstallationID),
			)
			return err
		}

		// Save to database
		if err := uc.installationRepo.Create(txCtx, installation); err != nil {
			uc.logger.Error(ctx, "failed to create installation in database",
				zap.Error(err),
				zap.Uint("user_id", input.UserID),
				zap.Int64("installation_id", input.InstallationID),
			)
			return fmt.Errorf("failed to create installation: %w", err)
		}

		// Prepare output
		output = &ConnectGitHubOutput{
			InstallationID: installation.InstallationID,
			AccountLogin:   installation.AccountLogin,
			AccountType:    string(installation.AccountType),
			CreatedAt:      installation.CreatedAt,
		}

		uc.logger.Info(ctx, "github installation connected successfully",
			zap.Uint("user_id", input.UserID),
			zap.Int64("installation_id", installation.InstallationID),
			zap.String("account_login", installation.AccountLogin),
		)

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "connect github completed",
		zap.Uint("user_id", input.UserID),
		zap.Int64("installation_id", output.InstallationID),
	)

	return output, nil
}
