package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure"
)

func TestStartInstallationUseCase_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	userID := uint(1)

	// Mock configuration with GitHub App settings
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-hmac",
		},
		GitHubApp: config.GitHubAppConfig{
			InstallationURL: "https://github.com/apps/test-app/installations/new",
			AppID:           "123456",
			PrivateKeyPath:  "/path/to/key.pem",
		},
	}

	mockStateRepo := new(infrastructure.MockOAuthStateRepository)
	mockStateRepo.On("Create", ctx, mock.MatchedBy(func(state *model.OAuthState) bool {
		return state.UserID == userID &&
			state.State != "" &&
			state.InstallationID == nil &&
			state.ConsumedAt == nil
	})).Return(nil)

	testLogger := logger.NewForTest()
	useCase := NewStartInstallationUseCase(cfg, mockStateRepo, testLogger)

	// Execute
	input := StartInstallationInput{UserID: userID}
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.InstallationURL, "Installation URL should not be empty")
	assert.NotEmpty(t, output.State, "State token should not be empty")
	assert.Contains(t, output.InstallationURL, "github.com", "URL should contain GitHub domain")
	assert.Contains(t, output.InstallationURL, "state=", "URL should contain state parameter")

	mockStateRepo.AssertExpectations(t)
	mockStateRepo.AssertCalled(t, "Create", ctx, mock.Anything)
}

func TestStartInstallationUseCase_GitHubNotConfigured(t *testing.T) {
	// Setup
	ctx := context.Background()
	userID := uint(1)

	// Mock configuration WITHOUT GitHub App settings
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
		GitHubApp: config.GitHubAppConfig{
			// Empty - not configured
		},
	}

	mockStateRepo := new(infrastructure.MockOAuthStateRepository)

	testLogger := logger.NewForTest()
	useCase := NewStartInstallationUseCase(cfg, mockStateRepo, testLogger)

	// Execute
	input := StartInstallationInput{UserID: userID}
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Equal(t, usererrors.ErrGitHubNotConfigured, err)

	// Should not call state repository if config is invalid
	mockStateRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestStartInstallationUseCase_InvalidUserID(t *testing.T) {
	// Setup
	ctx := context.Background()

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret: "test-secret-key",
		},
		GitHubApp: config.GitHubAppConfig{
			InstallationURL: "https://github.com/apps/test-app/installations/new",
			AppID:           "123456",
			PrivateKeyPath:  "/path/to/key.pem",
		},
	}

	mockStateRepo := new(infrastructure.MockOAuthStateRepository)

	testLogger := logger.NewForTest()
	useCase := NewStartInstallationUseCase(cfg, mockStateRepo, testLogger)

	// Execute with invalid user ID (0)
	input := StartInstallationInput{UserID: 0}
	output, err := useCase.Execute(ctx, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
	assert.Equal(t, usererrors.ErrUserIDRequired, err)

	mockStateRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}
