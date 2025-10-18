package infrastructure

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
)

// MockOAuthStateRepository is a mock implementation of OAuthStateRepository for testing
type MockOAuthStateRepository struct {
	mock.Mock
}

func (m *MockOAuthStateRepository) Create(ctx context.Context, state *model.OAuthState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

func (m *MockOAuthStateRepository) FindByState(ctx context.Context, state string) (*model.OAuthState, error) {
	args := m.Called(ctx, state)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.OAuthState), args.Error(1)
}

func (m *MockOAuthStateRepository) MarkAsConsumed(ctx context.Context, state string, installationID int64) error {
	args := m.Called(ctx, state, installationID)
	return args.Error(0)
}

func (m *MockOAuthStateRepository) DeleteExpired(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
