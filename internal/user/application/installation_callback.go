package application

import (
	"context"
	"fmt"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth/state"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
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
	logger           logger.Logger
}

func NewInstallationCallbackUseCase(
	cfg *config.Config,
	githubClient *github.Client,
	installationRepo repository.GitHubInstallationRepository,
	stateRepo repository.OAuthStateRepository,
	txManager db.TxManager,
	log logger.Logger,
) *InstallationCallbackUseCase {
	return &InstallationCallbackUseCase{
		config:           cfg,
		githubClient:     githubClient,
		installationRepo: installationRepo,
		stateRepo:        stateRepo,
		txManager:        txManager,
		stateValidator:   state.NewStateValidator(cfg.JWT.Secret),
		logger:           log,
	}
}

func (uc *InstallationCallbackUseCase) Execute(ctx context.Context, input InstallationCallbackInput) (*InstallationCallbackOutput, error) {
	uc.logger.Info(ctx, "installation callback started",
		zap.Int64("installation_id", input.InstallationID),
		zap.String("setup_action", input.SetupAction),
		zap.String("state_prefix", input.State[:min(10, len(input.State))]),
	)

	// Validate input
	if input.InstallationID <= 0 {
		uc.logger.Error(ctx, "invalid installation ID",
			zap.Int64("installation_id", input.InstallationID),
		)
		return nil, usererrors.ErrInvalidInstallationID
	}
	if input.State == "" {
		uc.logger.Error(ctx, "state is empty")
		return nil, usererrors.ErrInvalidState
	}

	// Check if GitHub client is configured
	if uc.githubClient == nil {
		uc.logger.Error(ctx, "github client not configured")
		return nil, usererrors.ErrGitHubNotConfigured
	}

	// Step 1: Validate HMAC signature
	userID, err := uc.stateValidator.ValidateState(input.State)
	if err != nil {
		uc.logger.Error(ctx, "failed to validate state HMAC signature",
			zap.Error(err),
			zap.String("state_prefix", input.State[:min(10, len(input.State))]),
		)
		return nil, usererrors.ErrInvalidState
	}

	// Step 2: Verify state exists in database and hasn't been consumed (prevents replay attacks)
	oauthState, err := uc.stateRepo.FindByState(ctx, input.State)
	if err != nil {
		uc.logger.Error(ctx, "failed to find state in database",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.String("state_prefix", input.State[:min(10, len(input.State))]),
		)
		return nil, usererrors.ErrInvalidState
	}

	// Step 3: Check if state is still valid (not expired, not consumed)
	if !oauthState.CanBeUsed() {
		uc.logger.Warn(ctx, "state cannot be used (expired or already consumed)",
			zap.Uint("user_id", userID),
			zap.Bool("is_consumed", oauthState.ConsumedAt != nil),
		)
		return nil, usererrors.ErrInvalidState
	}

	// Step 4: Verify user ID from state matches the one in database
	if oauthState.UserID != userID {
		uc.logger.Error(ctx, "user ID mismatch between state and database",
			zap.Uint("state_user_id", userID),
			zap.Uint("db_user_id", oauthState.UserID),
		)
		return nil, usererrors.ErrInvalidState
	}

	// Step 5: Mark state as consumed immediately (one-time use)
	if err := uc.stateRepo.MarkAsConsumed(ctx, input.State, input.InstallationID); err != nil {
		uc.logger.Error(ctx, "failed to mark state as consumed",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.Int64("installation_id", input.InstallationID),
		)
		return nil, fmt.Errorf("failed to consume state: %w", err)
	}

	uc.logger.Info(ctx, "state validated and consumed successfully",
		zap.Uint("user_id", userID),
		zap.Int64("installation_id", input.InstallationID),
	)

	// Step 6: Get installation info from GitHub API
	installationInfo, err := uc.githubClient.GetInstallationInfo(input.InstallationID)
	if err != nil {
		uc.logger.Error(ctx, "failed to get installation info from GitHub API",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.Int64("installation_id", input.InstallationID),
		)
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

				uc.logger.Info(ctx, "creating new installation",
					zap.Uint("user_id", userID),
					zap.Int64("installation_id", input.InstallationID),
					zap.String("account_login", installationInfo.Account.Login),
				)

				if err := uc.installationRepo.Create(txCtx, installation); err != nil {
					uc.logger.Error(ctx, "failed to create installation",
						zap.Error(err),
						zap.Uint("user_id", userID),
						zap.Int64("installation_id", input.InstallationID),
					)
					return fmt.Errorf("failed to create installation: %w", err)
				}
			} else {
				uc.logger.Error(ctx, "failed to find installation",
					zap.Error(err),
					zap.Int64("installation_id", input.InstallationID),
				)
				return fmt.Errorf("failed to find installation: %w", err)
			}
		} else {
			// Installation exists - verify user ownership before proceeding
			if installation.UserID != userID {
				uc.logger.Error(ctx, "installation ownership mismatch",
					zap.Uint("current_user_id", userID),
					zap.Uint("installation_user_id", installation.UserID),
					zap.Int64("installation_id", input.InstallationID),
				)
				return usererrors.ErrInstallationUnauthorized
			}

			// Installation exists and owned by user - reactivate if deleted or revoked
			if installation.IsDeleted || installation.Status == model.InstallationStatusRevoked {
				uc.logger.Info(ctx, "reactivating deleted or revoked installation",
					zap.Uint("user_id", userID),
					zap.Int64("installation_id", input.InstallationID),
					zap.Bool("is_deleted", installation.IsDeleted),
					zap.String("status", string(installation.Status)),
				)

				// Reactivate deleted or revoked installation
				// This restores is_deleted=FALSE, deleted_at=NULL, and status='active'
				if err := uc.installationRepo.Reactivate(
					txCtx,
					input.InstallationID,
					installationInfo.Account.Login,
					model.AccountType(installationInfo.Account.Type),
				); err != nil {
					uc.logger.Error(ctx, "failed to reactivate installation",
						zap.Error(err),
						zap.Uint("user_id", userID),
						zap.Int64("installation_id", input.InstallationID),
					)
					return fmt.Errorf("failed to reactivate installation: %w", err)
				}
				// Update local installation object for response
				installation.Status = model.InstallationStatusActive
				installation.AccountLogin = installationInfo.Account.Login
				installation.AccountType = model.AccountType(installationInfo.Account.Type)
				installation.IsDeleted = false
				installation.DeletedAt = nil
			} else {
				uc.logger.Info(ctx, "updating existing installation",
					zap.Uint("user_id", userID),
					zap.Int64("installation_id", input.InstallationID),
					zap.String("account_login", installationInfo.Account.Login),
				)

				// Update active installation
				installation.AccountLogin = installationInfo.Account.Login
				installation.AccountType = model.AccountType(installationInfo.Account.Type)
				installation.Status = model.InstallationStatusActive
				now := time.Now()
				installation.UpdatedAt = &now

				if err := uc.installationRepo.Update(txCtx, installation); err != nil {
					uc.logger.Error(ctx, "failed to update installation",
						zap.Error(err),
						zap.Uint("user_id", userID),
						zap.Int64("installation_id", input.InstallationID),
					)
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

		uc.logger.Info(ctx, "installation processed successfully",
			zap.Uint("user_id", userID),
			zap.Int64("installation_id", installation.InstallationID),
			zap.String("account_login", installation.AccountLogin),
			zap.Bool("is_new", isNew),
		)

		return nil
	})

	if err != nil {
		return nil, err
	}

	uc.logger.Info(ctx, "installation callback completed",
		zap.Uint("user_id", uint(userID)),
		zap.Int64("installation_id", result.InstallationID),
		zap.Bool("is_new", result.IsNew),
	)

	return &InstallationCallbackOutput{
		UserID:       uint(userID),
		Installation: result,
	}, nil
}
