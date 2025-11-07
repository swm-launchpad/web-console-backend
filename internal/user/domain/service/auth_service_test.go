package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
)

func TestAuthService_RegisterUser(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 입력으로 사용자 등록", func(t *testing.T) {
		// Arrange
		mockUserService := new(MockUserService)
		jwtUtil := jwt.NewJWTUtil("test-secret")
		passwordUtil := password.NewPasswordUtil()

		testLogger := logger.NewForTest()
		service := NewAuthService(mockUserService, jwtUtil, passwordUtil, testLogger)

		email := "test@example.com"
		plainPassword := "password123"
		nickname := "testnick"
		userID := uint(1)

		// Mocking with expected password hash pattern
		mockUserService.On("CheckEmailAvailability", ctx, email).Return(nil)
		mockUserService.On("CreateUser", ctx, email, mock.AnythingOfType("string"), nickname).Return(&model.User{
			UserID:       userID,
			Email:        email,
			Nickname:     nickname,
			PasswordHash: "hashed",
			Status:       model.UserStatusActive,
		}, nil)

		// Act
		user, token, err := service.RegisterUser(ctx, email, plainPassword, nickname)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.UserID)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, nickname, user.Nickname)
		assert.NotEmpty(t, token)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: nickname 유효성 검증 실패", func(t *testing.T) {
		// Arrange
		testLogger := logger.NewForTest()
		service := NewAuthService(
			new(MockUserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
			testLogger,
		)

		// Act
		user, token, err := service.RegisterUser(ctx, "test@example.com", "password123", "")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "nickname is required")
	})

	t.Run("실패: 짧은 nickname", func(t *testing.T) {
		// Arrange
		testLogger := logger.NewForTest()
		service := NewAuthService(
			new(MockUserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
			testLogger,
		)

		// Act
		user, token, err := service.RegisterUser(ctx, "test@example.com", "password123", "a")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "nickname must be at least 2 characters long")
	})

	t.Run("실패: 약한 비밀번호", func(t *testing.T) {
		// Arrange
		testLogger := logger.NewForTest()
		service := NewAuthService(
			new(MockUserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
			testLogger,
		)

		// Act
		user, token, err := service.RegisterUser(ctx, "test@example.com", "pass", "testnick")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.True(t, errors.Is(err, ErrWeakPassword))
	})

	t.Run("실패: 잘못된 이메일 형식", func(t *testing.T) {
		// Arrange
		testLogger := logger.NewForTest()
		service := NewAuthService(
			new(MockUserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
			testLogger,
		)

		// Act
		user, token, err := service.RegisterUser(ctx, "invalid-email", "password123", "testnick")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.True(t, errors.Is(err, ErrInvalidEmail))
	})

	t.Run("실패: email 이미 존재", func(t *testing.T) {
		// Arrange
		mockUserService := new(MockUserService)
		testLogger := logger.NewForTest()
		service := NewAuthService(
			mockUserService,
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
			testLogger,
		)

		email := "existing@example.com"
		mockUserService.On("CheckEmailAvailability", ctx, email).Return(errors.New("email already exists"))

		// Act
		user, token, err := service.RegisterUser(ctx, email, "password123", "testnick")

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
		mockUserService := new(MockUserService)
		jwtUtil := jwt.NewJWTUtil("test-secret")
		passwordUtil := password.NewPasswordUtil()

		testLogger := logger.NewForTest()
		service := NewAuthService(mockUserService, jwtUtil, passwordUtil, testLogger)

		email := "test@example.com"
		plainPassword := "password123"
		// Hash the password for test
		passwordHash, _ := passwordUtil.HashPassword(plainPassword)
		userID := uint(1)

		expectedUser := &model.User{
			UserID:       userID,
			Email:        email,
			Nickname:     "testnick",
			PasswordHash: passwordHash,
			Status:       model.UserStatusActive,
		}

		mockUserService.On("GetUserByEmail", ctx, email).Return(expectedUser, nil)
		mockUserService.On("ValidateUserCredentials", ctx, expectedUser).Return(nil)

		// Act
		user, token, err := service.AuthenticateUser(ctx, email, plainPassword)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser, user)
		assert.NotEmpty(t, token)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 빈 email", func(t *testing.T) {
		// Arrange
		testLogger := logger.NewForTest()
		service := NewAuthService(
			new(MockUserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
			testLogger,
		)

		// Act
		user, token, err := service.AuthenticateUser(ctx, "", "password123")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "email is required")
	})

	t.Run("실패: 빈 password", func(t *testing.T) {
		// Arrange
		testLogger := logger.NewForTest()
		service := NewAuthService(
			new(MockUserService),
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
			testLogger,
		)

		// Act
		user, token, err := service.AuthenticateUser(ctx, "test@example.com", "")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Contains(t, err.Error(), "password is required")
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockUserService := new(MockUserService)
		testLogger := logger.NewForTest()
		service := NewAuthService(
			mockUserService,
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
			testLogger,
		)

		email := "nonexistent@example.com"
		mockUserService.On("GetUserByEmail", ctx, email).Return((*model.User)(nil), errors.New("user not found"))

		// Act
		user, token, err := service.AuthenticateUser(ctx, email, "password123")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.True(t, errors.Is(err, ErrInvalidCredentials))

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 비활성 사용자", func(t *testing.T) {
		// Arrange
		mockUserService := new(MockUserService)
		testLogger := logger.NewForTest()
		service := NewAuthService(
			mockUserService,
			jwt.NewJWTUtil("test-secret"),
			password.NewPasswordUtil(),
			testLogger,
		)

		email := "inactive@example.com"
		user := &model.User{
			UserID:   1,
			Email:    email,
			Nickname: "inactive",
			Status:   model.UserStatusInactive,
		}

		mockUserService.On("GetUserByEmail", ctx, email).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(ErrUserNotActive)

		// Act
		resultUser, token, err := service.AuthenticateUser(ctx, email, "password123")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resultUser)
		assert.Empty(t, token)
		assert.True(t, errors.Is(err, ErrUserNotActive))

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 잘못된 비밀번호", func(t *testing.T) {
		// Arrange
		mockUserService := new(MockUserService)
		passwordUtil := password.NewPasswordUtil()
		testLogger := logger.NewForTest()
		service := NewAuthService(
			mockUserService,
			jwt.NewJWTUtil("test-secret"),
			passwordUtil,
			testLogger,
		)

		email := "test@example.com"
		correctPassword := "correct_password"
		wrongPassword := "wrong_password"
		passwordHash, _ := passwordUtil.HashPassword(correctPassword)

		user := &model.User{
			UserID:       1,
			Email:        email,
			Nickname:     "testnick",
			PasswordHash: passwordHash,
			Status:       model.UserStatusActive,
		}

		mockUserService.On("GetUserByEmail", ctx, email).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(nil)

		// Act
		resultUser, token, err := service.AuthenticateUser(ctx, email, wrongPassword)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resultUser)
		assert.Empty(t, token)
		assert.True(t, errors.Is(err, ErrInvalidCredentials))

		mockUserService.AssertExpectations(t)
	})
}

