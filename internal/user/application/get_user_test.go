package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

func TestGetUserUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 사용자 조회", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		userID := uint(1)
		email := "test@example.com"
		name := "Test User"
		phone := "010-1234-5678"
		organization := "Test Company"
		now := time.Now()

		user := &model.User{
			UserID:       userID,
			Username:     "testuser",
			Email:        email,
			Name:         &name,
			Phone:        &phone,
			Organization: &organization,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		input := GetUserInput{
			UserID: userID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, userID, output.UserID)
		assert.Equal(t, "testuser", output.Username)
		assert.Equal(t, email, output.Email)
		assert.Equal(t, name, output.Name)
		assert.Equal(t, phone, output.Phone)
		assert.Equal(t, organization, output.Organization)
		assert.Equal(t, "active", output.Status)
		assert.Equal(t, now, output.CreatedAt)

		mockService.AssertExpectations(t)
	})

	t.Run("성공: 선택적 필드가 없는 사용자 조회", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		userID := uint(2)
		now := time.Now()

		user := &model.User{
			UserID:       userID,
			Username:     "minimaluser",
			Email:        "minimal@example.com",
			Name:         nil,
			Phone:        nil,
			Organization: nil,
			Status:       model.UserStatusPending,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		input := GetUserInput{
			UserID: userID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, userID, output.UserID)
		assert.Equal(t, "minimaluser", output.Username)
		assert.Equal(t, "minimal@example.com", output.Email)
		assert.Empty(t, output.Name)
		assert.Empty(t, output.Phone)
		assert.Empty(t, output.Organization)
		assert.Equal(t, "pending", output.Status)
		assert.Equal(t, now, output.CreatedAt)

		mockService.AssertExpectations(t)
	})

	t.Run("성공: 다양한 상태의 사용자 조회", func(t *testing.T) {
		statuses := []model.UserStatus{
			model.UserStatusActive,
			model.UserStatusInactive,
			model.UserStatusSuspended,
			model.UserStatusPending,
		}

		for _, status := range statuses {
			// Arrange
			mockService := new(service.MockUserService)
			testLogger := logger.NewForTest()
			uc := NewGetUserUseCase(mockService, testLogger)

			userID := uint(3)
			now := time.Now()

			user := &model.User{
				UserID:    userID,
				Username:  "statususer",
				Email:     "status@example.com",
				Status:    status,
				IsDeleted: false,
				CreatedAt: now,
				UpdatedAt: &now,
			}

			input := GetUserInput{
				UserID: userID,
			}

			// Set expectations
			mockService.On("GetUserByID", ctx, userID).Return(user, nil)

			// Act
			output, err := uc.Execute(ctx, input)

			// Assert
			require.NoError(t, err, "Status: %s", status)
			assert.NotNil(t, output)
			assert.Equal(t, string(status), output.Status)

			mockService.AssertExpectations(t)
		}
	})

	t.Run("실패: userID가 0", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		input := GetUserInput{
			UserID: 0,
		}

		// Service.GetUserByID returns error for invalid data
		mockService.On("GetUserByID", ctx, uint(0)).Return((*model.User)(nil), service.ErrInvalidUserData)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, service.ErrInvalidUserData, err)

		mockService.AssertExpectations(t)
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		userID := uint(999)
		input := GetUserInput{
			UserID: userID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return((*model.User)(nil), usererrors.ErrUserNotFound)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Equal(t, usererrors.ErrUserNotFound, err)

		mockService.AssertExpectations(t)
	})

	t.Run("실패: 데이터베이스 오류", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		userID := uint(1)
		input := GetUserInput{
			UserID: userID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return((*model.User)(nil), errors.New("database connection error"))

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "database connection error")

		mockService.AssertExpectations(t)
	})

	t.Run("성공: 삭제된 사용자도 조회 가능", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		userID := uint(4)
		now := time.Now()
		deletedAt := now.Add(-24 * time.Hour)

		user := &model.User{
			UserID:    userID,
			Username:  "deleteduser",
			Email:     "deleted@example.com",
			Status:    model.UserStatusActive,
			IsDeleted: true,
			DeletedAt: &deletedAt,
			CreatedAt: now.Add(-48 * time.Hour),
			UpdatedAt: &now,
		}

		input := GetUserInput{
			UserID: userID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, userID, output.UserID)
		assert.Equal(t, "deleteduser", output.Username)
		assert.Equal(t, "active", output.Status) // 삭제되었어도 상태는 유지

		mockService.AssertExpectations(t)
	})

	t.Run("성공: 모든 선택적 필드가 채워진 사용자", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		userID := uint(5)
		email := "full@example.com"
		name := "Full User"
		phone := "010-9999-8888"
		organization := "Full Company"
		now := time.Now()

		user := &model.User{
			UserID:       userID,
			Username:     "fulluser",
			Email:        email,
			Name:         &name,
			Phone:        &phone,
			Organization: &organization,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    now.Add(-720 * time.Hour), // 30 days ago
			UpdatedAt:    &now,
		}

		input := GetUserInput{
			UserID: userID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, userID, output.UserID)
		assert.Equal(t, "fulluser", output.Username)
		assert.Equal(t, email, output.Email)
		assert.Equal(t, name, output.Name)
		assert.Equal(t, phone, output.Phone)
		assert.Equal(t, organization, output.Organization)
		assert.Equal(t, "active", output.Status)

		mockService.AssertExpectations(t)
	})

	t.Run("성공: 빈 문자열 선택적 필드", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		userID := uint(6)
		emptyStr := ""
		now := time.Now()

		user := &model.User{
			UserID:       userID,
			Username:     "emptyfields",
			Email:        "empty@example.com",
			Name:         &emptyStr, // 빈 문자열
			Phone:        nil,       // nil
			Organization: nil,       // nil
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    now,
			UpdatedAt:    &now,
		}

		input := GetUserInput{
			UserID: userID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, userID, output.UserID)
		assert.Equal(t, "emptyfields", output.Username)
		assert.Equal(t, "empty@example.com", output.Email)
		assert.Empty(t, output.Name) // 빈 문자열
		assert.Empty(t, output.Phone)
		assert.Empty(t, output.Organization)

		mockService.AssertExpectations(t)
	})
}

