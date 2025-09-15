package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/users/infrastructure/persistence/sqlc"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
)

type userRepository struct {
	queries *sqlc.Queries
}

func NewUserRepository(db sqlc.DBTX) repository.UserRepository {
	return &userRepository{
		queries: sqlc.New(db),
	}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	params := sqlc.CreateUserParams{
		Username:          user.Username,
		PasswordHash:      user.PasswordHash,
		PasswordUpdatedAt: toNullTime(user.PasswordUpdatedAt),
		Name:              toNullString(user.Name),
		Email:             ptrToString(user.Email),
		Phone:             toNullString(user.Phone),
		Organization:      toNullString(user.Organization),
		Status:            sqlc.UsersStatus(user.Status),
		IsDeleted:         user.IsDeleted,
		DeletedAt:         toNullTime(user.DeletedAt),
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         toNullTime(user.UpdatedAt),
	}

	result, err := r.queries.CreateUser(ctx, params)
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
	params := sqlc.UpdateUserParams{
		PasswordHash:      user.PasswordHash,
		PasswordUpdatedAt: toNullTime(user.PasswordUpdatedAt),
		Name:              toNullString(user.Name),
		Email:             ptrToString(user.Email),
		Phone:             toNullString(user.Phone),
		Organization:      toNullString(user.Organization),
		Status:            sqlc.UsersStatus(user.Status),
		IsDeleted:         user.IsDeleted,
		DeletedAt:         toNullTime(user.DeletedAt),
		UpdatedAt:         toNullTime(user.UpdatedAt),
		UserID:            uint32(user.UserID),
	}

	result, err := r.queries.UpdateUser(ctx, params)
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
	sqlcUser, err := r.queries.GetUserByID(ctx, uint32(userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	return toDomainUser(
		uint(sqlcUser.UserID),
		sqlcUser.Username,
		sqlcUser.PasswordHash,
		sqlcUser.PasswordUpdatedAt,
		sqlcUser.Name,
		sqlcUser.Email,
		sqlcUser.Phone,
		sqlcUser.Organization,
		sqlc.UsersStatus(sqlcUser.Status),
		sqlcUser.IsDeleted,
		sqlcUser.DeletedAt,
		sqlcUser.CreatedAt,
		sqlcUser.UpdatedAt,
	), nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	sqlcUser, err := r.queries.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	return toDomainUser(
		uint(sqlcUser.UserID),
		sqlcUser.Username,
		sqlcUser.PasswordHash,
		sqlcUser.PasswordUpdatedAt,
		sqlcUser.Name,
		sqlcUser.Email,
		sqlcUser.Phone,
		sqlcUser.Organization,
		sqlc.UsersStatus(sqlcUser.Status),
		sqlcUser.IsDeleted,
		sqlcUser.DeletedAt,
		sqlcUser.CreatedAt,
		sqlcUser.UpdatedAt,
	), nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	sqlcUser, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrUserNotFound
		}
		return nil, err
	}

	return toDomainUser(
		uint(sqlcUser.UserID),
		sqlcUser.Username,
		sqlcUser.PasswordHash,
		sqlcUser.PasswordUpdatedAt,
		sqlcUser.Name,
		sqlcUser.Email,
		sqlcUser.Phone,
		sqlcUser.Organization,
		sqlc.UsersStatus(sqlcUser.Status),
		sqlcUser.IsDeleted,
		sqlcUser.DeletedAt,
		sqlcUser.CreatedAt,
		sqlcUser.UpdatedAt,
	), nil
}

func (r *userRepository) Delete(ctx context.Context, userID uint) error {
	now := time.Now()
	params := sqlc.DeleteUserParams{
		DeletedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt: sql.NullTime{Time: now, Valid: true},
		UserID:    uint32(userID),
	}

	result, err := r.queries.DeleteUser(ctx, params)
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
	return r.queries.ExistsByUsername(ctx, username)
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.queries.ExistsByEmail(ctx, email)
}

// Helper function for converting between domain and sqlc models
func toDomainUser(
	userID uint,
	username string,
	passwordHash string,
	passwordUpdatedAt sql.NullTime,
	name sql.NullString,
	email string,
	phone sql.NullString,
	organization sql.NullString,
	status sqlc.UsersStatus,
	isDeleted bool,
	deletedAt sql.NullTime,
	createdAt time.Time,
	updatedAt sql.NullTime,
) *model.User {
	return &model.User{
		UserID:            userID,
		Username:          username,
		PasswordHash:      passwordHash,
		PasswordUpdatedAt: fromNullTime(passwordUpdatedAt),
		Name:              fromNullString(name),
		Email:             stringToPtr(email),
		Phone:             fromNullString(phone),
		Organization:      fromNullString(organization),
		Status:            model.UserStatus(status),
		IsDeleted:         isDeleted,
		DeletedAt:         fromNullTime(deletedAt),
		CreatedAt:         createdAt,
		UpdatedAt:         fromNullTime(updatedAt),
	}
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func fromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func fromNullTime(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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