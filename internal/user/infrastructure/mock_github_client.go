package infrastructure

import (
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/github"
)

// MockGitHubClient is a mock implementation of GitHub Client for testing
type MockGitHubClient struct {
	mock.Mock
}

func (m *MockGitHubClient) GetInstallationInfo(installationID int64) (*github.InstallationInfo, error) {
	args := m.Called(installationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.InstallationInfo), args.Error(1)
}

func (m *MockGitHubClient) GenerateInstallationToken(installationID int64) (*github.InstallationToken, error) {
	args := m.Called(installationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.InstallationToken), args.Error(1)
}

func (m *MockGitHubClient) ListInstallationRepositories(installationID int64) ([]*github.Repository, error) {
	args := m.Called(installationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*github.Repository), args.Error(1)
}
