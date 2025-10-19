package application

import (
	"context"
	"fmt"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth/state"
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
	stateRepo        repository.OAuthStateRepository
	txManager        db.TxManager
	stateValidator   *state.StateValidator
}

func NewInstallationCallbackUseCase(
	cfg *config.Config,
	githubClient *github.Client,
	installationRepo repository.GitHubInstallationRepository,
	stateRepo repository.OAuthStateRepository,
	txManager db.TxManager,
) *InstallationCallbackUseCase {
	return &InstallationCallbackUseCase{
		config:           cfg,
		githubClient:     githubClient,
		installationRepo: installationRepo,
		stateRepo:        stateRepo,
		txManager:        txManager,
		stateValidator:   state.NewStateValidator(cfg.JWT.Secret),
	}
}

func (uc *InstallationCallbackUseCase) Execute(ctx context.Context, input InstallationCallbackInput) (*InstallationCallbackOutput, error) {
	// Validate input
	if input.InstallationID <= 0 {
		return nil, usererrors.ErrInvalidInstallationID
	}
	if input.State == "" {
		return nil, usererrors.ErrInvalidState
	}

	// Check if GitHub client is configured
	if uc.githubClient == nil {
		return nil, usererrors.ErrGitHubNotConfigured
	}

	// Step 1: Validate HMAC signature
	userID, err := uc.stateValidator.ValidateState(input.State)
	if err != nil {
		return nil, usererrors.ErrInvalidState
	}

	// Step 2: Verify state exists in database and hasn't been consumed (prevents replay attacks)
	oauthState, err := uc.stateRepo.FindByState(ctx, input.State)
	if err != nil {
		return nil, usererrors.ErrInvalidState
	}

	// Step 3: Check if state is still valid (not expired, not consumed)
	if !oauthState.CanBeUsed() {
		return nil, usererrors.ErrInvalidState
	}

	// Step 4: Verify user ID from state matches the one in database
	if oauthState.UserID != userID {
		return nil, usererrors.ErrInvalidState
	}

	// Step 5: Mark state as consumed immediately (one-time use)
	if err := uc.stateRepo.MarkAsConsumed(ctx, input.State, input.InstallationID); err != nil {
		return nil, fmt.Errorf("failed to consume state: %w", err)
	}

	// Step 6: Get installation info from GitHub API
	installationInfo, err := uc.githubClient.GetInstallationInfo(input.InstallationID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", usererrors.ErrGitHubAPIFailed, err)
	}

	var result *InstallationInfo

	// Process installation
	err = uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Check if installation already exists (including revoked ones)
		installation, err := uc.installationRepo.FindByInstallationIDIncludingRevoked(txCtx, input.InstallationID)
		isNew := false

		if err != nil {
			// Installation doesn't exist - create new one
			if err == usererrors.ErrInstallationNotFound {
				isNew = true
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
			} else {
				return fmt.Errorf("failed to find installation: %w", err)
			}
		} else {
			// Installation exists - verify user ownership before proceeding
			if installation.UserID != userID {
				return usererrors.ErrInstallationUnauthorized
			}

			// Installation exists and owned by user - reactivate if deleted or revoked
			if installation.IsDeleted || installation.Status == model.InstallationStatusRevoked {
				// Reactivate deleted or revoked installation
				// This restores is_deleted=FALSE, deleted_at=NULL, and status='active'
				if err := uc.installationRepo.Reactivate(
					txCtx,
					input.InstallationID,
					installationInfo.Account.Login,
					model.AccountType(installationInfo.Account.Type),
				); err != nil {
					return fmt.Errorf("failed to reactivate installation: %w", err)
				}
				// Update local installation object for response
				installation.Status = model.InstallationStatusActive
				installation.AccountLogin = installationInfo.Account.Login
				installation.AccountType = model.AccountType(installationInfo.Account.Type)
				installation.IsDeleted = false
				installation.DeletedAt = nil
			} else {
				// Update active installation
				installation.AccountLogin = installationInfo.Account.Login
				installation.AccountType = model.AccountType(installationInfo.Account.Type)
				installation.Status = model.InstallationStatusActive
				now := time.Now()
				installation.UpdatedAt = &now

				if err := uc.installationRepo.Update(txCtx, installation); err != nil {
					return fmt.Errorf("failed to update installation: %w", err)
				}
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
