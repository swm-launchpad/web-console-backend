package service

import (
	"context"

	"github.com/stretchr/testify/mock"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
)

type MockVolumeService struct {
	mock.Mock
}

func (m *MockVolumeService) CreateVolume(ctx context.Context, projectID uint, name string, capacity uint32) (*model.Volume, error) {
	args := m.Called(ctx, projectID, name, capacity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Volume), args.Error(1)
}

func (m *MockVolumeService) GetVolume(ctx context.Context, volumeID uint) (*model.Volume, error) {
	args := m.Called(ctx, volumeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Volume), args.Error(1)
}

func (m *MockVolumeService) ListVolumesByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Volume), args.Error(1)
}

func (m *MockVolumeService) DeleteVolume(ctx context.Context, volumeID uint) error {
	args := m.Called(ctx, volumeID)
	return args.Error(0)
}

func (m *MockVolumeService) DeleteVolumesByProjectID(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}
