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

func (m *MockContainerClient) GetContainerConfigs(ctx context.Context, projectID uint) (*dto.ContainerBuildConfig, *dto.ContainerDeploymentConfig, error) {
	args := m.Called(ctx, projectID)

	var buildConfig *dto.ContainerBuildConfig
	if args.Get(0) != nil {
		buildConfig = args.Get(0).(*dto.ContainerBuildConfig)
	}

	var deployConfig *dto.ContainerDeploymentConfig
	if args.Get(1) != nil {
		deployConfig = args.Get(1).(*dto.ContainerDeploymentConfig)
	}

	return buildConfig, deployConfig, args.Error(2)
}

func (m *MockContainerClient) GetUnifiedContainerConfig(ctx context.Context, projectID uint) (*dto.UnifiedContainerConfig, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.UnifiedContainerConfig), args.Error(1)
}
