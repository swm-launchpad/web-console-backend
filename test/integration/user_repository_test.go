package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/users/infrastructure/persistence"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

func TestUserRepository_Integration(t *testing.T) {

	// 테스트 DB 설정
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	// Repository 생성
	userRepo := persistence.NewUserRepository(testDB.DB)
	ctx := context.Background()

	t.Run("Create and FindByID", func(t *testing.T) {
		// Given
		name := "Test User"
		phone := "010-1234-5678"
		org := "Test Corp"

		user := &model.User{
			Username:     "testuser",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        "test@example.com",
			Name:         &name,
			Phone:        &phone,
			Organization: &org,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// When - Create
		err := userRepo.Create(ctx, user)

		// Then
		require.NoError(t, err)
		assert.NotZero(t, user.UserID) // AUTO_INCREMENT ID가 할당되어야 함

		// When - FindByID
		foundUser, err := userRepo.FindByID(ctx, user.UserID)

		// Then
		require.NoError(t, err)
		assert.Equal(t, user.UserID, foundUser.UserID)
		assert.Equal(t, user.Username, foundUser.Username)
		assert.Equal(t, user.Email, foundUser.Email)
		assert.Equal(t, *user.Name, *foundUser.Name)
		assert.Equal(t, *user.Phone, *foundUser.Phone)
		assert.Equal(t, *user.Organization, *foundUser.Organization)
		assert.Equal(t, user.Status, foundUser.Status)
	})

	t.Run("FindByUsername", func(t *testing.T) {
		// Given
		user := &model.User{
			Username:     "uniqueuser",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        "username@example.com",
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// Create user first
		err := userRepo.Create(ctx, user)
		require.NoError(t, err)

		// When
		foundUser, err := userRepo.FindByUsername(ctx, "uniqueuser")

		// Then
		require.NoError(t, err)
		assert.Equal(t, user.UserID, foundUser.UserID)
		assert.Equal(t, user.Username, foundUser.Username)
	})

	t.Run("FindByEmail", func(t *testing.T) {
		// Given
		email := "findbyemail@example.com"
		user := &model.User{
			Username:     "emailuser",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        email,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// Create user first
		err := userRepo.Create(ctx, user)
		require.NoError(t, err)

		// When
		foundUser, err := userRepo.FindByEmail(ctx, email)

		// Then
		require.NoError(t, err)
		assert.Equal(t, user.UserID, foundUser.UserID)
		assert.Equal(t, user.Email, foundUser.Email)
	})

	t.Run("Update", func(t *testing.T) {
		// Given
		user := &model.User{
			Username:     "updateuser",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        "update@example.com",
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// Create user first
		err := userRepo.Create(ctx, user)
		require.NoError(t, err)

		// When - Update
		newEmail := "newemail@example.com"
		newName := "Updated Name"
		newPhone := "010-9999-8888"
		now := time.Now()

		user.Email = newEmail
		user.Name = &newName
		user.Phone = &newPhone
		user.Status = model.UserStatusInactive
		user.UpdatedAt = &now

		err = userRepo.Update(ctx, user)

		// Then
		require.NoError(t, err)

		// Verify update
		updatedUser, err := userRepo.FindByID(ctx, user.UserID)
		require.NoError(t, err)
		assert.Equal(t, newEmail, updatedUser.Email)
		assert.Equal(t, newName, *updatedUser.Name)
		assert.Equal(t, newPhone, *updatedUser.Phone)
		assert.Equal(t, model.UserStatusInactive, updatedUser.Status)
	})

	t.Run("Delete (Soft Delete)", func(t *testing.T) {
		// Given
		user := &model.User{
			Username:     "deleteuser",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        "delete@example.com",
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// Create user first
		err := userRepo.Create(ctx, user)
		require.NoError(t, err)

		// When
		err = userRepo.Delete(ctx, user.UserID)

		// Then
		require.NoError(t, err)

		// Verify soft delete - deleted user should not be found
		deletedUser, err := userRepo.FindByID(ctx, user.UserID)
		assert.Equal(t, repository.ErrUserNotFound, err)
		assert.Nil(t, deletedUser)

		// Verify deleted user is not found by username either
		deletedUser, err = userRepo.FindByUsername(ctx, "deleteuser")
		assert.Equal(t, repository.ErrUserNotFound, err)
		assert.Nil(t, deletedUser)
	})

	t.Run("ExistsByUsername", func(t *testing.T) {
		// Given
		user := &model.User{
			Username:     "existsuser",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        "exists@example.com",
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// When - Before creation
		exists, err := userRepo.ExistsByUsername(ctx, "existsuser")
		require.NoError(t, err)
		assert.False(t, exists)

		// Create user
		err = userRepo.Create(ctx, user)
		require.NoError(t, err)

		// When - After creation
		exists, err = userRepo.ExistsByUsername(ctx, "existsuser")

		// Then
		require.NoError(t, err)
		assert.True(t, exists)

		// When - After soft delete
		err = userRepo.Delete(ctx, user.UserID)
		require.NoError(t, err)

		exists, err = userRepo.ExistsByUsername(ctx, "existsuser")
		require.NoError(t, err)
		assert.False(t, exists) // Soft deleted users should not exist
	})

	t.Run("ExistsByEmail", func(t *testing.T) {
		// Given
		email := "existsemail@example.com"
		user := &model.User{
			Username:     "emailexistsuser",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        email,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// When - Before creation
		exists, err := userRepo.ExistsByEmail(ctx, email)
		require.NoError(t, err)
		assert.False(t, exists)

		// Create user
		err = userRepo.Create(ctx, user)
		require.NoError(t, err)

		// When - After creation
		exists, err = userRepo.ExistsByEmail(ctx, email)

		// Then
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Transaction Rollback", func(t *testing.T) {
		// Given
		tx, err := testDB.DB.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback() // Always rollback at the end

		// Create repository with transaction
		txRepo := persistence.NewUserRepository(tx)

		user := &model.User{
			Username:     "txuser",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        "txtest@example.com",
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// When - Create user in transaction
		err = txRepo.Create(ctx, user)
		require.NoError(t, err)

		// Verify user exists in transaction
		foundInTx, err := txRepo.FindByUsername(ctx, "txuser")
		require.NoError(t, err)
		assert.NotNil(t, foundInTx)
		assert.Equal(t, "txuser", foundInTx.Username)

		// Rollback transaction
		err = tx.Rollback()
		require.NoError(t, err)

		// Then - Verify user doesn't exist after rollback
		foundAfterRollback, err := userRepo.FindByUsername(ctx, "txuser")
		assert.Equal(t, repository.ErrUserNotFound, err)
		assert.Nil(t, foundAfterRollback)
	})

	t.Run("Duplicate Username Error", func(t *testing.T) {
		// Given
		user1 := &model.User{
			Username:     "duplicateusername",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        "dup1@example.com",
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// Create first user
		err := userRepo.Create(ctx, user1)
		require.NoError(t, err)

		// Try to create second user with same username
		user2 := &model.User{
			Username:     "duplicateusername",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        "dup2@example.com",
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		// When
		err = userRepo.Create(ctx, user2)

		// Then
		assert.Error(t, err)
		assert.Equal(t, repository.ErrUserAlreadyExists, err)
	})

	t.Run("Update Non-existent User", func(t *testing.T) {
		// Given
		user := &model.User{
			UserID:       999999, // Non-existent ID
			Username:     "nonexistent",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        "nonexistent@example.com",
			Status:       model.UserStatusActive,
		}

		// When
		err := userRepo.Update(ctx, user)

		// Then
		assert.Error(t, err)
		assert.Equal(t, repository.ErrUserNotFound, err)
	})
}
