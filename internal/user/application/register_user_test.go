package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

func TestRegisterUserUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	txManager := db.NewStubTxManager()

	t.Run("성공: 유효한 입력으로 사용자 등록", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		uc := NewRegisterUserUseCase(mockAuthService, txManager)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "password123",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		name := "Test User"
		expectedUser := &model.User{
			UserID:   1,
			Username: input.Username,
			Email:    input.Email,
			Name:     &name,
			Status:   model.UserStatusActive,
		}
		expectedToken := "jwt_token"

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Username, input.Password, input.Email, &name).
			Return(expectedUser, expectedToken, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(1), output.UserID)
		assert.Equal(t, expectedToken, output.Token)

		mockAuthService.AssertExpectations(t)
	})

	t.Run("성공: name 없이 사용자 등록", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		uc := NewRegisterUserUseCase(mockAuthService, txManager)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "password123",
			Email:    "test@example.com",
			Name:     "",
		}

		expectedUser := &model.User{
			UserID:   2,
			Username: input.Username,
			Email:    input.Email,
			Name:     nil,
			Status:   model.UserStatusActive,
		}
		expectedToken := "jwt_token"

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Username, input.Password, input.Email, (*string)(nil)).
			Return(expectedUser, expectedToken, nil)

		output, err := uc.Execute(ctx, input)

		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(2), output.UserID)
		assert.Equal(t, expectedToken, output.Token)

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 유효성 검증 실패", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		uc := NewRegisterUserUseCase(mockAuthService, txManager)

		input := RegisterUserInput{
			Username: "",
			Password: "password123",
			Email:    "test@example.com",
			Name:     "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Username, input.Password, input.Email, (*string)(nil)).
			Return((*model.User)(nil), "", errors.New("username is required"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "username is required")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: username 이미 존재", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		uc := NewRegisterUserUseCase(mockAuthService, txManager)

		input := RegisterUserInput{
			Username: "existinguser",
			Password: "password123",
			Email:    "test@example.com",
			Name:     "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Username, input.Password, input.Email, (*string)(nil)).
			Return((*model.User)(nil), "", errors.New("username already exists"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "username already exists")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: email 이미 존재", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		uc := NewRegisterUserUseCase(mockAuthService, txManager)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "password123",
			Email:    "existing@example.com",
			Name:     "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Username, input.Password, input.Email, (*string)(nil)).
			Return((*model.User)(nil), "", errors.New("email already exists"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "email already exists")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 약한 비밀번호", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		uc := NewRegisterUserUseCase(mockAuthService, txManager)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "weak",
			Email:    "test@example.com",
			Name:     "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Username, input.Password, input.Email, (*string)(nil)).
			Return((*model.User)(nil), "", errors.New("password is too weak"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "password is too weak")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 이메일 형식", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		uc := NewRegisterUserUseCase(mockAuthService, txManager)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "password123",
			Email:    "invalid-email",
			Name:     "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Username, input.Password, input.Email, (*string)(nil)).
			Return((*model.User)(nil), "", errors.New("invalid email format"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "invalid email format")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 토큰 생성 실패", func(t *testing.T) {
		mockAuthService := new(service.MockAuthService)
		uc := NewRegisterUserUseCase(mockAuthService, txManager)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "password123",
			Email:    "test@example.com",
			Name:     "",
		}

		mockAuthService.
			On("RegisterUser", mock.Anything, input.Username, input.Password, input.Email, (*string)(nil)).
			Return((*model.User)(nil), "", errors.New("token generation failed"))

		output, err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "token generation failed")

		mockAuthService.AssertExpectations(t)
	})
}
