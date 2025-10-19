package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
)

// MockGitHubInstallationRepository is a mock implementation of GitHubInstallationRepository for testing
type MockGitHubInstallationRepository struct {
	mock.Mock
}

func (m *MockGitHubInstallationRepository) Create(ctx context.Context, installation *model.GitHubInstallation) error {
	args := m.Called(ctx, installation)
	return args.Error(0)
}

func (m *MockGitHubInstallationRepository) FindByInstallationID(ctx context.Context, installationID int64) (*model.GitHubInstallation, error) {
	args := m.Called(ctx, installationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitHubInstallation), args.Error(1)
}

func (m *MockGitHubInstallationRepository) FindByInstallationIDIncludingRevoked(ctx context.Context, installationID int64) (*model.GitHubInstallation, error) {
	args := m.Called(ctx, installationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.GitHubInstallation), args.Error(1)
}

func (m *MockGitHubInstallationRepository) FindByUserID(ctx context.Context, userID uint) ([]*model.GitHubInstallation, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.GitHubInstallation), args.Error(1)
}

func (m *MockGitHubInstallationRepository) Update(ctx context.Context, installation *model.GitHubInstallation) error {
	args := m.Called(ctx, installation)
	return args.Error(0)
}

func (m *MockGitHubInstallationRepository) Delete(ctx context.Context, installationID int64) error {
	args := m.Called(ctx, installationID)
	return args.Error(0)
}

func (m *MockGitHubInstallationRepository) ExistsByInstallationID(ctx context.Context, installationID int64) (bool, error) {
	args := m.Called(ctx, installationID)
	return args.Bool(0), args.Error(1)
}

func (m *MockGitHubInstallationRepository) MarkAsRevoked(ctx context.Context, installationID int64) error {
	args := m.Called(ctx, installationID)
	return args.Error(0)
}

func (m *MockGitHubInstallationRepository) Reactivate(ctx context.Context, installationID int64, accountLogin string, accountType model.AccountType) error {
	args := m.Called(ctx, installationID, accountLogin, accountType)
	return args.Error(0)
}

func (m *MockGitHubInstallationRepository) CacheToken(ctx context.Context, installationID int64, token string, expiresAt *string) error {
	args := m.Called(ctx, installationID, token, expiresAt)
	return args.Error(0)
}

func (m *MockGitHubInstallationRepository) InvalidateToken(ctx context.Context, installationID int64) error {
	args := m.Called(ctx, installationID)
	return args.Error(0)
}

func (m *MockGitHubInstallationRepository) ValidateUserOwnership(ctx context.Context, installationID int64, userID uint) error {
	args := m.Called(ctx, installationID, userID)
	return args.Error(0)
}
