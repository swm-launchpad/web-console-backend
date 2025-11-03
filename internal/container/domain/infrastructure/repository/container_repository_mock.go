package repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	containermodel "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
)

type MockContainerRepository struct {
	mock.Mock
}

func (m *MockContainerRepository) Create(ctx context.Context, container *containermodel.Container) error {
	args := m.Called(ctx, container)
	return args.Error(0)
}

func (m *MockContainerRepository) Save(ctx context.Context, container *containermodel.Container) error {
	args := m.Called(ctx, container)
	return args.Error(0)
}

func (m *MockContainerRepository) FindByID(ctx context.Context, containerID uint) (*containermodel.Container, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*containermodel.Container), args.Error(1)
}

func (m *MockContainerRepository) FindByIDForUpdate(ctx context.Context, containerID uint) (*containermodel.Container, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*containermodel.Container), args.Error(1)
}

func (m *MockContainerRepository) FindByProjectID(ctx context.Context, projectID uint) ([]*containermodel.Container, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*containermodel.Container), args.Error(1)
}

func (m *MockContainerRepository) FindBySlug(ctx context.Context, slug string) (*containermodel.Container, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*containermodel.Container), args.Error(1)
}

func (m *MockContainerRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

func (m *MockContainerRepository) ExistsByNameAndProjectID(ctx context.Context, projectID uint, name string) (bool, error) {
	args := m.Called(ctx, projectID, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockContainerRepository) Delete(ctx context.Context, containerID uint) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

func (m *MockContainerRepository) DeleteByProjectID(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockContainerRepository) List(ctx context.Context, offset, limit int) ([]*containermodel.Container, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*containermodel.Container), args.Error(1)
}

func (m *MockContainerRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockContainerRepository) CountByProjectID(ctx context.Context, projectID uint) (int64, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockContainerRepository) CountByTemplateID(ctx context.Context, templateID uint) (int64, error) {
	args := m.Called(ctx, templateID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockContainerRepository) GetTotalResourceUsageByProject(ctx context.Context, projectID uint) (uint32, uint32, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(uint32), args.Get(1).(uint32), args.Error(2)
}

func (m *MockContainerRepository) CheckInternalPortExistsInProject(ctx context.Context, projectID uint, internalPort uint16) (bool, error) {
	args := m.Called(ctx, projectID, internalPort)
	return args.Bool(0), args.Error(1)
}

func (m *MockContainerRepository) CheckFQDNExists(ctx context.Context, fqdn string) (bool, error) {
	args := m.Called(ctx, fqdn)
	return args.Bool(0), args.Error(1)
}
