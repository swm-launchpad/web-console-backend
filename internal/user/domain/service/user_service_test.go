package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure"
)

func TestUserService_CreateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 사용자 생성", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		username := "testuser"
		email := "test@example.com"
		passwordHash := "hashed_password"
		name := "Test User"

		mockRepo.On("Create", ctx, mock.MatchedBy(func(user *model.User) bool {
			return user.Username == username &&
				user.Email == email &&
				user.PasswordHash == passwordHash &&
				user.Status == model.UserStatusPending &&
				!user.IsDeleted
		})).Return(nil)

		// Act
		user, err := service.CreateUser(ctx, username, email, passwordHash, &name)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, email, user.Email)
		assert.Equal(t, passwordHash, user.PasswordHash)
		assert.Equal(t, name, *user.Name)
		assert.Equal(t, model.UserStatusPending, user.Status)
		assert.False(t, user.IsDeleted)

		mockRepo.AssertExpectations(t)
	})

	t.Run("성공: name 없이 사용자 생성", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		username := "testuser2"
		email := "test2@example.com"
		passwordHash := "hashed_password2"

		mockRepo.On("Create", ctx, mock.MatchedBy(func(user *model.User) bool {
			return user.Username == username &&
				user.Email == email &&
				user.Name == nil
		})).Return(nil)

		// Act
		user, err := service.CreateUser(ctx, username, email, passwordHash, nil)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Nil(t, user.Name)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 빈 username", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		// Act
		user, err := service.CreateUser(ctx, "", "test@example.com", "password", nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "username is required")

		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("실패: 빈 email", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		// Act
		user, err := service.CreateUser(ctx, "testuser", "", "password", nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "email is required")

		mockRepo.AssertNotCalled(t, "Create")
	})

	t.Run("실패: repository 에러", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		username := "testuser"
		email := "test@example.com"
		passwordHash := "hashed_password"

		mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))

		// Act
		user, err := service.CreateUser(ctx, username, email, passwordHash, nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "database error")

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 이미 존재하는 사용자", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		username := "existing"
		email := "existing@example.com"
		passwordHash := "hashed_password"

		mockRepo.On("Create", ctx, mock.Anything).Return(usererrors.ErrUserAlreadyExists)

		// Act
		user, err := service.CreateUser(ctx, username, email, passwordHash, nil)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, usererrors.ErrUserAlreadyExists, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_GetUserByID(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 ID로 사용자 조회", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		userID := uint(1)
		expectedUser := &model.User{
			UserID:   userID,
			Username: "testuser",
			Email:    "test@example.com",
		}

		mockRepo.On("FindByID", ctx, userID).Return(expectedUser, nil)

		// Act
		user, err := service.GetUserByID(ctx, userID)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser, user)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: userID가 0", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		// Act
		user, err := service.GetUserByID(ctx, 0)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrInvalidUserData, err)

		mockRepo.AssertNotCalled(t, "FindByID")
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		userID := uint(999)

		mockRepo.On("FindByID", ctx, userID).Return((*model.User)(nil), usererrors.ErrUserNotFound)

		// Act
		user, err := service.GetUserByID(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, usererrors.ErrUserNotFound, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_GetUserByUsername(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 username으로 사용자 조회", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		username := "testuser"
		expectedUser := &model.User{
			UserID:   1,
			Username: username,
			Email:    "test@example.com",
		}

		mockRepo.On("FindByUsername", ctx, username).Return(expectedUser, nil)

		// Act
		user, err := service.GetUserByUsername(ctx, username)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser, user)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 빈 username", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		// Act
		user, err := service.GetUserByUsername(ctx, "")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrInvalidUserData, err)

		mockRepo.AssertNotCalled(t, "FindByUsername")
	})
}

func TestUserService_GetUserByEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 email로 사용자 조회", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		email := "test@example.com"
		expectedUser := &model.User{
			UserID:   1,
			Username: "testuser",
			Email:    email,
		}

		mockRepo.On("FindByEmail", ctx, email).Return(expectedUser, nil)

		// Act
		user, err := service.GetUserByEmail(ctx, email)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser, user)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 빈 email", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		// Act
		user, err := service.GetUserByEmail(ctx, "")

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, ErrInvalidUserData, err)

		mockRepo.AssertNotCalled(t, "FindByEmail")
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 유효한 사용자 업데이트", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		user := &model.User{
			UserID:   1,
			Username: "testuser",
			Email:    "test@example.com",
		}

		mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
			return u.UserID == user.UserID && u.UpdatedAt != nil
		})).Return(nil)

		// Act
		err := service.UpdateUser(ctx, user)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user.UpdatedAt)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: nil user", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		// Act
		err := service.UpdateUser(ctx, nil)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidUserData, err)

		mockRepo.AssertNotCalled(t, "Update")
	})

	t.Run("실패: userID가 0", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		user := &model.User{
			UserID:   0,
			Username: "testuser",
			Email:    "test@example.com",
		}

		// Act
		err := service.UpdateUser(ctx, user)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidUserData, err)

		mockRepo.AssertNotCalled(t, "Update")
	})
}

