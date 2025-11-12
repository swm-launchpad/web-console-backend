package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
)

// MockProjectRepository is a mock implementation of ProjectRepository
// This is a re-export of the project repository mock for testing purposes
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) Create(ctx context.Context, project *projectmodel.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) Save(ctx context.Context, project *projectmodel.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) FindByID(ctx context.Context, projectID uint) (*projectmodel.Project, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*projectmodel.Project), args.Error(1)
}

func (m *MockProjectRepository) FindByIDForUpdate(ctx context.Context, projectID uint) (*projectmodel.Project, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*projectmodel.Project), args.Error(1)
}

func (m *MockProjectRepository) FindByUserID(ctx context.Context, userID uint) ([]*projectmodel.Project, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*projectmodel.Project), args.Error(1)
}

func (m *MockProjectRepository) FindBySlug(ctx context.Context, slug string) (*projectmodel.Project, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*projectmodel.Project), args.Error(1)
}

func (m *MockProjectRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectRepository) ExistsByNameAndUserID(ctx context.Context, name string, userID uint) (bool, error) {
	args := m.Called(ctx, name, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectRepository) Delete(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockProjectRepository) FindProjectsWithActiveOperations(ctx context.Context) ([]*projectmodel.Project, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*projectmodel.Project), args.Error(1)
}
