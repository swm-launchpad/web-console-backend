package persistence

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	return db, mock
}

func TestUserRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 사용자 생성 및 AUTO_INCREMENT ID 할당", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		email := "test@example.com"
		name := "Test User"
		user := &model.User{
			Username:     "testuser",
			PasswordHash: "$2a$10$hashedpassword",
			Email:        &email,
			Name:         &name,
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		query := `
		INSERT INTO USERS (
			username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

		// Set expectations
		mock.ExpectExec(query).
			WithArgs(
				user.Username,
				user.PasswordHash,
				user.PasswordUpdatedAt,
				user.Name,
				user.Email,
				user.Phone,
				user.Organization,
				user.Status,
				user.IsDeleted,
				user.DeletedAt,
				user.CreatedAt,
				user.UpdatedAt,
			).
			WillReturnResult(sqlmock.NewResult(123, 1))

		// Act
		err := repo.Create(ctx, user)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, uint(123), user.UserID) // AUTO_INCREMENT ID가 할당되어야 함
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("실패: 중복된 username", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		user := &model.User{
			Username:     "duplicateuser",
			PasswordHash: "$2a$10$hashedpassword",
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		query := `
		INSERT INTO USERS (
			username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

		// Set expectations - simulate duplicate key error
		mock.ExpectExec(query).
			WithArgs(
				user.Username,
				user.PasswordHash,
				user.PasswordUpdatedAt,
				user.Name,
				user.Email,
				user.Phone,
				user.Organization,
				user.Status,
				user.IsDeleted,
				user.DeletedAt,
				user.CreatedAt,
				user.UpdatedAt,
			).
			WillReturnError(errors.New("Error 1062: Duplicate entry 'duplicateuser' for key 'username'"))

		// Act
		err := repo.Create(ctx, user)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, repository.ErrUserAlreadyExists, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindByID(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: ID로 사용자 조회", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		userID := uint(123)
		email := "test@example.com"
		name := "Test User"
		now := time.Now()

		query := `
		SELECT
			user_id, username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		FROM USERS
		WHERE user_id = ? AND is_deleted = FALSE
	`

		rows := sqlmock.NewRows([]string{
			"user_id", "username", "password_hash", "password_updated_at",
			"name", "email", "phone", "organization",
			"status", "is_deleted", "deleted_at", "created_at", "updated_at",
		}).AddRow(
			userID, "testuser", "$2a$10$hashedpassword", nil,
			name, email, nil, nil,
			"active", false, nil, now, &now,
		)

		mock.ExpectQuery(query).
			WithArgs(userID).
			WillReturnRows(rows)

		// Act
		user, err := repo.FindByID(ctx, userID)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.UserID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, email, *user.Email)
		assert.Equal(t, name, *user.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		userID := uint(999)

		query := `
		SELECT
			user_id, username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		FROM USERS
		WHERE user_id = ? AND is_deleted = FALSE
	`

		mock.ExpectQuery(query).
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		// Act
		user, err := repo.FindByID(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, repository.ErrUserNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindByUsername(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: username으로 사용자 조회", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		username := "testuser"
		userID := uint(123)
		now := time.Now()

		query := `
		SELECT
			user_id, username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		FROM USERS
		WHERE username = ? AND is_deleted = FALSE
	`

		rows := sqlmock.NewRows([]string{
			"user_id", "username", "password_hash", "password_updated_at",
			"name", "email", "phone", "organization",
			"status", "is_deleted", "deleted_at", "created_at", "updated_at",
		}).AddRow(
			userID, username, "$2a$10$hashedpassword", nil,
			nil, nil, nil, nil,
			"active", false, nil, now, &now,
		)

		mock.ExpectQuery(query).
			WithArgs(username).
			WillReturnRows(rows)

		// Act
		user, err := repo.FindByUsername(ctx, username)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.UserID)
		assert.Equal(t, username, user.Username)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("실패: 존재하지 않는 username", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		username := "nonexistent"

		query := `
		SELECT
			user_id, username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		FROM USERS
		WHERE username = ? AND is_deleted = FALSE
	`

		mock.ExpectQuery(query).
			WithArgs(username).
			WillReturnError(sql.ErrNoRows)

		// Act
		user, err := repo.FindByUsername(ctx, username)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, repository.ErrUserNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_FindByEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: email로 사용자 조회", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		email := "test@example.com"
		userID := uint(123)
		now := time.Now()

		query := `
		SELECT
			user_id, username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		FROM USERS
		WHERE email = ? AND is_deleted = FALSE
	`

		rows := sqlmock.NewRows([]string{
			"user_id", "username", "password_hash", "password_updated_at",
			"name", "email", "phone", "organization",
			"status", "is_deleted", "deleted_at", "created_at", "updated_at",
		}).AddRow(
			userID, "testuser", "$2a$10$hashedpassword", nil,
			nil, email, nil, nil,
			"active", false, nil, now, &now,
		)

		mock.ExpectQuery(query).
			WithArgs(email).
			WillReturnRows(rows)

		// Act
		user, err := repo.FindByEmail(ctx, email)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.UserID)
		assert.Equal(t, email, *user.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 사용자 정보 업데이트", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		now := time.Now()
		email := "updated@example.com"
		name := "Updated User"
		phone := "010-1234-5678"
		org := "Updated Org"

		user := &model.User{
			UserID:              123,
			Username:            "testuser",
			PasswordHash:        "$2a$10$newhashedpassword",
			PasswordUpdatedAt:   &now,
			Email:               &email,
			Name:                &name,
			Phone:               &phone,
			Organization:        &org,
			Status:              model.UserStatusActive,
			IsDeleted:           false,
			UpdatedAt:           &now,
		}

		query := `
		UPDATE USERS SET
			password_hash = ?, password_updated_at = ?, name = ?,
			email = ?, phone = ?, organization = ?,
			status = ?, is_deleted = ?, deleted_at = ?, updated_at = ?
		WHERE user_id = ?
	`

		mock.ExpectExec(query).
			WithArgs(
				user.PasswordHash,
				user.PasswordUpdatedAt,
				user.Name,
				user.Email,
				user.Phone,
				user.Organization,
				user.Status,
				user.IsDeleted,
				user.DeletedAt,
				user.UpdatedAt,
				user.UserID,
			).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Act
		err := repo.Update(ctx, user)

		// Assert
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		now := time.Now()
		user := &model.User{
			UserID:       999,
			Username:     "nonexistent",
			PasswordHash: "$2a$10$hashedpassword",
			Status:       model.UserStatusActive,
			UpdatedAt:    &now,
		}

		query := `
		UPDATE USERS SET
			password_hash = ?, password_updated_at = ?, name = ?,
			email = ?, phone = ?, organization = ?,
			status = ?, is_deleted = ?, deleted_at = ?, updated_at = ?
		WHERE user_id = ?
	`

		mock.ExpectExec(query).
			WithArgs(
				user.PasswordHash,
				user.PasswordUpdatedAt,
				user.Name,
				user.Email,
				user.Phone,
				user.Organization,
				user.Status,
				user.IsDeleted,
				user.DeletedAt,
				user.UpdatedAt,
				user.UserID,
			).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Act
		err := repo.Update(ctx, user)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, repository.ErrUserNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: 사용자 삭제 (soft delete)", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		userID := uint(123)

		query := `
		UPDATE USERS SET
			is_deleted = TRUE,
			deleted_at = ?,
			updated_at = ?
		WHERE user_id = ?
	`

		mock.ExpectExec(query).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Act
		err := repo.Delete(ctx, userID)

		// Assert
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("실패: 존재하지 않는 사용자", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		userID := uint(999)

		query := `
		UPDATE USERS SET
			is_deleted = TRUE,
			deleted_at = ?,
			updated_at = ?
		WHERE user_id = ?
	`

		mock.ExpectExec(query).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Act
		err := repo.Delete(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, repository.ErrUserNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_ExistsByUsername(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: username이 존재하는 경우", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		username := "existinguser"

		query := `SELECT EXISTS(SELECT 1 FROM USERS WHERE username = ? AND is_deleted = FALSE)`

		rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)

		mock.ExpectQuery(query).
			WithArgs(username).
			WillReturnRows(rows)

		// Act
		exists, err := repo.ExistsByUsername(ctx, username)

		// Assert
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("성공: username이 존재하지 않는 경우", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		username := "nonexistentuser"

		query := `SELECT EXISTS(SELECT 1 FROM USERS WHERE username = ? AND is_deleted = FALSE)`

		rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)

		mock.ExpectQuery(query).
			WithArgs(username).
			WillReturnRows(rows)

		// Act
		exists, err := repo.ExistsByUsername(ctx, username)

		// Assert
		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("성공: email이 존재하는 경우", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		email := "existing@example.com"

		query := `SELECT EXISTS(SELECT 1 FROM USERS WHERE email = ? AND is_deleted = FALSE)`

		rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)

		mock.ExpectQuery(query).
			WithArgs(email).
			WillReturnRows(rows)

		// Act
		exists, err := repo.ExistsByEmail(ctx, email)

		// Assert
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("성공: email이 존재하지 않는 경우", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		email := "nonexistent@example.com"

		query := `SELECT EXISTS(SELECT 1 FROM USERS WHERE email = ? AND is_deleted = FALSE)`

		rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)

		mock.ExpectQuery(query).
			WithArgs(email).
			WillReturnRows(rows)

		// Act
		exists, err := repo.ExistsByEmail(ctx, email)

		// Assert
		assert.NoError(t, err)
		assert.False(t, exists)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("데이터베이스 연결 오류", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		query := `
		SELECT
			user_id, username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		FROM USERS
		WHERE user_id = ? AND is_deleted = FALSE
	`

		mock.ExpectQuery(query).
			WithArgs(uint(1)).
			WillReturnError(errors.New("database connection error"))

		// Act
		user, err := repo.FindByID(ctx, 1)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "database connection error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("트랜잭션 롤백 시나리오", func(t *testing.T) {
		// Arrange
		db, mock := setupMockDB(t)
		defer db.Close()

		repo := NewUserRepository(db)

		// 트랜잭션 시작
		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		user := &model.User{
			Username:     "txuser",
			PasswordHash: "$2a$10$hashedpassword",
			Status:       model.UserStatusActive,
			IsDeleted:    false,
			CreatedAt:    time.Now(),
		}

		query := `
		INSERT INTO USERS (
			username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

		mock.ExpectExec(query).
			WithArgs(
				user.Username,
				user.PasswordHash,
				user.PasswordUpdatedAt,
				user.Name,
				user.Email,
				user.Phone,
				user.Organization,
				user.Status,
				user.IsDeleted,
				user.DeletedAt,
				user.CreatedAt,
				user.UpdatedAt,
			).
			WillReturnResult(sqlmock.NewResult(999, 1))

		err = repo.Create(ctx, user)
		assert.NoError(t, err)

		// 트랜잭션 롤백
		mock.ExpectRollback()
		err = tx.Rollback()
		assert.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}