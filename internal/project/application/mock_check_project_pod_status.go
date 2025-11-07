package application

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockCheckProjectPodStatusUseCase is a mock implementation of CheckProjectPodStatusUseCase for testing
type MockCheckProjectPodStatusUseCase struct {
	mock.Mock
}

// Execute mocks the Execute method
func (m *MockCheckProjectPodStatusUseCase) Execute(ctx context.Context, input CheckProjectPodStatusInput) (*CheckProjectPodStatusOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CheckProjectPodStatusOutput), args.Error(1)
}
