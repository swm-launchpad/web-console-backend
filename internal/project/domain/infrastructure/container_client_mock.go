package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

type MockContainerClient struct {
	mock.Mock
}

func (m *MockContainerClient) GetContainerConfig(ctx context.Context, projectID uint) (*dto.ContainerDeploymentConfig, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ContainerDeploymentConfig), args.Error(1)
}

func (m *MockContainerClient) GetContainerBuildConfig(ctx context.Context, projectID uint) (*dto.ContainerBuildConfig, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.ContainerBuildConfig), args.Error(1)
}
