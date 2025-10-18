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

type StartOAuthInput struct {
	UserID uint
}

type StartOAuthOutput struct {
	AuthorizeURL string `json:"authorize_url"`
	State        string `json:"state"`
}

type StartOAuthUseCase struct {
	config *config.Config
}

func NewStartOAuthUseCase(cfg *config.Config) *StartOAuthUseCase {
	return &StartOAuthUseCase{
		config: cfg,
	}
}

func (uc *StartOAuthUseCase) Execute(ctx context.Context, input StartOAuthInput) (*StartOAuthOutput, error) {
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

	// Build GitHub App Installation URL with state and redirect_uri parameters
	// This allows users to install the app on multiple organizations/accounts
	// After installation, GitHub will redirect to OAuth callback with code
	// (if "Request user authorization (OAuth) during installation" is enabled)
	githubInstallationURL := uc.config.GitHubApp.InstallationURL

	// Build callback URL using backend base URL
	callbackURL := fmt.Sprintf("%s/api/v1/github/oauth/callback", uc.config.Server.BaseURL)

	params := url.Values{}
	params.Set("state", state)
	params.Set("redirect_uri", callbackURL)

	authorizeURL := fmt.Sprintf("%s?%s", githubInstallationURL, params.Encode())

	return &StartOAuthOutput{
		AuthorizeURL: authorizeURL,
		State:        state,
	}, nil
}
