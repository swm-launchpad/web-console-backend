package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/mocks"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
)

func TestLoginUserUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 인증정보로 로그인", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "testuser",
			Password: "Password123!",
		}

		email := "test@example.com"
		name := "Test User"
		hashedPassword := "$2a$10$hashedpassword"
		now := time.Now()
		user := &model.User{
			UserID:       1,
			Username:     input.Username,
			PasswordHash: hashedPassword,
			Email:        email,
			Name:         &name,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		token := "jwt.token.here"

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, input.Password).Return(nil)
		mockAuthService.On("GenerateToken", ctx, uint(1)).Return(token, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(1), output.UserID)
		assert.Equal(t, token, output.Token)
		assert.Equal(t, input.Username, output.Username)
		assert.Equal(t, email, output.Email)
		assert.Equal(t, name, output.Name)

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("성공: 선택적 필드가 없는 사용자 로그인", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "minimaluser",
			Password: "Password123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		now := time.Now()
		user := &model.User{
			UserID:       2,
			Username:     input.Username,
			PasswordHash: hashedPassword,
			Email:        "minimal@example.com",
			Name:         nil, // No name
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		token := "jwt.token.here"

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, input.Password).Return(nil)
		mockAuthService.On("GenerateToken", ctx, uint(2)).Return(token, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, uint(2), output.UserID)
		assert.Equal(t, token, output.Token)
		assert.Equal(t, input.Username, output.Username)
		assert.Equal(t, "minimal@example.com", output.Email)
		assert.Empty(t, output.Name)

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: username이 빈 문자열", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "",
			Password: "Password123!",
		}

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "username is required")

		mockRepo.AssertNotCalled(t, "FindByUsername")
		mockAuthService.AssertNotCalled(t, "VerifyPassword")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: password가 빈 문자열", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "testuser",
			Password: "",
		}

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "password is required")

		mockRepo.AssertNotCalled(t, "FindByUsername")
		mockAuthService.AssertNotCalled(t, "VerifyPassword")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "nonexistent",
			Password: "Password123!",
		}

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return((*model.User)(nil), repository.ErrUserNotFound)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "invalid credentials")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "VerifyPassword")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 잘못된 비밀번호", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "testuser",
			Password: "WrongPassword123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		now := time.Now()
		user := &model.User{
			UserID:       1,
			Username:     input.Username,
			PasswordHash: hashedPassword,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, input.Password).Return(errors.New("password does not match"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "invalid credentials")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 비활성 사용자 (Inactive)", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "inactiveuser",
			Password: "Password123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		now := time.Now()
		user := &model.User{
			UserID:       1,
			Username:     input.Username,
			PasswordHash: hashedPassword,
			Status:       model.UserStatusInactive,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, input.Password).Return(nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "user is not active")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 정지된 사용자 (Suspended)", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "suspendeduser",
			Password: "Password123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		now := time.Now()
		user := &model.User{
			UserID:       1,
			Username:     input.Username,
			PasswordHash: hashedPassword,
			Status:       model.UserStatusSuspended,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, input.Password).Return(nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "user is not active")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 대기 중인 사용자 (Pending)", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "pendinguser",
			Password: "Password123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		now := time.Now()
		user := &model.User{
			UserID:       1,
			Username:     input.Username,
			PasswordHash: hashedPassword,
			Status:       model.UserStatusPending,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, input.Password).Return(nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "user is not active")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 삭제된 사용자", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "deleteduser",
			Password: "Password123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		now := time.Now()
		deletedAt := now.Add(-24 * time.Hour)
		user := &model.User{
			UserID:       1,
			Username:     input.Username,
			PasswordHash: hashedPassword,
			Status:       model.UserStatusActive, // Active but deleted
			IsDeleted:    true,
			DeletedAt:    &deletedAt,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, input.Password).Return(nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "user is not active")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})

	t.Run("실패: 토큰 생성 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "testuser",
			Password: "Password123!",
		}

		hashedPassword := "$2a$10$hashedpassword"
		now := time.Now()
		user := &model.User{
			UserID:       1,
			Username:     input.Username,
			PasswordHash: hashedPassword,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return(user, nil)
		mockAuthService.On("VerifyPassword", hashedPassword, input.Password).Return(nil)
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

	t.Run("실패: 데이터베이스 연결 오류", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "testuser",
			Password: "Password123!",
		}

		// Set expectations
		mockRepo.On("FindByUsername", ctx, input.Username).Return((*model.User)(nil), errors.New("database connection error"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "database connection error")

		mockRepo.AssertExpectations(t)
		mockAuthService.AssertNotCalled(t, "VerifyPassword")
		mockAuthService.AssertNotCalled(t, "GenerateToken")
	})
}

func TestLoginUserUseCase_SecurityConsiderations(t *testing.T) {
	ctx := context.Background()

	t.Run("타이밍 공격 방지: 사용자 없음과 잘못된 비밀번호 동일한 에러", func(t *testing.T) {
		// Arrange
		mockRepo1 := new(mocks.MockUserRepository)
		mockAuthService1 := new(mocks.MockAuthService)

		mockRepo2 := new(mocks.MockUserRepository)
		mockAuthService2 := new(mocks.MockAuthService)

		uc1 := NewLoginUserUseCase(mockRepo1, mockAuthService1)
		uc2 := NewLoginUserUseCase(mockRepo2, mockAuthService2)

		input1 := LoginUserInput{
			Username: "nonexistent",
			Password: "Password123!",
		}

		input2 := LoginUserInput{
			Username: "existinguser",
			Password: "WrongPassword123!",
		}

		// Set expectations for non-existent user
		mockRepo1.On("FindByUsername", ctx, input1.Username).Return((*model.User)(nil), repository.ErrUserNotFound)

		// Set expectations for wrong password
		hashedPassword := "$2a$10$hashedpassword"
		now := time.Now()
		user := &model.User{
			UserID:       1,
			Username:     input2.Username,
			PasswordHash: hashedPassword,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}
		mockRepo2.On("FindByUsername", ctx, input2.Username).Return(user, nil)
		mockAuthService2.On("VerifyPassword", hashedPassword, input2.Password).Return(errors.New("password does not match"))

		// Act
		_, err1 := uc1.Execute(ctx, input1)
		_, err2 := uc2.Execute(ctx, input2)

		// Assert - 두 에러 메시지가 동일해야 함
		assert.Error(t, err1)
		assert.Error(t, err2)
		assert.Equal(t, err1.Error(), err2.Error())
		assert.Equal(t, "invalid credentials", err1.Error())

		mockRepo1.AssertExpectations(t)
		mockRepo2.AssertExpectations(t)
		mockAuthService2.AssertExpectations(t)
	})

	t.Run("대소문자 구분 로그인", func(t *testing.T) {
		// Arrange
		mockRepo := new(mocks.MockUserRepository)
		mockAuthService := new(mocks.MockAuthService)

		uc := NewLoginUserUseCase(mockRepo, mockAuthService)

		input := LoginUserInput{
			Username: "TestUser", // Different case
			Password: "Password123!",
		}

		// Set expectations - username은 정확히 일치해야 함
		mockRepo.On("FindByUsername", ctx, "TestUser").Return((*model.User)(nil), repository.ErrUserNotFound)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "invalid credentials")

		mockRepo.AssertExpectations(t)
	})
}