package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockTektonCleanupClient struct {
	mock.Mock
}

func (m *MockTektonCleanupClient) TriggerCleanup(ctx context.Context, projectID, namespace string) error {
	args := m.Called(ctx, projectID, namespace)
	return args.Error(0)
}
