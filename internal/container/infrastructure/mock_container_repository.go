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

// CheckInternalPortExistsInProject mocks the CheckInternalPortExistsInProject method
func (m *MockContainerRepository) CheckInternalPortExistsInProject(ctx context.Context, projectID uint, internalPort uint16) (bool, error) {
	args := m.Called(ctx, projectID, internalPort)
	return args.Bool(0), args.Error(1)
}

// CheckFQDNExists mocks the CheckFQDNExists method
func (m *MockContainerRepository) CheckFQDNExists(ctx context.Context, fqdn string) (bool, error) {
	args := m.Called(ctx, fqdn)
	return args.Bool(0), args.Error(1)
}

// FindAllSlugsByProjectIDIncludingDeleted mocks the FindAllSlugsByProjectIDIncludingDeleted method
func (m *MockContainerRepository) FindAllSlugsByProjectIDIncludingDeleted(ctx context.Context, projectID uint) ([]string, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// CheckFQDNExistsInOtherProject mocks the CheckFQDNExistsInOtherProject method
func (m *MockContainerRepository) CheckFQDNExistsInOtherProject(ctx context.Context, fqdn string, projectID uint) (bool, error) {
	args := m.Called(ctx, fqdn, projectID)
	return args.Bool(0), args.Error(1)
}

// CheckFQDNExistsInOtherProjectExcludingSelf mocks the CheckFQDNExistsInOtherProjectExcludingSelf method
func (m *MockContainerRepository) CheckFQDNExistsInOtherProjectExcludingSelf(ctx context.Context, fqdn string, networkID uint, projectID uint) (bool, error) {
	args := m.Called(ctx, fqdn, networkID, projectID)
	return args.Bool(0), args.Error(1)
}

// CheckInternalPortExistsInProjectExcludingSelf mocks the CheckInternalPortExistsInProjectExcludingSelf method
func (m *MockContainerRepository) CheckInternalPortExistsInProjectExcludingSelf(ctx context.Context, projectID uint, internalPort uint16, networkID uint) (bool, error) {
	args := m.Called(ctx, projectID, internalPort, networkID)
	return args.Bool(0), args.Error(1)
}

// SoftDeleteNetworksByContainerID mocks the SoftDeleteNetworksByContainerID method
func (m *MockContainerRepository) SoftDeleteNetworksByContainerID(ctx context.Context, containerID uint) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

// BeginTx mocks the BeginTx method
func (m *MockContainerRepository) BeginTx(ctx context.Context) (context.Context, interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(context.Context), args.Get(1), args.Error(2)
}

// Commit mocks the Commit method
func (m *MockContainerRepository) Commit(ctx context.Context, tx interface{}) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// Rollback mocks the Rollback method
func (m *MockContainerRepository) Rollback(ctx context.Context, tx interface{}) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

// HasProjectPermission mocks the HasProjectPermission method
func (m *MockContainerRepository) HasProjectPermission(ctx context.Context, userID uint, containerID uint) (bool, error) {
	args := m.Called(ctx, userID, containerID)
	return args.Bool(0), args.Error(1)
}

// GetContainerByID mocks the GetContainerByID method
func (m *MockContainerRepository) GetContainerByID(ctx context.Context, containerID uint) (*model.Container, error) {
	args := m.Called(ctx, containerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Container), args.Error(1)
}

// UpdateNetwork mocks the UpdateNetwork method
func (m *MockContainerRepository) UpdateNetwork(ctx context.Context, network *model.Network) error {
	args := m.Called(ctx, network)
	return args.Error(0)
}

// CheckFQDNExistsForProject mocks the CheckFQDNExistsForProject method
func (m *MockContainerRepository) CheckFQDNExistsForProject(ctx context.Context, fqdn string, projectID uint) (bool, error) {
	args := m.Called(ctx, fqdn, projectID)
	return args.Bool(0), args.Error(1)
}

// CheckFQDNExistsForProjectExcludingSelf mocks the CheckFQDNExistsForProjectExcludingSelf method
func (m *MockContainerRepository) CheckFQDNExistsForProjectExcludingSelf(ctx context.Context, fqdn string, networkID uint, projectID uint) (bool, error) {
	args := m.Called(ctx, fqdn, networkID, projectID)
	return args.Bool(0), args.Error(1)
}
