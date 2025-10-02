package service

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockPermissionService struct {
	mock.Mock
}

func (m *MockPermissionService) CanUserModifyProject(ctx context.Context, userID uint, projectID uint) error {
	args := m.Called(ctx, userID, projectID)
	return args.Error(0)
}

func (m *MockPermissionService) CanUserAccessProject(ctx context.Context, userID uint, projectID uint) error {
	args := m.Called(ctx, userID, projectID)
	return args.Error(0)
}

func (m *MockPermissionService) CanUserAddVolume(ctx context.Context, userID uint, projectID uint) error {
	args := m.Called(ctx, userID, projectID)
	return args.Error(0)
}

func (m *MockPermissionService) CanUserRemoveVolume(ctx context.Context, userID uint, volumeID uint) error {
	args := m.Called(ctx, userID, volumeID)
	return args.Error(0)
}

func (m *MockPermissionService) CanUserAccessVolume(ctx context.Context, userID uint, volumeID uint) error {
	args := m.Called(ctx, userID, volumeID)
	return args.Error(0)
}
