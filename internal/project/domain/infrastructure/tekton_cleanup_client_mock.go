package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockTektonCleanupClient struct {
	mock.Mock
}

func (m *MockTektonCleanupClient) TriggerCleanup(ctx context.Context, projectID, namespace string, imageNames []string) error {
	args := m.Called(ctx, projectID, namespace, imageNames)
	return args.Error(0)
}
