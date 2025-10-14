package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

type MockKubeClient struct {
	mock.Mock
}

func (m *MockKubeClient) GetPipelineRunStatus(ctx context.Context, pipelineRunName string) (*dto.PipelineRunStatus, error) {
	args := m.Called(ctx, pipelineRunName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PipelineRunStatus), args.Error(1)
}

func (m *MockKubeClient) GetPipelineRunLogs(ctx context.Context, pipelineRunName string) (string, error) {
	args := m.Called(ctx, pipelineRunName)
	return args.String(0), args.Error(1)
}

func (m *MockKubeClient) ListPipelineRuns(ctx context.Context, projectID uint) ([]*dto.PipelineRunInfo, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*dto.PipelineRunInfo), args.Error(1)
}
