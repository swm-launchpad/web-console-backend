package application

import (
	"context"
	"fmt"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
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
}

func NewConnectGitHubUseCase(
	installationRepo repository.GitHubInstallationRepository,
	githubClient *github.Client,
	txManager db.TxManager,
) *ConnectGitHubUseCase {
	return &ConnectGitHubUseCase{
		installationRepo: installationRepo,
		githubClient:     githubClient,
		txManager:        txManager,
	}
}

func (uc *ConnectGitHubUseCase) Execute(ctx context.Context, input ConnectGitHubInput) (*ConnectGitHubOutput, error) {
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

	var output *ConnectGitHubOutput

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check if installation already exists
		exists, err := uc.installationRepo.ExistsByInstallationID(txCtx, input.InstallationID)
		if err != nil {
			return fmt.Errorf("failed to check installation existence: %w", err)
		}
		if exists {
			return usererrors.ErrInstallationExists
		}

		// Get installation info from GitHub API
		installationInfo, err := uc.githubClient.GetInstallationInfo(input.InstallationID)
		if err != nil {
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
			return err
		}

		// Save to database
		if err := uc.installationRepo.Create(txCtx, installation); err != nil {
			return fmt.Errorf("failed to create installation: %w", err)
		}

		// Prepare output
		output = &ConnectGitHubOutput{
			InstallationID: installation.InstallationID,
			AccountLogin:   installation.AccountLogin,
			AccountType:    string(installation.AccountType),
			CreatedAt:      installation.CreatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return output, nil
}
