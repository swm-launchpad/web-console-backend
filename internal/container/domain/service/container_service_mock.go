package service

import (
	"context"

	"github.com/stretchr/testify/mock"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

type MockContainerService struct {
	mock.Mock
}

func (m *MockContainerService) CreateContainer(ctx context.Context, projectID uint, name string, gitConfig value.GitConfig, resourceLimits value.ResourceLimits, templateID *uint, templateConfig map[string]interface{}, githubInstallationID *int64) (*model.Container, error) {
	args := m.Called(ctx, projectID, name, gitConfig, resourceLimits, templateID, templateConfig, githubInstallationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Container), args.Error(1)
}

func (m *MockContainerService) GetContainer(ctx context.Context, containerID uint) (*model.Container, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Container), args.Error(1)
}

func (m *MockContainerService) GetContainerBySlug(ctx context.Context, slug string) (*model.Container, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Container), args.Error(1)
}

func (m *MockContainerService) UpdateContainer(ctx context.Context, containerID uint, updateFn func(*model.Container) error) (*model.Container, error) {
	args := m.Called(ctx, containerID, updateFn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Container), args.Error(1)
}

func (m *MockContainerService) DeleteContainer(ctx context.Context, containerID uint) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockContainerService) ListContainersByProjectID(ctx context.Context, projectID uint) ([]*model.Container, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Container), args.Error(1)
}

func (m *MockContainerService) CountContainersByProjectID(ctx context.Context, projectID uint) (int, error) {
	args := m.Called(ctx, projectID)
	return args.Int(0), args.Error(1)
}

func (m *MockContainerService) CheckContainerNameExists(ctx context.Context, projectID uint, name string) (bool, error) {
	args := m.Called(ctx, projectID, name)
	return args.Bool(0), args.Error(1)
}
