package service

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) CanUserModifyContainer(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}

func (m *MockPermissionService) CanUserAccessContainer(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}

func (m *MockPermissionService) CanUserCreateContainer(ctx context.Context, userID uint, projectID uint) error {
	args := m.Called(ctx, userID, projectID)
	return args.Error(0)
}

func (m *MockPermissionService) CanUserManageEnvVars(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}

func (m *MockPermissionService) CanUserManageNetworks(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}

func (m *MockPermissionService) CanUserManageMounts(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}
