package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure"
)

// MockTektonNodePortClient is a mock implementation of TektonNodePortClient
type MockTektonNodePortClient struct {
	mock.Mock
}

// TriggerNodePortCreation mocks the TriggerNodePortCreation method
func (m *MockTektonNodePortClient) TriggerNodePortCreation(ctx context.Context, params infrastructure.NodePortCreationParams) (string, error) {
	args := m.Called(ctx, params)
	return args.String(0), args.Error(1)
}

// GetPipelineRunResult mocks the GetPipelineRunResult method
func (m *MockTektonNodePortClient) GetPipelineRunResult(ctx context.Context, eventID string) (*infrastructure.NodePortInfo, error) {
	args := m.Called(ctx, eventID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infrastructure.NodePortInfo), args.Error(1)
}

// GetNodePortService mocks the GetNodePortService method
func (m *MockTektonNodePortClient) GetNodePortService(ctx context.Context, serviceName string, namespace string) (*infrastructure.NodePortInfo, error) {
	args := m.Called(ctx, serviceName, namespace)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*infrastructure.NodePortInfo), args.Error(1)
}
