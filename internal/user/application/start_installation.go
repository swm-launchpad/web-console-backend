package application

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth/state"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
)

type StartInstallationInput struct {
	UserID uint
}

type StartInstallationOutput struct {
	InstallationURL string `json:"installation_url"`
	State           string `json:"state"`
}

type StartInstallationUseCase struct {
	config         *config.Config
	stateValidator *state.StateValidator
	stateRepo      repository.OAuthStateRepository
	logger         logger.Logger
}

func NewStartInstallationUseCase(
	cfg *config.Config,
	stateRepo repository.OAuthStateRepository,
	log logger.Logger,
) *StartInstallationUseCase {
	return &StartInstallationUseCase{
		config:         cfg,
		stateValidator: state.NewStateValidator(cfg.JWT.Secret),
		stateRepo:      stateRepo,
		logger:         log,
	}
}

func (uc *StartInstallationUseCase) Execute(ctx context.Context, input StartInstallationInput) (*StartInstallationOutput, error) {
	uc.logger.Info(ctx, "start installation started",
		zap.Uint("user_id", input.UserID),
	)

	if input.UserID == 0 {
		uc.logger.Error(ctx, "user ID is required",
			zap.Uint("user_id", input.UserID),
		)
		return nil, usererrors.ErrUserIDRequired
	}

	// Validate GitHub App configuration
	if uc.config.GitHubApp.InstallationURL == "" ||
		uc.config.GitHubApp.AppID == "" ||
		uc.config.GitHubApp.PrivateKeyPath == "" {
		uc.logger.Error(ctx, "github app not configured",
			zap.Uint("user_id", input.UserID),
		)
		return nil, usererrors.ErrGitHubNotConfigured
	}

	// Generate HMAC-signed state for CSRF protection
	// Format: random:timestamp:userID:signature
	signedState, err := uc.stateValidator.GenerateState(input.UserID)
	if err != nil {
		uc.logger.Error(ctx, "failed to generate state",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state in database for server-side verification (prevents replay attacks)
	oauthState := &model.OAuthState{
		State:          signedState,
		UserID:         input.UserID,
		InstallationID: nil, // Will be set during callback
		ExpiresAt:      time.Now().Add(10 * time.Minute),
		CreatedAt:      time.Now(),
		ConsumedAt:     nil,
	}

	if err := uc.stateRepo.Create(ctx, oauthState); err != nil {
		uc.logger.Error(ctx, "failed to store state in database",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, fmt.Errorf("failed to store state: %w", err)
	}

	// Build GitHub App Installation URL
	githubInstallationURL := uc.config.GitHubApp.InstallationURL

	params := url.Values{}
	params.Set("state", signedState)
	// Note: GitHub App Installation does not support redirect_uri parameter
	// For development, manually modify the callback URL after GitHub redirects

	installationURL := fmt.Sprintf("%s?%s", githubInstallationURL, params.Encode())

	uc.logger.Info(ctx, "start installation completed",
		zap.Uint("user_id", input.UserID),
		zap.String("state_prefix", signedState[:min(10, len(signedState))]),
	)

	return &StartInstallationOutput{
		InstallationURL: installationURL,
		State:           signedState,
	}, nil
}
