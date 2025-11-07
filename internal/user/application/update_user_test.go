package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

func TestUpdateUserUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 모든 필드 업데이트", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		useCase := NewUpdateUserUseCase(mockUserService, testLogger)

		userID := uint(1)
		name := "Updated Name"
		phone := "010-1234-5678"
		organization := "Updated Org"

		existingUser := &model.User{
			UserID:       userID,
			Nickname:     "testuser",
			Email:        "test@example.com",
			Phone:        stringPtr("010-0000-0000"),
			Organization: stringPtr("Old Org"),
			Status:       model.UserStatusActive,
			CreatedAt:    time.Now(),
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(existingUser, nil)
		mockUserService.On("ValidateUserCredentials", ctx, mock.AnythingOfType("*model.User")).Return(nil)
		mockUserService.On("UpdateUser", ctx, mock.AnythingOfType("*model.User")).Return(nil)

		input := UpdateUserInput{
			UserID:       userID,
			Nickname:     &name,
			Phone:        &phone,
			Organization: &organization,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, userID, output.UserID)
		assert.Equal(t, "test@example.com", output.Email)
		assert.Equal(t, name, output.Nickname)
		assert.Equal(t, phone, output.Phone)
		assert.Equal(t, organization, output.Organization)
		assert.Equal(t, string(model.UserStatusActive), output.Status)

		mockUserService.AssertExpectations(t)
	})

	t.Run("성공: 일부 필드만 업데이트", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		useCase := NewUpdateUserUseCase(mockUserService, testLogger)

		userID := uint(2)
		name := "Only Name Updated"

		existingUser := &model.User{
			UserID:       userID,
			Nickname:     "testuser2",
			Email:        "test2@example.com",
			Phone:        nil,
			Organization: nil,
			Status:       model.UserStatusActive,
			CreatedAt:    time.Now(),
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(existingUser, nil)
		mockUserService.On("ValidateUserCredentials", ctx, mock.AnythingOfType("*model.User")).Return(nil)
		mockUserService.On("UpdateUser", ctx, mock.AnythingOfType("*model.User")).Return(nil)

		input := UpdateUserInput{
			UserID:   userID,
			Nickname: &name,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, name, output.Nickname)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		useCase := NewUpdateUserUseCase(mockUserService, testLogger)

		userID := uint(999)
		name := "Test Name"

		mockUserService.On("GetUserByID", ctx, userID).Return((*model.User)(nil), usererrors.ErrUserNotFound)

		input := UpdateUserInput{
			UserID:   userID,
			Nickname: &name,
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
		testLogger := logger.NewForTest()
		useCase := NewUpdateUserUseCase(mockUserService, testLogger)

		userID := uint(3)
		name := "Test Name"

		inactiveUser := &model.User{
			UserID:   userID,
			Nickname: "inactiveuser",
			Email:    "inactive@example.com",
			Status:   model.UserStatusInactive,
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(inactiveUser, nil)
		mockUserService.On("ValidateUserCredentials", ctx, inactiveUser).Return(usererrors.ErrUserNotActive)

		input := UpdateUserInput{
			UserID:   userID,
			Nickname: &name,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, usererrors.ErrUserNotActive, err)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 업데이트할 필드가 없음", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		useCase := NewUpdateUserUseCase(mockUserService, testLogger)

		userID := uint(4)

		existingUser := &model.User{
			UserID:   userID,
			Nickname: "testuser",
			Email:    "test@example.com",
			Status:   model.UserStatusActive,
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(existingUser, nil)
		mockUserService.On("ValidateUserCredentials", ctx, existingUser).Return(nil)

		input := UpdateUserInput{
			UserID: userID,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, usererrors.ErrNoFieldsToUpdate, err)

		mockUserService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 업데이트 오류", func(t *testing.T) {
		// Arrange
		mockUserService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		useCase := NewUpdateUserUseCase(mockUserService, testLogger)

		userID := uint(5)
		name := "Test Name"

		existingUser := &model.User{
			UserID:   userID,
			Nickname: "testuser",
			Email:    "test@example.com",
			Status:   model.UserStatusActive,
		}

		mockUserService.On("GetUserByID", ctx, userID).Return(existingUser, nil)
		mockUserService.On("ValidateUserCredentials", ctx, mock.AnythingOfType("*model.User")).Return(nil)
		mockUserService.On("UpdateUser", ctx, mock.AnythingOfType("*model.User")).Return(errors.New("database error"))

		input := UpdateUserInput{
			UserID:   userID,
			Nickname: &name,
		}

		// Act
		output, err := useCase.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)

		mockUserService.AssertExpectations(t)
	})
}

func stringPtr(s string) *string {
	return &s
}
