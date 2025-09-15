package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/users/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO USERS (
			username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
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
	)

	if err != nil {
		// Check for duplicate username or email
		if isDuplicateError(err) {
			return repository.ErrUserAlreadyExists
		}
		return err
	}

	// Get the auto-generated ID
	lastID, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.UserID = uint(lastID)

	return nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE USERS SET
			password_hash = ?, password_updated_at = ?, name = ?,
			email = ?, phone = ?, organization = ?,
			status = ?, is_deleted = ?, deleted_at = ?, updated_at = ?
		WHERE user_id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
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
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) FindByID(ctx context.Context, userID uint) (*model.User, error) {
	query := `
		SELECT
			user_id, username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		FROM USERS
		WHERE user_id = ? AND is_deleted = FALSE
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.UserID,
		&user.Username,
		&user.PasswordHash,
		&user.PasswordUpdatedAt,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Organization,
		&user.Status,
		&user.IsDeleted,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT
			user_id, username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		FROM USERS
		WHERE username = ? AND is_deleted = FALSE
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.UserID,
		&user.Username,
		&user.PasswordHash,
		&user.PasswordUpdatedAt,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Organization,
		&user.Status,
		&user.IsDeleted,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT
			user_id, username, password_hash, password_updated_at,
			name, email, phone, organization,
			status, is_deleted, deleted_at, created_at, updated_at
		FROM USERS
		WHERE email = ? AND is_deleted = FALSE
	`

	user := &model.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.UserID,
		&user.Username,
		&user.PasswordHash,
		&user.PasswordUpdatedAt,
		&user.Name,
		&user.Email,
		&user.Phone,
		&user.Organization,
		&user.Status,
		&user.IsDeleted,
		&user.DeletedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func (r *userRepository) Delete(ctx context.Context, userID uint) error {
	query := `
		UPDATE USERS SET
			is_deleted = TRUE,
			deleted_at = ?,
			updated_at = ?
		WHERE user_id = ?
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, now, now, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM USERS WHERE username = ? AND is_deleted = FALSE)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM USERS WHERE email = ? AND is_deleted = FALSE)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func isDuplicateError(err error) bool {
	// MySQL duplicate entry error code is 1062
	if err != nil && err.Error() != "" {
		return contains(err.Error(), "Duplicate entry") || contains(err.Error(), "1062")
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && s[0:len(substr)] == substr || contains(s[1:], substr))
}