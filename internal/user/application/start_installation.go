package application

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth/state"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
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
}

func NewStartInstallationUseCase(
	cfg *config.Config,
	stateRepo repository.OAuthStateRepository,
) *StartInstallationUseCase {
	return &StartInstallationUseCase{
		config:         cfg,
		stateValidator: state.NewStateValidator(cfg.JWT.Secret),
		stateRepo:      stateRepo,
	}
}

func (uc *StartInstallationUseCase) Execute(ctx context.Context, input StartInstallationInput) (*StartInstallationOutput, error) {
	if input.UserID == 0 {
		return nil, usererrors.ErrUserIDRequired
	}

	// Validate GitHub App configuration
	if uc.config.GitHubApp.InstallationURL == "" ||
		uc.config.GitHubApp.AppID == "" ||
		uc.config.GitHubApp.PrivateKeyPath == "" {
		return nil, usererrors.ErrGitHubNotConfigured
	}

	// Generate HMAC-signed state for CSRF protection
	// Format: random:timestamp:userID:signature
	signedState, err := uc.stateValidator.GenerateState(input.UserID)
	if err != nil {
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
		return nil, fmt.Errorf("failed to store state: %w", err)
	}

	// Build GitHub App Installation URL
	githubInstallationURL := uc.config.GitHubApp.InstallationURL

	params := url.Values{}
	params.Set("state", signedState)
	// Note: GitHub App Installation does not support redirect_uri parameter
	// For development, manually modify the callback URL after GitHub redirects

	installationURL := fmt.Sprintf("%s?%s", githubInstallationURL, params.Encode())

	return &StartInstallationOutput{
		InstallationURL: installationURL,
		State:           signedState,
	}, nil
}
