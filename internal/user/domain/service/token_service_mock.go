package service

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
)

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) CreateEmailVerificationToken(ctx context.Context, userID uint) (*token.VerificationToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*token.VerificationToken), args.Error(1)
}

func (m *MockTokenService) CreatePasswordResetToken(ctx context.Context, userID uint) (*token.VerificationToken, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*token.VerificationToken), args.Error(1)
}

func (m *MockTokenService) ValidateAndUseToken(ctx context.Context, tokenStr string, expectedType token.TokenType) (*token.VerificationToken, error) {
	args := m.Called(ctx, tokenStr, expectedType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*token.VerificationToken), args.Error(1)
}

func (m *MockTokenService) CanResendVerificationEmail(ctx context.Context, userID uint) (bool, time.Duration, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Get(1).(time.Duration), args.Error(2)
}

func (m *MockTokenService) InvalidateUserTokens(ctx context.Context, userID uint, tokenType token.TokenType) error {
	args := m.Called(ctx, userID, tokenType)
	return args.Error(0)
}
