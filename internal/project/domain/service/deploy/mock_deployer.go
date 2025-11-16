package deploy

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
)

// MockDeployer is a mock implementation of the Deployer interface for testing
type MockDeployer struct {
	mock.Mock
}

// DeployProject mocks the DeployProject method
func (m *MockDeployer) DeployProject(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deployment.Deployment), args.Error(1)
}

// GetDeploymentStatus mocks the GetDeploymentStatus method
func (m *MockDeployer) GetDeploymentStatus(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deployment.Deployment), args.Error(1)
}

// RefreshActiveDeployment mocks the RefreshActiveDeployment method
func (m *MockDeployer) RefreshActiveDeployment(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deployment.Deployment), args.Error(1)
}

// RefreshActiveBuildStatuses mocks the RefreshActiveBuildStatuses method
func (m *MockDeployer) RefreshActiveBuildStatuses(ctx context.Context, projectID uint) ([]*build_history.BuildHistory, bool, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]*build_history.BuildHistory), args.Bool(1), args.Error(2)
}

// BuildAndDeployProject mocks the BuildAndDeployProject method
func (m *MockDeployer) BuildAndDeployProject(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}