func TestAuthService_ValidateRegistrationInput(t *testing.T) {
	testLogger := logger.NewForTest()
	service := NewAuthService(nil, nil, nil, testLogger)

	t.Run("성공: 유효한 입력", func(t *testing.T) {
		err := service.ValidateRegistrationInput("test@example.com", "password123", "testnick")
		assert.NoError(t, err)
	})

	t.Run("실패: 빈 email", func(t *testing.T) {
		err := service.ValidateRegistrationInput("", "password123", "testnick")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email is required")
	})

	t.Run("실패: 잘못된 email 형식 - @ 없음", func(t *testing.T) {
		err := service.ValidateRegistrationInput("test.example.com", "password123", "testnick")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidEmail))
	})

	t.Run("실패: 잘못된 email 형식 - . 없음", func(t *testing.T) {
		err := service.ValidateRegistrationInput("test@example", "password123", "testnick")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidEmail))
	})

	t.Run("실패: 빈 password", func(t *testing.T) {
		err := service.ValidateRegistrationInput("test@example.com", "", "testnick")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password is required")
	})

	t.Run("실패: 짧은 password", func(t *testing.T) {
		err := service.ValidateRegistrationInput("test@example.com", "pass", "testnick")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrWeakPassword))
	})

	t.Run("실패: 빈 nickname", func(t *testing.T) {
		err := service.ValidateRegistrationInput("test@example.com", "password123", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nickname is required")
	})

	t.Run("실패: 짧은 nickname", func(t *testing.T) {
		err := service.ValidateRegistrationInput("test@example.com", "password123", "a")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nickname must be at least 2 characters long")
	})
}

func TestAuthService_ValidateLoginInput(t *testing.T) {
	testLogger := logger.NewForTest()
	service := NewAuthService(nil, nil, nil, testLogger)

	t.Run("성공: 유효한 입력", func(t *testing.T) {
		err := service.ValidateLoginInput("test@example.com", "password123")
		assert.NoError(t, err)
	})

	t.Run("실패: 빈 email", func(t *testing.T) {
		err := service.ValidateLoginInput("", "password123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email is required")
	})

	t.Run("실패: 빈 password", func(t *testing.T) {
		err := service.ValidateLoginInput("test@example.com", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "password is required")
	})
}

func TestAuthService_GenerateToken(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 userID로 토큰 생성", func(t *testing.T) {
		// Arrange
		testLogger := logger.NewForTest()
		service := NewAuthService(
			nil,
			jwt.NewJWTUtil("test-secret"),
			nil,
			testLogger,
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
		testLogger := logger.NewForTest()
		service := NewAuthService(
			nil,
			jwt.NewJWTUtil("test-secret"),
			nil,
			testLogger,
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
		testLogger := logger.NewForTest()
		service := NewAuthService(
			nil,
			nil,
			password.NewPasswordUtil(),
			testLogger,
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
		testLogger := logger.NewForTest()
		service := NewAuthService(
			nil,
			nil,
			password.NewPasswordUtil(),
			testLogger,
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
		testLogger := logger.NewForTest()
		service := NewAuthService(
			nil,
			nil,
			passwordUtil,
			testLogger,
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
		testLogger := logger.NewForTest()
		service := NewAuthService(
			nil,
			nil,
			password.NewPasswordUtil(),
			testLogger,
		)

		// Act
		err := service.VerifyPassword("", "password123")

		// Assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidCredentials))
	})

	t.Run("실패: 빈 plainPassword", func(t *testing.T) {
		// Arrange
		testLogger := logger.NewForTest()
		service := NewAuthService(
			nil,
			nil,
			password.NewPasswordUtil(),
			testLogger,
		)

		// Act
		err := service.VerifyPassword("hashed_password", "")

		// Assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidCredentials))
	})

	t.Run("실패: 잘못된 비밀번호", func(t *testing.T) {
		// Arrange
		passwordUtil := password.NewPasswordUtil()
		testLogger := logger.NewForTest()
		service := NewAuthService(
			nil,
			nil,
			passwordUtil,
			testLogger,
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
