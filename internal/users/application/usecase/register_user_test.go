package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/mocks"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
)

func TestRegisterUserUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 정보로 사용자 등록", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "Password123!",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		hashedPassword := "$2a$10$hashedpassword"
		token := "jwt.token.here"

		// Set expectations
		mockRepo.On("ExistsByUsername", ctx, input.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil)
		mockAuthService.On("HashPassword", input.Password).Return(hashedPassword, nil)
		mockRepo.On("Create", ctx, mock.MatchedBy(func(user *model.User) bool {
			return user.Username == input.Username &&
				user.PasswordHash == hashedPassword &&
				user.Email != nil && *user.Email == input.Email &&
				user.Name != nil && *user.Name == input.Name &&
				user.Status == model.UserStatusActive
		})).Return(nil).Run(func(args mock.Arguments) {
			// Simulate auto-increment ID assignment
			user := args.Get(1).(*model.User)
			user.UserID = 1
		})
		mockAuthService.On("GenerateToken", ctx, uint(1)).Return(token, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(1), output.UserID)
		assert.Equal(t, token, output.Token)

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("성공: 선택적 필드 없이 사용자 등록", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "testuser2",
			Password: "Password456!",
			Email:    "test2@example.com",
			Name:     "", // Optional field
		}

		hashedPassword := "$2a$10$hashedpassword2"
		token := "jwt.token.here2"

		// Set expectations
		mockRepo.On("ExistsByUsername", ctx, input.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil)
		mockAuthService.On("HashPassword", input.Password).Return(hashedPassword, nil)
		mockRepo.On("Create", ctx, mock.MatchedBy(func(user *model.User) bool {
			return user.Username == input.Username &&
				user.PasswordHash == hashedPassword &&
				user.Email != nil && *user.Email == input.Email &&
				(user.Name == nil || *user.Name == "") &&
				user.Status == model.UserStatusActive
		})).Return(nil).Run(func(args mock.Arguments) {
			user := args.Get(1).(*model.User)
			user.UserID = 2
		})
		mockAuthService.On("GenerateToken", ctx, uint(2)).Return(token, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(2), output.UserID)
		assert.Equal(t, token, output.Token)

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: username이 빈 문자열", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "",
			Password: "Password123!",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "username is required")

		mockRepo.AssertNotCalled(t, "ExistsByUsername")
		mockRepo.AssertNotCalled(t, "ExistsByEmail")
		mockAuthService.AssertNotCalled(t, "HashPassword")
		mockRepo.AssertNotCalled(t, "Create")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: password가 빈 문자열", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "password is required")

		mockRepo.AssertNotCalled(t, "Create")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: email이 빈 문자열", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "Password123!",
			Email:    "",
			Name:     "Test User",
		}

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "email is required")

		mockRepo.AssertNotCalled(t, "Create")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: username이 이미 존재", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "existinguser",
			Password: "Password123!",
			Email:    "new@example.com",
			Name:     "Test User",
		}

		// Set expectations
		mockRepo.On("ExistsByUsername", ctx, input.Username).Return(true, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "username already exists")

		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "ExistsByEmail")
		mockAuthService.AssertNotCalled(t, "HashPassword")
		mockRepo.AssertNotCalled(t, "Create")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: email이 이미 존재", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "newuser",
			Password: "Password123!",
			Email:    "existing@example.com",
			Name:     "Test User",
		}

		// Set expectations
		mockRepo.On("ExistsByUsername", ctx, input.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", ctx, input.Email).Return(true, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "email already exists")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "HashPassword")
		mockRepo.AssertNotCalled(t, "Create")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 약한 비밀번호", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "weak",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		// Set expectations - password validation happens first, before checking username/email
		mockAuthService.On("HashPassword", input.Password).Return("", errors.New("password is too weak"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "password is too weak")

		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "Create")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 데이터베이스 생성 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "Password123!",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		hashedPassword := "$2a$10$hashedpassword"

		// Set expectations
		mockRepo.On("ExistsByUsername", ctx, input.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil)
		mockAuthService.On("HashPassword", input.Password).Return(hashedPassword, nil)
		mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "database error")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 토큰 생성 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "Password123!",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		hashedPassword := "$2a$10$hashedpassword"

		// Set expectations
		mockRepo.On("ExistsByUsername", ctx, input.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil)
		mockAuthService.On("HashPassword", input.Password).Return(hashedPassword, nil)
		mockRepo.On("Create", ctx, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			user := args.Get(1).(*model.User)
			user.UserID = 1
		})
		mockAuthService.On("GenerateToken", ctx, uint(1)).Return("", errors.New("token generation failed"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "failed to generate token")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: username 중복 체크 중 데이터베이스 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "Password123!",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		// Set expectations
		mockRepo.On("ExistsByUsername", ctx, input.Username).Return(false, errors.New("database connection error"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "database connection error")

		mockRepo.AssertExpectations(t)
		mockRepo.AssertNotCalled(t, "ExistsByEmail")
		mockAuthService.AssertNotCalled(t, "HashPassword")
		mockRepo.AssertNotCalled(t, "Create")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: email 중복 체크 중 데이터베이스 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "testuser",
			Password: "Password123!",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		// Set expectations
		mockRepo.On("ExistsByUsername", ctx, input.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", ctx, input.Email).Return(false, errors.New("database connection error"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "database connection error")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "HashPassword")
		mockRepo.AssertNotCalled(t, "Create")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})
}

