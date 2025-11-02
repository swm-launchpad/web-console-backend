package service

import (
	"context"

	"github.com/stretchr/testify/mock"
	model "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
)

type MockProjectService struct {
	mock.Mock
}

func (m *MockProjectService) CreateProject(ctx context.Context, name string, ownerID uint, limits value.ResourceLimits, plan *value.Plan) (*model.Project, error) {
	args := m.Called(ctx, name, ownerID, limits, plan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) GetProject(ctx context.Context, projectID uint) (*model.Project, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) GetProjectBySlug(ctx context.Context, slug string) (*model.Project, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) UpdateProject(ctx context.Context, projectID uint, actingUserID uint, updateFn func(*model.Project) error) (*model.Project, error) {
	args := m.Called(ctx, projectID, actingUserID, updateFn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) UpdateProjectBySlug(ctx context.Context, slug string, actingUserID uint, updateFn func(*model.Project) error) (*model.Project, error) {
	args := m.Called(ctx, slug, actingUserID, updateFn)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Project), args.Error(1)
}

func (m *MockProjectService) DeleteProject(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockProjectService) DeleteProjectBySlug(ctx context.Context, slug string) error {
	args := m.Called(ctx, slug)
	return args.Error(0)
}

func (m *MockProjectService) ListProjects(ctx context.Context, userID uint) ([]*model.Project, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Project), args.Error(1)
}

func (m *MockProjectService) CountProjectsByUserID(ctx context.Context, userID uint) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockProjectService) CheckProjectNameExists(ctx context.Context, name string, userID uint) (bool, error) {
	args := m.Called(ctx, name, userID)
	return args.Bool(0), args.Error(1)
}
