package service

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockResourceValidationService is a mock implementation of ResourceValidationService
type MockResourceValidationService struct {
	mock.Mock
}

// ValidateProjectResourceLimits mocks the ValidateProjectResourceLimits method
func (m *MockResourceValidationService) ValidateProjectResourceLimits(
	ctx context.Context,
	projectID uint,
	requestedCPU, requestedMemory uint32,
	excludeContainerID uint,
) error {
	args := m.Called(ctx, projectID, requestedCPU, requestedMemory, excludeContainerID)
	return args.Error(0)
}
