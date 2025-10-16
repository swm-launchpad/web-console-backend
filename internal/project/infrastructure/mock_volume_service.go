package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
)

// MockVolumeService is a mock implementation of VolumeService
type MockVolumeService struct {
	mock.Mock
}

// CreateVolume creates a new volume
func (m *MockVolumeService) CreateVolume(ctx context.Context, projectID uint, name string, capacity uint32) (*model.Volume, error) {
	args := m.Called(ctx, projectID, name, capacity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Volume), args.Error(1)
}

// GetVolume retrieves a volume by ID
func (m *MockVolumeService) GetVolume(ctx context.Context, volumeID uint) (*model.Volume, error) {
	args := m.Called(ctx, volumeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Volume), args.Error(1)
}

// ListVolumesByProjectID retrieves all volumes for a project
func (m *MockVolumeService) ListVolumesByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Volume), args.Error(1)
}

// DeleteVolume removes a volume
func (m *MockVolumeService) DeleteVolume(ctx context.Context, volumeID uint) error {
	args := m.Called(ctx, volumeID)
	return args.Error(0)
}

// DeleteVolumesByProjectID removes all volumes for a project
func (m *MockVolumeService) DeleteVolumesByProjectID(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}
