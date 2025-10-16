// Package errors defines domain-specific errors for the user bounded context.
// These errors represent business-level meanings without transport-specific codes.
package errors

import "errors"

// Repository errors - errors that occur at the data access layer
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

// Domain errors - business logic violations and domain rule errors
var (
	ErrInvalidUserData           = errors.New("invalid user data")
	ErrUserNotActive             = errors.New("user is not active")
	ErrCannotActivateDeletedUser = errors.New("cannot activate deleted user")
	ErrCannotDeleteUser          = errors.New("cannot delete user")
)

// Authentication errors - errors related to user authentication and security
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWeakPassword       = errors.New("password is too weak")
	ErrInvalidEmail       = errors.New("invalid email format")
)

// Validation errors - input validation and data integrity errors
var (
	ErrUsernameRequired = errors.New("username is required")
	ErrPasswordRequired = errors.New("password is required")
	ErrEmailRequired    = errors.New("email is required")
	ErrUsernameTooShort = errors.New("username must be at least 3 characters long")
	ErrInvalidUserID    = errors.New("invalid user ID")
	ErrPasswordEmpty    = errors.New("password cannot be empty")
	ErrValidationFailed = errors.New("validation failed")
	ErrInvalidFormat    = errors.New("invalid format")
	ErrMissingField     = errors.New("required field is missing")
	ErrNoFieldsToUpdate = errors.New("no fields to update")
)

// Duplicate errors - errors for unique constraint violations
var (
	ErrUsernameExists = errors.New("username already exists")
	ErrEmailExists    = errors.New("email already exists")
)

// Infrastructure errors - errors related to data persistence and external services
var (
	ErrDatabaseUnavailable = errors.New("database unavailable")
	ErrDatabaseOperation   = errors.New("database operation failed")
)

// Service errors - errors related to business service operations
var (
	ErrTokenGenerationFailed = errors.New("failed to generate token")
)

// Token errors - errors related to verification tokens
var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenAlreadyUsed   = errors.New("token has already been used")
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrEmailSendFailed    = errors.New("failed to send email")
	ErrTooManyRequests    = errors.New("too many requests, please try again later")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrVerificationFailed = errors.New("verification failed")
)