func TestUserService_ActivateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 사용자 활성화", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		userID := uint(1)
		now := time.Now()
		user := &model.User{
			UserID:    userID,
			Username:  "testuser",
			Email:     "test@example.com",
			Status:    model.UserStatusPending,
			IsDeleted: false,
			CreatedAt: now,
		}

		mockRepo.On("FindByID", ctx, userID).Return(user, nil)
		mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
			return u.Status == model.UserStatusActive
		})).Return(nil)

		// Act
		err := service.ActivateUser(ctx, userID)

		// Assert
		require.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		userID := uint(999)

		mockRepo.On("FindByID", ctx, userID).Return((*model.User)(nil), usererrors.ErrUserNotFound)

		// Act
		err := service.ActivateUser(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, usererrors.ErrUserNotFound, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 삭제된 사용자", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		userID := uint(1)
		user := &model.User{
			UserID:    userID,
			Username:  "testuser",
			IsDeleted: true,
		}

		mockRepo.On("FindByID", ctx, userID).Return(user, nil)

		// Act
		err := service.ActivateUser(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot activate deleted user")

		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_ValidateUserCredentials(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 활성 사용자", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		user := &model.User{
			UserID:    1,
			Username:  "testuser",
			Status:    model.UserStatusActive,
			IsDeleted: false,
		}

		// Act
		err := service.ValidateUserCredentials(ctx, user)

		// Assert
		require.NoError(t, err)
	})

	t.Run("실패: nil user", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		// Act
		err := service.ValidateUserCredentials(ctx, nil)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidUserData, err)
	})

	t.Run("실패: 비활성 사용자", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		user := &model.User{
			UserID:    1,
			Username:  "testuser",
			Status:    model.UserStatusInactive,
			IsDeleted: false,
		}

		// Act
		err := service.ValidateUserCredentials(ctx, user)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotActive, err)
	})

	t.Run("실패: 삭제된 사용자", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		user := &model.User{
			UserID:    1,
			Username:  "testuser",
			Status:    model.UserStatusActive,
			IsDeleted: true,
		}

		// Act
		err := service.ValidateUserCredentials(ctx, user)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotActive, err)
	})
}

func TestUserService_UpdatePassword(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 비밀번호 업데이트", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		userID := uint(1)
		newPasswordHash := "new_hashed_password"
		user := &model.User{
			UserID:       userID,
			Username:     "testuser",
			PasswordHash: "old_hashed_password",
		}

		mockRepo.On("FindByID", ctx, userID).Return(user, nil)
		mockRepo.On("Update", ctx, mock.MatchedBy(func(u *model.User) bool {
			return u.PasswordHash == newPasswordHash && u.PasswordUpdatedAt != nil
		})).Return(nil)

		// Act
		err := service.UpdatePassword(ctx, userID, newPasswordHash)

		// Assert
		require.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 사용자를 찾을 수 없음", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		userID := uint(999)

		mockRepo.On("FindByID", ctx, userID).Return((*model.User)(nil), usererrors.ErrUserNotFound)

		// Act
		err := service.UpdatePassword(ctx, userID, "new_password")

		// Assert
		assert.Error(t, err)
		assert.Equal(t, usererrors.ErrUserNotFound, err)

		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_CheckUsernameAvailability(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: username 사용 가능", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		username := "newuser"

		mockRepo.On("ExistsByUsername", ctx, username).Return(false, nil)

		// Act
		err := service.CheckUsernameAvailability(ctx, username)

		// Assert
		require.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 빈 username", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		// Act
		err := service.CheckUsernameAvailability(ctx, "")

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidUserData, err)

		mockRepo.AssertNotCalled(t, "ExistsByUsername")
	})

	t.Run("실패: username 이미 존재", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		username := "existinguser"

		mockRepo.On("ExistsByUsername", ctx, username).Return(true, nil)

		// Act
		err := service.CheckUsernameAvailability(ctx, username)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username already exists")

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: repository 에러", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		username := "testuser"

		mockRepo.On("ExistsByUsername", ctx, username).Return(false, errors.New("database error"))

		// Act
		err := service.CheckUsernameAvailability(ctx, username)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		mockRepo.AssertExpectations(t)
	})
}

func TestUserService_CheckEmailAvailability(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: email 사용 가능", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		email := "new@example.com"

		mockRepo.On("ExistsByEmail", ctx, email).Return(false, nil)

		// Act
		err := service.CheckEmailAvailability(ctx, email)

		// Assert
		require.NoError(t, err)

		mockRepo.AssertExpectations(t)
	})

	t.Run("실패: 빈 email", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		// Act
		err := service.CheckEmailAvailability(ctx, "")

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidUserData, err)

		mockRepo.AssertNotCalled(t, "ExistsByEmail")
	})

	t.Run("실패: email 이미 존재", func(t *testing.T) {
		// Arrange
		mockRepo := new(infrastructure.MockUserRepository)
		service := NewUserService(mockRepo)

		email := "existing@example.com"

		mockRepo.On("ExistsByEmail", ctx, email).Return(true, nil)

		// Act
		err := service.CheckEmailAvailability(ctx, email)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email already exists")

		mockRepo.AssertExpectations(t)
	})
}
