package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

type InstallationCallbackInput struct {
	InstallationID int64
	SetupAction    string // "install", "update", or "request"
	State          string
}

type InstallationCallbackOutput struct {
	UserID       uint              `json:"user_id"`
	Installation *InstallationInfo `json:"installation"`
}

type InstallationInfo struct {
	InstallationID int64     `json:"installation_id"`
	AccountLogin   string    `json:"account_login"`
	AccountType    string    `json:"account_type"`
	IsNew          bool      `json:"is_new"`
	CreatedAt      time.Time `json:"created_at"`
}

type InstallationCallbackUseCase struct {
	config           *config.Config
	githubClient     *github.Client
	installationRepo repository.GitHubInstallationRepository
	txManager        db.TxManager
}

func NewInstallationCallbackUseCase(
	cfg *config.Config,
	githubClient *github.Client,
	installationRepo repository.GitHubInstallationRepository,
	txManager db.TxManager,
) *InstallationCallbackUseCase {
	return &InstallationCallbackUseCase{
		config:           cfg,
		githubClient:     githubClient,
		installationRepo: installationRepo,
		txManager:        txManager,
	}
}

func (uc *InstallationCallbackUseCase) Execute(ctx context.Context, input InstallationCallbackInput) (*InstallationCallbackOutput, error) {
	// Validate state format and extract user ID
	// State format: "base64_random:user_id"
	parts := strings.Split(input.State, ":")
	if len(parts) != 2 {
		return nil, usererrors.ErrInvalidState
	}

	userID, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil, usererrors.ErrInvalidState
	}

	// Get installation info from GitHub API
	installationInfo, err := uc.githubClient.GetInstallationInfo(input.InstallationID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", usererrors.ErrGitHubAPIFailed, err)
	}

	var result *InstallationInfo

	// Process installation
	err = uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check if installation already exists
		exists, err := uc.installationRepo.ExistsByInstallationID(txCtx, input.InstallationID)
		if err != nil {
			return fmt.Errorf("failed to check installation existence: %w", err)
		}

		isNew := !exists
		var installation *model.GitHubInstallation

		if exists {
			// Update existing installation
			installation, err = uc.installationRepo.FindByInstallationID(txCtx, input.InstallationID)
			if err != nil {
				return fmt.Errorf("failed to find installation: %w", err)
			}

			// Update installation info
			installation.AccountLogin = installationInfo.Account.Login
			installation.AccountType = model.AccountType(installationInfo.Account.Type)
			installation.Status = model.InstallationStatusActive
			now := time.Now()
			installation.UpdatedAt = &now

			if err := uc.installationRepo.Update(txCtx, installation); err != nil {
				return fmt.Errorf("failed to update installation: %w", err)
			}
		} else {
			// Create new installation
			installation = &model.GitHubInstallation{
				InstallationID: input.InstallationID,
				UserID:         uint(userID),
				AccountLogin:   installationInfo.Account.Login,
				AccountType:    model.AccountType(installationInfo.Account.Type),
				Status:         model.InstallationStatusActive,
				CreatedAt:      time.Now(),
			}

			if err := uc.installationRepo.Create(txCtx, installation); err != nil {
				return fmt.Errorf("failed to create installation: %w", err)
			}
		}

		result = &InstallationInfo{
			InstallationID: installation.InstallationID,
			AccountLogin:   installation.AccountLogin,
			AccountType:    string(installation.AccountType),
			IsNew:          isNew,
			CreatedAt:      installation.CreatedAt,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &InstallationCallbackOutput{
		UserID:       uint(userID),
		Installation: result,
	}, nil
}
