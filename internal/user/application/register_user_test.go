package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model/token"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

func TestRegisterUserUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	txManager := db.NewStubTxManager()

	t.Run("성공: 유효한 입력으로 사용자 등록", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		uc := NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)

		input := RegisterUserInput{
			Email:    "test@example.com",
			Password: "password123",
			Nickname: "Test User",
		}

		expectedUser := &model.User{
			UserID:   1,
			Email:    input.Email,
			Nickname: input.Nickname,
			Status:   model.UserStatusPending, // Changed to pending
		}
		expectedToken := "jwt_token"
		verificationToken := &token.VerificationToken{
			TokenID:   1,
			UserID:    1,
			Token:     "verification_token_123",
			TokenType: token.TokenTypeEmailVerification,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Email, input.Password, input.Nickname).
			Return(expectedUser, expectedToken, nil)

		mockTokenService.
			On("CreateEmailVerificationToken", mock.Anything, uint(1)).
			Return(verificationToken, nil)

		mockEmailService.
			On("SendVerificationEmail", mock.Anything, input.Email, input.Nickname, verificationToken.Token).
			Return(nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(1), output.UserID)
		assert.Equal(t, expectedToken, output.Token)

		mockAuthService.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

	t.Run("성공: name 없이 사용자 등록", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		uc := NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)

		input := RegisterUserInput{
			Email:    "test@example.com",
			Password: "password123",
			Nickname: "testnick2",
		}

		expectedUser := &model.User{
			UserID:   2,
			Email:    input.Email,
			Nickname: input.Nickname,
			Status:   model.UserStatusPending,
		}
		expectedToken := "jwt_token"
		verificationToken := &token.VerificationToken{
			TokenID:   2,
			UserID:    2,
			Token:     "verification_token_456",
			TokenType: token.TokenTypeEmailVerification,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Email, input.Password, input.Nickname).
			Return(expectedUser, expectedToken, nil)

		mockTokenService.
			On("CreateEmailVerificationToken", mock.Anything, uint(2)).
			Return(verificationToken, nil)

		mockEmailService.
			On("SendVerificationEmail", mock.Anything, input.Email, input.Nickname, verificationToken.Token).
			Return(nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(2), output.UserID)
		assert.Equal(t, expectedToken, output.Token)

		mockAuthService.AssertExpectations(t)
		mockTokenService.AssertExpectations(t)
		mockEmailService.AssertExpectations(t)
	})

	t.Run("실패: 유효성 검증 실패", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		uc := NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)

		input := RegisterUserInput{
			Email:    "",
			Password: "password123",
			Nickname: "testnick",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Email, input.Password, input.Nickname).
			Return((*model.User)(nil), "", errors.New("email is required"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "email is required")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: email 이미 존재", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		uc := NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)

		input := RegisterUserInput{
			Email:    "testuser",
			Password: "password123",
			Nickname: "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Email, input.Password, input.Nickname).
			Return((*model.User)(nil), "", errors.New("email already exists"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "email already exists")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 약한 비밀번호", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		uc := NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)

		input := RegisterUserInput{
			Email:    "testuser",
			Password: "weak",
			Nickname: "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Email, input.Password, input.Nickname).
			Return((*model.User)(nil), "", errors.New("password is too weak"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "password is too weak")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 이메일 형식", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		uc := NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)

		input := RegisterUserInput{
			Email:    "testuser",
			Password: "password123",
			Nickname: "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Email, input.Password, input.Nickname).
			Return((*model.User)(nil), "", errors.New("invalid email format"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "invalid email format")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 토큰 생성 실패", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		mockTokenService := new(service.MockTokenService)
		mockEmailService := new(email.MockService)
		testLogger := logger.NewForTest()
		uc := NewRegisterUserUseCase(mockAuthService, mockTokenService, mockEmailService, txManager, testLogger)

		input := RegisterUserInput{
			Email:    "testuser",
			Password: "password123",
			Nickname: "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Email, input.Password, input.Nickname).
			Return((*model.User)(nil), "", errors.New("token generation failed"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "token generation failed")

		mockAuthService.AssertExpectations(t)
	})
}
