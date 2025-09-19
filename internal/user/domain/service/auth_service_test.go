package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	usermock "github.com/swm-launchpad/web-console-backend/internal/user/mock"
)


func TestAuthService_RegisterUser(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 입력으로 사용자 등록", func(t *testing.T) {
		// Arrange
		mockUserService := new(usermock.UserService)
		jwtUtil := jwt.NewJWTUtil("test-secret")
		passwordUtil := password.NewPasswordUtil()

		service := NewAuthService(mockUserService, jwtUtil, passwordUtil)

		username := "testuser"
		plainPassword := "password123"
		email := "test@example.com"
		name := "Test User"
		userID := uint(1)

		// Mocking with expected password hash pattern
		mockUserService.On("CheckUsernameAvailability", ctx, username).Return(nil)
		mockUserService.On("CheckEmailAvailability", ctx, email).Return(nil)
		mockUserService.On("CreateUser", ctx, username, email, mock.AnythingOfType("string"), &name).Return(&model.User{
			UserID:       userID,
			Username:     username,
			Email:        email,
			Name:         &name,
			PasswordHash: "hashed",
			Status:       model.UserStatusActive,
		}, nil)

		// Act
		user, token, err := service.RegisterUser(ctx, username, plainPassword, email, &name)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.UserID)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, email, user.Email)
		assert.NotEmpty(t, token)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: username 유효성 검증 실패", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			new(usermock.UserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		// Act
		user, token, err := service.RegisterUser(ctx, "", "password123", "test@example.com", nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "username is required")
	})

	t.Run("실패: 짧은 username", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			new(usermock.UserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		// Act
		user, token, err := service.RegisterUser(ctx, "ab", "password123", "test@example.com", nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "username must be at least 3 characters long")
	})

	t.Run("실패: 약한 비밀번호", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			new(usermock.UserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		// Act
		user, token, err := service.RegisterUser(ctx, "testuser", "pass", "test@example.com", nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Equal(t, ErrWeakPassword, err)
	})

	t.Run("실패: 잘못된 이메일 형식", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			new(usermock.UserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		// Act
		user, token, err := service.RegisterUser(ctx, "testuser", "password123", "invalid-email", nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Equal(t, ErrInvalidEmail, err)
	})

	t.Run("실패: username 이미 존재", func(t *testing.T) {
		// Arrange
		mockUserService := new(usermock.UserService)
		service := NewAuthService(
			mockUserService,
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		username := "existinguser"
		mockUserService.On("CheckUsernameAvailability", ctx, username).Return(errors.New("username already exists"))

		// Act
		user, token, err := service.RegisterUser(ctx, username, "password123", "test@example.com", nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "username already exists")

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: email 이미 존재", func(t *testing.T) {
		// Arrange
		mockUserService := new(usermock.UserService)
		service := NewAuthService(
			mockUserService,
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		username := "testuser"
		email := "existing@example.com"

		mockUserService.On("CheckUsernameAvailability", ctx, username).Return(nil)
		mockUserService.On("CheckEmailAvailability", ctx, email).Return(errors.New("email already exists"))

		// Act
		user, token, err := service.RegisterUser(ctx, username, "password123", email, nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "email already exists")

		mockUserService.AssertExpectations(t)
	})
}

func TestAuthService_AuthenticateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 자격증명으로 로그인", func(t *testing.T) {
		// Arrange
		mockUserService := new(usermock.UserService)
		jwtUtil := jwt.NewJWTUtil("test-secret")
		passwordUtil := password.NewPasswordUtil()

		service := NewAuthService(mockUserService, jwtUtil, passwordUtil)

		username := "testuser"
		plainPassword := "password123"
		// Hash the password for test
		passwordHash, _ := passwordUtil.HashPassword(plainPassword)
		userID := uint(1)

		expectedUser := &model.User{
			UserID:       userID,
			Username:     username,
			Email:        "test@example.com",
			PasswordHash: passwordHash,
			Status:       model.UserStatusActive,
		}

		mockUserService.On("GetUserByUsername", ctx, username).Return(expectedUser, nil)
		mockUserService.On("ValidateUserCredentials", ctx, expectedUser).Return(nil)

		// Act
		user, token, err := service.AuthenticateUser(ctx, username, plainPassword)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser, user)
		assert.NotEmpty(t, token)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 빈 username", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			new(usermock.UserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		// Act
		user, token, err := service.AuthenticateUser(ctx, "", "password123")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "username is required")
	})

	t.Run("실패: 빈 password", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			new(usermock.UserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		// Act
		user, token, err := service.AuthenticateUser(ctx, "testuser", "")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "password is required")
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockUserService := new(usermock.UserService)
		service := NewAuthService(
			mockUserService,
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		username := "nonexistent"
		mockUserService.On("GetUserByUsername", ctx, username).Return((*model.User)(nil), errors.New("user not found"))

		// Act
		user, token, err := service.AuthenticateUser(ctx, username, "password123")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Equal(t, ErrInvalidCredentials, err)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 비활성 사용자", func(t *testing.T) {
		// Arrange
		mockUserService := new(usermock.UserService)
		service := NewAuthService(
			mockUserService,
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
		)

		username := "inactiveuser"
		user := &model.User{
			UserID:   1,
			Username: username,
			Status:   model.UserStatusInactive,
		}

		mockUserService.On("GetUserByUsername", ctx, username).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(ErrUserNotActive)

		// Act
		resultUser, token, err := service.AuthenticateUser(ctx, username, "password123")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resultUser)
		assert.Empty(t, token)
		assert.Equal(t, auth.ErrUserNotActive, err)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 비밀번호", func(t *testing.T) {
		// Arrange
		mockUserService := new(usermock.UserService)
		passwordUtil := password.NewPasswordUtil()
		service := NewAuthService(
			mockUserService,
			jwt.NewJWTUtil("test-secret"),
			passwordUtil,
		)

		username := "testuser"
		correctPassword := "correct_password"
		wrongPassword := "wrong_password"
		passwordHash, _ := passwordUtil.HashPassword(correctPassword)

		user := &model.User{
			UserID:       1,
			Username:     username,
			PasswordHash: passwordHash,
			Status:       model.UserStatusActive,
		}

		mockUserService.On("GetUserByUsername", ctx, username).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(nil)

		// Act
		resultUser, token, err := service.AuthenticateUser(ctx, username, wrongPassword)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resultUser)
		assert.Empty(t, token)
		assert.Equal(t, ErrInvalidCredentials, err)

		mockUserService.AssertExpectations(t)
	})
}

