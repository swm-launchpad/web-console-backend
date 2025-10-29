package deploy

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
)

// MockDeployService is a mock implementation of DeployService for testing
type MockDeployService struct {
	mock.Mock
}

// DeployProject mocks the DeployProject method
func (m *MockDeployService) DeployProject(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deployment.Deployment), args.Error(1)
}

// GetDeploymentStatus mocks the GetDeploymentStatus method
func (m *MockDeployService) GetDeploymentStatus(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deployment.Deployment), args.Error(1)
}

// RefreshActiveDeployment mocks the RefreshActiveDeployment method
func (m *MockDeployService) RefreshActiveDeployment(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deployment.Deployment), args.Error(1)
}

// BuildAndDeployProject mocks the BuildAndDeployProject method
func (m *MockDeployService) BuildAndDeployProject(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}
