package service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
)

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) RegisterUser(ctx context.Context, username, plainPassword, email string, name *string) (*model.User, string, error) {
	args := m.Called(ctx, username, plainPassword, email, name)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).(*model.User), args.String(1), args.Error(2)
}

func (m *MockAuthService) AuthenticateUser(ctx context.Context, username, plainPassword string) (*model.User, string, error) {
	args := m.Called(ctx, username, plainPassword)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Error(2)
	}
	return args.Get(0).(*model.User), args.String(1), args.Error(2)
}

func (m *MockAuthService) ValidateRegistrationInput(username, plainPassword, email string) error {
	args := m.Called(username, plainPassword, email)
	return args.Error(0)
}

func (m *MockAuthService) ValidateLoginInput(username, plainPassword string) error {
	args := m.Called(username, plainPassword)
	return args.Error(0)
}

func (m *MockAuthService) GenerateToken(ctx context.Context, userID uint) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) HashPassword(plainPassword string) (string, error) {
	args := m.Called(plainPassword)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) VerifyPassword(passwordHash, plainPassword string) error {
	args := m.Called(passwordHash, plainPassword)
	return args.Error(0)
}
