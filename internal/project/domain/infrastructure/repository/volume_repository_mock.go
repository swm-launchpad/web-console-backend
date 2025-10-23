package repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
)

type MockVolumeRepository struct {
	mock.Mock
}

func (m *MockVolumeRepository) Create(ctx context.Context, volume *model.Volume) error {
	args := m.Called(ctx, volume)
	return args.Error(0)
}

func (m *MockVolumeRepository) FindByID(ctx context.Context, volumeID uint) (*model.Volume, error) {
	args := m.Called(ctx, volumeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Volume), args.Error(1)
}

func (m *MockVolumeRepository) FindByProjectID(ctx context.Context, projectID uint) ([]*model.Volume, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Volume), args.Error(1)
}

func (m *MockVolumeRepository) FindByName(ctx context.Context, projectID uint, name string) (*model.Volume, error) {
	args := m.Called(ctx, projectID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Volume), args.Error(1)
}

func (m *MockVolumeRepository) FindBySlug(ctx context.Context, slug string) (*model.Volume, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Volume), args.Error(1)
}

func (m *MockVolumeRepository) ExistsByName(ctx context.Context, projectID uint, name string) (bool, error) {
	args := m.Called(ctx, projectID, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockVolumeRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

func (m *MockVolumeRepository) Delete(ctx context.Context, volumeID uint) error {
	args := m.Called(ctx, volumeID)
	return args.Error(0)
}

func (m *MockVolumeRepository) DeleteByProjectID(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockVolumeRepository) List(ctx context.Context, offset, limit int) ([]*model.Volume, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Volume), args.Error(1)
}

func (m *MockVolumeRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockVolumeRepository) CountByProjectID(ctx context.Context, projectID uint) (int64, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockVolumeRepository) GetTotalCapacityByProjectID(ctx context.Context, projectID uint) (uint32, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(uint32), args.Error(1)
}
