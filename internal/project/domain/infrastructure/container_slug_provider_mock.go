package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockContainerSlugProvider is a mock implementation of ContainerSlugProvider
type MockContainerSlugProvider struct {
	mock.Mock
}

// GetContainerSlugsByProjectID mocks the GetContainerSlugsByProjectID method
func (m *MockContainerSlugProvider) GetContainerSlugsByProjectID(ctx context.Context, projectID uint) ([]string, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// Ensure MockContainerSlugProvider implements ContainerSlugProvider
var _ ContainerSlugProvider = (*MockContainerSlugProvider)(nil)
