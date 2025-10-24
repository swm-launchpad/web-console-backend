package email

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) SendVerificationEmail(ctx context.Context, email, username, token string) error {
	args := m.Called(ctx, email, username, token)
	return args.Error(0)
}

func (m *MockService) SendPasswordResetEmail(ctx context.Context, email, username, token string) error {
	args := m.Called(ctx, email, username, token)
	return args.Error(0)
}