func TestGetUserUseCase_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("최대 userID 값", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		maxUserID := ^uint(0) // 최대 uint 값
		now := time.Now()

		user := &model.User{
			UserID:    maxUserID,
			Username:  "maxuser",
			Email:     "max@example.com",
			Status:    model.UserStatusActive,
			IsDeleted: false,
			CreatedAt: now,
			UpdatedAt: &now,
		}

		input := GetUserInput{
			UserID: maxUserID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, maxUserID).Return(user, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, maxUserID, output.UserID)

		mockService.AssertExpectations(t)
	})

	t.Run("매우 긴 username 가진 사용자", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		userID := uint(7)
		longUsername := ""
		for i := 0; i < 200; i++ {
			longUsername += "a"
		}
		now := time.Now()

		user := &model.User{
			UserID:    userID,
			Username:  longUsername,
			Email:     "long@example.com",
			Status:    model.UserStatusActive,
			IsDeleted: false,
			CreatedAt: now,
			UpdatedAt: &now,
		}

		input := GetUserInput{
			UserID: userID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, longUsername, output.Username)

		mockService.AssertExpectations(t)
	})

	t.Run("UpdatedAt이 nil인 사용자", func(t *testing.T) {
		// Arrange
		mockService := new(service.MockUserService)
		testLogger := logger.NewForTest()
		uc := NewGetUserUseCase(mockService, testLogger)

		userID := uint(8)
		now := time.Now()

		user := &model.User{
			UserID:    userID,
			Username:  "noupdateuser",
			Email:     "noupdate@example.com",
			Status:    model.UserStatusActive,
			IsDeleted: false,
			CreatedAt: now,
			UpdatedAt: nil, // nil UpdatedAt
		}

		input := GetUserInput{
			UserID: userID,
		}

		// Set expectations
		mockService.On("GetUserByID", ctx, userID).Return(user, nil)

		// Act
		output, err := uc.Execute(ctx, input)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, userID, output.UserID)

		mockService.AssertExpectations(t)
	})
}