func TestAuthService_ValidateRegistrationInput(t *testing.T) {
	service := NewAuthService(nil, nil, nil)

	t.Run("성공: 유효한 입력", func(t *testing.T) {
		err := service.ValidateRegistrationInput("testuser", "password123", "test@example.com")
		assert.NoError(t, err)
	})

	t.Run("실패: 빈 username", func(t *testing.T) {
		err := service.ValidateRegistrationInput("", "password123", "test@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username is required")
	})

	t.Run("실패: 짧은 username", func(t *testing.T) {
		err := service.ValidateRegistrationInput("ab", "password123", "test@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username must be at least 3 characters long")
	})

	t.Run("실패: 빈 password", func(t *testing.T) {
		err := service.ValidateRegistrationInput("testuser", "", "test@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password is required")
	})

	t.Run("실패: 짧은 password", func(t *testing.T) {
		err := service.ValidateRegistrationInput("testuser", "pass", "test@example.com")
		assert.Error(t, err)
		assert.Equal(t, ErrWeakPassword, err)
	})

	t.Run("실패: 빈 email", func(t *testing.T) {
		err := service.ValidateRegistrationInput("testuser", "password123", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email is required")
	})

	t.Run("실패: 잘못된 email 형식 - @ 없음", func(t *testing.T) {
		err := service.ValidateRegistrationInput("testuser", "password123", "test.example.com")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidEmail, err)
	})

	t.Run("실패: 잘못된 email 형식 - . 없음", func(t *testing.T) {
		err := service.ValidateRegistrationInput("testuser", "password123", "test@example")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidEmail, err)
	})
}

func TestAuthService_ValidateLoginInput(t *testing.T) {
	service := NewAuthService(nil, nil, nil)

	t.Run("성공: 유효한 입력", func(t *testing.T) {
		err := service.ValidateLoginInput("testuser", "password123")
		assert.NoError(t, err)
	})

	t.Run("실패: 빈 username", func(t *testing.T) {
		err := service.ValidateLoginInput("", "password123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username is required")
	})

	t.Run("실패: 빈 password", func(t *testing.T) {
		err := service.ValidateLoginInput("testuser", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password is required")
	})
}

func TestAuthService_GenerateToken(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 userID로 토큰 생성", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			nil,
			jwt.NewJWTUtil("test-secret"),
			nil,
		)

		userID := uint(1)

		// Act
		token, err := service.GenerateToken(ctx, userID)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("실패: userID가 0", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			nil,
			jwt.NewJWTUtil("test-secret"),
			nil,
		)

		// Act
		token, err := service.GenerateToken(ctx, 0)

		// Assert
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "invalid user ID")
	})
}

func TestAuthService_HashPassword(t *testing.T) {
	t.Run("성공: 비밀번호 해싱", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			nil,
			nil,
			password.NewPasswordUtil(),
		)

		plainPassword := "password123"

		// Act
		hash, err := service.HashPassword(plainPassword)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, plainPassword, hash)
	})

	t.Run("실패: 빈 비밀번호", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			nil,
			nil,
			password.NewPasswordUtil(),
		)

		// Act
		hash, err := service.HashPassword("")

		// Assert
		assert.Error(t, err)
		assert.Empty(t, hash)
		assert.Contains(t, err.Error(), "password cannot be empty")
	})
}

func TestAuthService_VerifyPassword(t *testing.T) {
	t.Run("성공: 비밀번호 검증", func(t *testing.T) {
		// Arrange
		passwordUtil := password.NewPasswordUtil()
		service := NewAuthService(
			nil,
			nil,
			passwordUtil,
		)

		plainPassword := "password123"
		passwordHash, _ := passwordUtil.HashPassword(plainPassword)

		// Act
		err := service.VerifyPassword(passwordHash, plainPassword)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("실패: 빈 passwordHash", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			nil,
			nil,
			password.NewPasswordUtil(),
		)

		// Act
		err := service.VerifyPassword("", "password123")

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
	})

	t.Run("실패: 빈 plainPassword", func(t *testing.T) {
		// Arrange
		service := NewAuthService(
			nil,
			nil,
			password.NewPasswordUtil(),
		)

		// Act
		err := service.VerifyPassword("hashed_password", "")

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
	})

	t.Run("실패: 잘못된 비밀번호", func(t *testing.T) {
		// Arrange
		passwordUtil := password.NewPasswordUtil()
		service := NewAuthService(
			nil,
			nil,
			passwordUtil,
		)

		correctPassword := "correct_password"
		wrongPassword := "wrong_password"
		passwordHash, _ := passwordUtil.HashPassword(correctPassword)

		// Act
		err := service.VerifyPassword(passwordHash, wrongPassword)

		// Assert
		assert.Error(t, err)
	})
}
