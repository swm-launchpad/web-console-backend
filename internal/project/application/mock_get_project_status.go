package application

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockGetProjectStatusUseCase is a mock implementation of GetProjectStatusUseCase for testing
type MockGetProjectStatusUseCase struct {
	mock.Mock
}

// Execute mocks the Execute method
func (m *MockGetProjectStatusUseCase) Execute(ctx context.Context, input GetProjectStatusInput) (*ProjectStatusOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProjectStatusOutput), args.Error(1)
}

// MockRefreshProjectStatusUseCase is a mock implementation of RefreshProjectStatusUseCase for testing
type MockRefreshProjectStatusUseCase struct {
	mock.Mock
}

// Execute mocks the Execute method
func (m *MockRefreshProjectStatusUseCase) Execute(ctx context.Context, input RefreshProjectStatusInput) (*ProjectStatusOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProjectStatusOutput), args.Error(1)
}
