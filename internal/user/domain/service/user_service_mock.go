package service

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
)

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) CreateUser(ctx context.Context, email, passwordHash, nickname string) (*model.User, error) {
	args := m.Called(ctx, email, passwordHash, nickname)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) GetUserByID(ctx context.Context, userID uint) (*model.User, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserService) UpdateUser(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserService) ActivateUser(ctx context.Context, userID uint) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserService) ValidateUserCredentials(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserService) UpdatePassword(ctx context.Context, userID uint, passwordHash string) error {
	args := m.Called(ctx, userID, passwordHash)
	return args.Error(0)
}

func (m *MockUserService) CheckEmailAvailability(ctx context.Context, email string) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}
