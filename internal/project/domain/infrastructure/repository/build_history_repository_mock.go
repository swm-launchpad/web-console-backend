package repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
)

type MockBuildHistoryRepository struct {
	mock.Mock
}

func (m *MockBuildHistoryRepository) Create(ctx context.Context, b *build_history.BuildHistory) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBuildHistoryRepository) Save(ctx context.Context, b *build_history.BuildHistory) error {
	args := m.Called(ctx, b)
	return args.Error(0)
}

func (m *MockBuildHistoryRepository) FindByID(ctx context.Context, buildHistoryID uint) (*build_history.BuildHistory, error) {
	args := m.Called(ctx, buildHistoryID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*build_history.BuildHistory), args.Error(1)
}

func (m *MockBuildHistoryRepository) FindLatestByContainerID(ctx context.Context, containerID uint) (*build_history.BuildHistory, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*build_history.BuildHistory), args.Error(1)
}

func (m *MockBuildHistoryRepository) FindByContainerID(ctx context.Context, containerID uint, limit, offset int) ([]*build_history.BuildHistory, error) {
	args := m.Called(ctx, containerID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*build_history.BuildHistory), args.Error(1)
}

func (m *MockBuildHistoryRepository) FindByTektonPipelineRunName(ctx context.Context, pipelineRunName string) (*build_history.BuildHistory, error) {
	args := m.Called(ctx, pipelineRunName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*build_history.BuildHistory), args.Error(1)
}

func (m *MockBuildHistoryRepository) FindActiveByContainerID(ctx context.Context, containerID uint) ([]*build_history.BuildHistory, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*build_history.BuildHistory), args.Error(1)
}
