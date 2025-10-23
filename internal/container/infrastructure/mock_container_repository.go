package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
)

// MockContainerRepository is a mock implementation of ContainerRepository interface
type MockContainerRepository struct {
	mock.Mock
}

// Create mocks the Create method
func (m *MockContainerRepository) Create(ctx context.Context, container *model.Container) error {
	args := m.Called(ctx, container)
	return args.Error(0)
}

// Save mocks the Save method
func (m *MockContainerRepository) Save(ctx context.Context, container *model.Container) error {
	args := m.Called(ctx, container)
	return args.Error(0)
}

// FindByID mocks the FindByID method
func (m *MockContainerRepository) FindByID(ctx context.Context, containerID uint) (*model.Container, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Container), args.Error(1)
}

// FindByIDForUpdate mocks the FindByIDForUpdate method
func (m *MockContainerRepository) FindByIDForUpdate(ctx context.Context, containerID uint) (*model.Container, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Container), args.Error(1)
}

// FindByProjectID mocks the FindByProjectID method
func (m *MockContainerRepository) FindByProjectID(ctx context.Context, projectID uint) ([]*model.Container, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Container), args.Error(1)
}

// FindBySlug mocks the FindBySlug method
func (m *MockContainerRepository) FindBySlug(ctx context.Context, slug string) (*model.Container, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Container), args.Error(1)
}

// ExistsBySlug mocks the ExistsBySlug method
func (m *MockContainerRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

// ExistsByNameAndProjectID mocks the ExistsByNameAndProjectID method
func (m *MockContainerRepository) ExistsByNameAndProjectID(ctx context.Context, projectID uint, name string) (bool, error) {
	args := m.Called(ctx, projectID, name)
	return args.Bool(0), args.Error(1)
}

// Delete mocks the Delete method
func (m *MockContainerRepository) Delete(ctx context.Context, containerID uint) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

// DeleteByProjectID mocks the DeleteByProjectID method
func (m *MockContainerRepository) DeleteByProjectID(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

// List mocks the List method
func (m *MockContainerRepository) List(ctx context.Context, offset, limit int) ([]*model.Container, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Container), args.Error(1)
}

// Count mocks the Count method
func (m *MockContainerRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// CountByProjectID mocks the CountByProjectID method
func (m *MockContainerRepository) CountByProjectID(ctx context.Context, projectID uint) (int64, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(int64), args.Error(1)
}

// CountByTemplateID mocks the CountByTemplateID method
func (m *MockContainerRepository) CountByTemplateID(ctx context.Context, templateID uint) (int64, error) {
	args := m.Called(ctx, templateID)
	return args.Get(0).(int64), args.Error(1)
}

// GetTotalResourceUsageByProject mocks the GetTotalResourceUsageByProject method
func (m *MockContainerRepository) GetTotalResourceUsageByProject(ctx context.Context, projectID uint) (totalCPU uint32, totalMemory uint32, err error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).(uint32), args.Get(1).(uint32), args.Error(2)
}
