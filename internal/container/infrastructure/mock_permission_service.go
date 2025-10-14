package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockPermissionService is a mock implementation of PermissionService interface
type MockPermissionService struct {
	mock.Mock
}

// CanUserModifyContainer mocks the CanUserModifyContainer method
func (m *MockPermissionService) CanUserModifyContainer(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}

// CanUserAccessContainer mocks the CanUserAccessContainer method
func (m *MockPermissionService) CanUserAccessContainer(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}

// CanUserCreateContainer mocks the CanUserCreateContainer method
func (m *MockPermissionService) CanUserCreateContainer(ctx context.Context, userID uint, projectID uint) error {
	args := m.Called(ctx, userID, projectID)
	return args.Error(0)
}

// CanUserManageEnvVars mocks the CanUserManageEnvVars method
func (m *MockPermissionService) CanUserManageEnvVars(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}

// CanUserManageNetworks mocks the CanUserManageNetworks method
func (m *MockPermissionService) CanUserManageNetworks(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}

// CanUserManageMounts mocks the CanUserManageMounts method
func (m *MockPermissionService) CanUserManageMounts(ctx context.Context, userID uint, containerID uint) error {
	args := m.Called(ctx, userID, containerID)
	return args.Error(0)
}
