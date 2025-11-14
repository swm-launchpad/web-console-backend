package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

func TestLoginUserUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 자격증명으로 로그인", func(t *testing.T) {
		// Arrange
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		uc := NewLoginUserUseCase(mockAuthService, testLogger)

		input := LoginUserInput{
			Email:    "testuser",
			Password: "password123",
		}

		nickname := "testnick"
		expectedUser := &model.User{
			UserID:   1,
			Email:    "test@example.com",
			Nickname: nickname,
			Status:   model.UserStatusActive,
		}
		expectedToken := "jwt_token"

		mockAuthService.On("AuthenticateUser", ctx, input.Email, input.Password).
			Return(expectedUser, expectedToken, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(1), output.UserID)
		assert.Equal(t, expectedToken, output.Token)
		assert.Equal(t, "test@example.com", output.Email)
		assert.Equal(t, nickname, output.Nickname)

		mockAuthService.AssertExpectations(t)
	})

	t.Run("성공: 다른 사용자 로그인", func(t *testing.T) {
		// Arrange
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		uc := NewLoginUserUseCase(mockAuthService, testLogger)

		input := LoginUserInput{
			Email:    "testuser",
			Password: "password123",
		}

		expectedUser := &model.User{
			UserID:   2,
			Email:    "test@example.com",
			Nickname: "testnick2",
			Status:   model.UserStatusActive,
		}
		expectedToken := "jwt_token"

		mockAuthService.On("AuthenticateUser", ctx, input.Email, input.Password).
			Return(expectedUser, expectedToken, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(2), output.UserID)
		assert.Equal(t, expectedToken, output.Token)
		assert.Equal(t, "test@example.com", output.Email)
		assert.Equal(t, "testnick2", output.Nickname)

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 빈 email", func(t *testing.T) {
		// Arrange
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		uc := NewLoginUserUseCase(mockAuthService, testLogger)

		input := LoginUserInput{
			Email:    "",
			Password: "password123",
		}

		mockAuthService.On("AuthenticateUser", ctx, input.Email, input.Password).
			Return((*model.User)(nil), "", errors.New("email is required"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "email is required")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 빈 password", func(t *testing.T) {
		// Arrange
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		uc := NewLoginUserUseCase(mockAuthService, testLogger)

		input := LoginUserInput{
			Email:    "testuser",
			Password: "",
		}

		mockAuthService.On("AuthenticateUser", ctx, input.Email, input.Password).
			Return((*model.User)(nil), "", errors.New("password is required"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "password is required")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		// Arrange
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		uc := NewLoginUserUseCase(mockAuthService, testLogger)

		input := LoginUserInput{
			Email:    "nonexistent",
			Password: "password123",
		}

		mockAuthService.On("AuthenticateUser", ctx, input.Email, input.Password).
			Return((*model.User)(nil), "", errors.New("invalid credentials"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "invalid credentials")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 비밀번호", func(t *testing.T) {
		// Arrange
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		uc := NewLoginUserUseCase(mockAuthService, testLogger)

		input := LoginUserInput{
			Email:    "testuser",
			Password: "wrongpassword",
		}

		mockAuthService.On("AuthenticateUser", ctx, input.Email, input.Password).
			Return((*model.User)(nil), "", errors.New("invalid credentials"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "invalid credentials")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 비활성 사용자", func(t *testing.T) {
		// Arrange
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		uc := NewLoginUserUseCase(mockAuthService, testLogger)

		input := LoginUserInput{
			Email:    "inactiveuser",
			Password: "password123",
		}

		mockAuthService.On("AuthenticateUser", ctx, input.Email, input.Password).
			Return((*model.User)(nil), "", errors.New("user account is not active"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "user account is not active")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 토큰 생성 실패", func(t *testing.T) {
		// Arrange
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		uc := NewLoginUserUseCase(mockAuthService, testLogger)

		input := LoginUserInput{
			Email:    "testuser",
			Password: "password123",
		}

		mockAuthService.On("AuthenticateUser", ctx, input.Email, input.Password).
			Return((*model.User)(nil), "", errors.New("token generation failed"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "token generation failed")

		mockAuthService.AssertExpectations(t)
	})

	t.Run("성공: 다양한 상태의 사용자", func(t *testing.T) {
		// Arrange
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		uc := NewLoginUserUseCase(mockAuthService, testLogger)

		input := LoginUserInput{
			Email:    "testuser",
			Password: "password123",
		}

		// Different user with various status
		org := "Test Org"
		expectedUser := &model.User{
			UserID:       3,
			Email:        "test@example.com",
			Nickname:     "testnick3",
			Organization: &org,
			Status:       model.UserStatusActive,
		}
		expectedToken := "jwt_token"

		mockAuthService.On("AuthenticateUser", ctx, input.Email, input.Password).
			Return(expectedUser, expectedToken, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(3), output.UserID)
		assert.Equal(t, expectedToken, output.Token)
		assert.Equal(t, "test@example.com", output.Email)
		assert.Equal(t, "testnick3", output.Nickname)

		mockAuthService.AssertExpectations(t)
	})
}
