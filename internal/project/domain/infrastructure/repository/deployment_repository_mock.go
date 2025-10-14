package repository

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
)

type MockDeploymentRepository struct {
	mock.Mock
}

func (m *MockDeploymentRepository) Create(ctx context.Context, d *deployment.Deployment) error {
	args := m.Called(ctx, d)
	return args.Error(0)
}

func (m *MockDeploymentRepository) Save(ctx context.Context, d *deployment.Deployment) error {
	args := m.Called(ctx, d)
	return args.Error(0)
}

func (m *MockDeploymentRepository) FindByID(ctx context.Context, deploymentID uint) (*deployment.Deployment, error) {
	args := m.Called(ctx, deploymentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deployment.Deployment), args.Error(1)
}

func (m *MockDeploymentRepository) FindLatestByProjectID(ctx context.Context, projectID uint) (*deployment.Deployment, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deployment.Deployment), args.Error(1)
}

func (m *MockDeploymentRepository) FindByProjectID(ctx context.Context, projectID uint, limit, offset int) ([]*deployment.Deployment, error) {
	args := m.Called(ctx, projectID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*deployment.Deployment), args.Error(1)
}

func (m *MockDeploymentRepository) FindByTektonPipelineRunName(ctx context.Context, pipelineRunName string) (*deployment.Deployment, error) {
	args := m.Called(ctx, pipelineRunName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*deployment.Deployment), args.Error(1)
}

func (m *MockDeploymentRepository) FindActiveDeploymentsByProjectID(ctx context.Context, projectID uint) ([]*deployment.Deployment, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*deployment.Deployment), args.Error(1)
}
