package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
)

type StartInstallationInput struct {
	UserID uint
}

type StartInstallationOutput struct {
	InstallationURL string `json:"installation_url"`
	State           string `json:"state"`
}

type StartInstallationUseCase struct {
	config *config.Config
}

func NewStartInstallationUseCase(cfg *config.Config) *StartInstallationUseCase {
	return &StartInstallationUseCase{
		config: cfg,
	}
}

func (uc *StartInstallationUseCase) Execute(ctx context.Context, input StartInstallationInput) (*StartInstallationOutput, error) {
	if input.UserID == 0 {
		return nil, usererrors.ErrUserIDRequired
	}

	// Generate random state for CSRF protection
	// State format: base64(random_bytes) + ":" + user_id
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := fmt.Sprintf("%s:%d", base64.URLEncoding.EncodeToString(stateBytes), input.UserID)

	// Build GitHub App Installation URL
	githubInstallationURL := uc.config.GitHubApp.InstallationURL

	params := url.Values{}
	params.Set("state", state)
	// Note: GitHub App Installation does not support redirect_uri parameter
	// For development, manually modify the callback URL after GitHub redirects

	installationURL := fmt.Sprintf("%s?%s", githubInstallationURL, params.Encode())

	return &StartInstallationOutput{
		InstallationURL: installationURL,
		State:           state,
	}, nil
}
