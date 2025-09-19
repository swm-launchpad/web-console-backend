package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure/sqlc"
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
		Email:             user.Email,
		Phone:             toNullString(user.Phone),
		Organization:      toNullString(user.Organization),
		Status:            sqlc.UsersStatus(user.Status),
		IsDeleted:         user.IsDeleted,
		DeletedAt:         toNullTime(user.DeletedAt),
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         toNullTime(user.UpdatedAt),
	}

	result, err := r.queriesWithContext(ctx).CreateUser(ctx, params)
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
		Email:             user.Email,
		Phone:             toNullString(user.Phone),
		Organization:      toNullString(user.Organization),
		Status:            sqlc.UsersStatus(user.Status),
		IsDeleted:         user.IsDeleted,
		DeletedAt:         toNullTime(user.DeletedAt),
		UpdatedAt:         toNullTime(user.UpdatedAt),
		UserID:            uint32(user.UserID),
	}

	result, err := r.queriesWithContext(ctx).UpdateUser(ctx, params)
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
	sqlcUser, err := r.queriesWithContext(ctx).GetUserByID(ctx, uint32(userID))
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
	sqlcUser, err := r.queriesWithContext(ctx).GetUserByUsername(ctx, username)
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
	sqlcUser, err := r.queriesWithContext(ctx).GetUserByEmail(ctx, email)
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

	result, err := r.queriesWithContext(ctx).DeleteUser(ctx, params)
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
	return r.queriesWithContext(ctx).ExistsByUsername(ctx, username)
}

func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.queriesWithContext(ctx).ExistsByEmail(ctx, email)
}

func (r *userRepository) queriesWithContext(ctx context.Context) *sqlc.Queries {
	if tx, ok := db.GetTx(ctx); ok {
		return r.queries.WithTx(tx)
	}
	return r.queries
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
		PasswordUpdatedAt: nullTimeToPtr(passwordUpdatedAt),
		Name:              nullStringToPtr(name),
		Email:             email,
		Phone:             nullStringToPtr(phone),
		Organization:      nullStringToPtr(organization),
		Status:            model.UserStatus(status),
		IsDeleted:         isDeleted,
		DeletedAt:         nullTimeToPtr(deletedAt),
		CreatedAt:         createdAt,
		UpdatedAt:         nullTimeToPtr(updatedAt),
	}
}

// isDuplicateError checks if the error is a duplicate key error
// MySQL duplicate entry error code is 1062
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "Duplicate entry") || strings.Contains(errMsg, "UNIQUE constraint failed")
}

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullStringToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	value := ns.String
	return &value
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	value := nt.Time
	return &value
}
