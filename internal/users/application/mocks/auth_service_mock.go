package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Deprecated: Use MockJWTUtil and MockPasswordUtil instead
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) GenerateToken(ctx context.Context, userID uint) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (uint, error) {
	args := m.Called(ctx, token)
	return uint(args.Int(0)), args.Error(1)
}

func (m *MockAuthService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) VerifyPassword(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}