func TestRegisterUserUseCase_ValidatePasswordStrength(t *testing.T) {
	t.Run("유효한 비밀번호 패턴", func(t *testing.T) {
		validPasswords := []string{
			"Password123!",
			"MyP@ssw0rd",
			"Str0ng!Pass",
			"C0mpl3x#Password",
			"V3ry$ecure",
			"12345678", // 실제 구현은 8자 이상만 체크
		}

		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)
		ctx := context.Background()

		for _, password := range validPasswords {
			input := RegisterUserInput{
				Username: "testuser",
				Password: password,
				Email:    "test@example.com",
				Name:     "Test User",
			}

			mockRepo.On("ExistsByUsername", ctx, input.Username).Return(false, nil).Once()
			mockRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil).Once()
			mockAuthService.On("HashPassword", password).Return("hashed", nil).Once()
			mockRepo.On("Create", ctx, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				user := args.Get(1).(*model.User)
				user.UserID = 1
			}).Once()
			mockAuthService.On("GenerateToken", ctx, uint(1)).Return("token", nil).Once()

			output, err := uc.Execute(ctx, input)

			assert.NoError(t, err, "Password: %s", password)
			assert.NotNil(t, output)
		}

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})
}

func TestRegisterUserUseCase_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("매우 긴 username", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		longUsername := ""
		for i := 0; i < 200; i++ {
			longUsername += "a"
		}

		input := RegisterUserInput{
			Username: longUsername,
			Password: "Password123!",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		hashedPassword := "$2a$10$hashedpassword"
		token := "jwt.token.here"

		// Set expectations
		mockRepo.On("ExistsByUsername", ctx, longUsername).Return(false, nil)
		mockRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil)
		mockAuthService.On("HashPassword", input.Password).Return(hashedPassword, nil)
		mockRepo.On("Create", ctx, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			user := args.Get(1).(*model.User)
			user.UserID = 1
		})
		mockAuthService.On("GenerateToken", ctx, uint(1)).Return(token, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("동시 등록 시나리오", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewRegisterUserUseCase(mockRepo, mockAuthService)

		input := RegisterUserInput{
			Username: "racecondition",
			Password: "Password123!",
			Email:    "race@example.com",
			Name:     "Race User",
		}

		hashedPassword := "$2a$10$hashedpassword"

		// Set expectations - ExistsByUsername returns false but Create fails due to race condition
		mockRepo.On("ExistsByUsername", ctx, input.Username).Return(false, nil)
		mockRepo.On("ExistsByEmail", ctx, input.Email).Return(false, nil)
		mockAuthService.On("HashPassword", input.Password).Return(hashedPassword, nil)
		mockRepo.On("Create", ctx, mock.Anything).Return(repository.ErrUserAlreadyExists)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "already exists")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})
}