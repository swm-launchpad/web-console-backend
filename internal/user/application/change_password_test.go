package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

func TestChangePasswordUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 비밀번호 변경", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		useCase := NewChangePasswordUseCase(mockUserService, mockAuthService, testLogger)

		userID := uint(1)
		currentPassword := "oldpass123"
		newPassword := "newpass456"
		oldHash := "$2a$10$oldhashedpassword"
		newHash := "$2a$10$newhashedpassword"

		user := &model.User{
			UserID:       userID,
			Nickname:     "testnick",
			Email:        "test@example.com",
			PasswordHash: oldHash,
			Status:       model.UserStatusActive,
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(nil)
		mockAuthService.On("VerifyPassword", oldHash, currentPassword).Return(nil)
		mockAuthService.On("HashPassword", newPassword).Return(newHash, nil)
		mockUserService.On("UpdatePassword", ctx, userID, newHash).Return(nil)

		input := ChangePasswordInput{
			UserID:          userID,
			CurrentPassword: currentPassword,
			NewPassword:     newPassword,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.True(t, output.Success)

		mockUserService.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		useCase := NewChangePasswordUseCase(mockUserService, mockAuthService, testLogger)

		userID := uint(999)

		mockUserService.On("GetUserByID", ctx, userID).Return((*model.User)(nil), usererrors.ErrUserNotFound)

		input := ChangePasswordInput{
			UserID:          userID,
			CurrentPassword: "oldpass123",
			NewPassword:     "newpass456",
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, usererrors.ErrUserNotFound, err)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 비활성 사용자", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		useCase := NewChangePasswordUseCase(mockUserService, mockAuthService, testLogger)

		userID := uint(2)

		user := &model.User{
			UserID:   userID,
			Nickname: "inactive",
			Email:    "inactive@example.com",
			Status:   model.UserStatusInactive,
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(usererrors.ErrUserNotActive)

		input := ChangePasswordInput{
			UserID:          userID,
			CurrentPassword: "oldpass123",
			NewPassword:     "newpass456",
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, usererrors.ErrUserNotActive, err)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 현재 비밀번호가 틀림", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		useCase := NewChangePasswordUseCase(mockUserService, mockAuthService, testLogger)

		userID := uint(3)
		currentPassword := "wrongpassword"
		newPassword := "newpass456"
		oldHash := "$2a$10$oldhashedpassword"

		user := &model.User{
			UserID:       userID,
			Nickname:     "testnick",
			Email:        "test@example.com",
			PasswordHash: oldHash,
			Status:       model.UserStatusActive,
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(nil)
		mockAuthService.On("VerifyPassword", oldHash, currentPassword).Return(errors.New("password mismatch"))

		input := ChangePasswordInput{
			UserID:          userID,
			CurrentPassword: currentPassword,
			NewPassword:     newPassword,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, usererrors.ErrInvalidCredentials, err)

		mockUserService.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 새 비밀번호가 너무 짧음", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		useCase := NewChangePasswordUseCase(mockUserService, mockAuthService, testLogger)

		userID := uint(4)
		currentPassword := "oldpass123"
		newPassword := "short" // 5자 - 8자 미만
		oldHash := "$2a$10$oldhashedpassword"

		user := &model.User{
			UserID:       userID,
			Nickname:     "testnick",
			Email:        "test@example.com",
			PasswordHash: oldHash,
			Status:       model.UserStatusActive,
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(nil)
		mockAuthService.On("VerifyPassword", oldHash, currentPassword).Return(nil)

		input := ChangePasswordInput{
			UserID:          userID,
			CurrentPassword: currentPassword,
			NewPassword:     newPassword,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, usererrors.ErrWeakPassword, err)

		mockUserService.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 비밀번호 해싱 오류", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		useCase := NewChangePasswordUseCase(mockUserService, mockAuthService, testLogger)

		userID := uint(5)
		currentPassword := "oldpass123"
		newPassword := "newpass456"
		oldHash := "$2a$10$oldhashedpassword"

		user := &model.User{
			UserID:       userID,
			Nickname:     "testnick",
			Email:        "test@example.com",
			PasswordHash: oldHash,
			Status:       model.UserStatusActive,
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(nil)
		mockAuthService.On("VerifyPassword", oldHash, currentPassword).Return(nil)
		mockAuthService.On("HashPassword", newPassword).Return("", errors.New("hashing error"))

		input := ChangePasswordInput{
			UserID:          userID,
			CurrentPassword: currentPassword,
			NewPassword:     newPassword,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)

		mockUserService.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})

	t.Run("실패: 비밀번호 업데이트 오류", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		mockAuthService := new(service.MockAuthService)
		testLogger := logger.NewForTest()
		useCase := NewChangePasswordUseCase(mockUserService, mockAuthService, testLogger)

		userID := uint(6)
		currentPassword := "oldpass123"
		newPassword := "newpass456"
		oldHash := "$2a$10$oldhashedpassword"
		newHash := "$2a$10$newhashedpassword"

		user := &model.User{
			UserID:       userID,
			Nickname:     "testnick",
			Email:        "test@example.com",
			PasswordHash: oldHash,
			Status:       model.UserStatusActive,
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(user, nil)
		mockUserService.On("ValidateUserCredentials", ctx, user).Return(nil)
		mockAuthService.On("VerifyPassword", oldHash, currentPassword).Return(nil)
		mockAuthService.On("HashPassword", newPassword).Return(newHash, nil)
		mockUserService.On("UpdatePassword", ctx, userID, newHash).Return(errors.New("database error"))

		input := ChangePasswordInput{
			UserID:          userID,
			CurrentPassword: currentPassword,
			NewPassword:     newPassword,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)

		mockUserService.AssertExpectations(t)
		mockAuthService.AssertExpectations(t)
	})
}
