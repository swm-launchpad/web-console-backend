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
	ErrNicknameRequired = errors.New("nickname is required")
	ErrPasswordRequired = errors.New("password is required")
	ErrEmailRequired    = errors.New("email is required")
	ErrNicknameTooShort = errors.New("nickname must be at least 2 characters long")
	ErrInvalidUserID    = errors.New("invalid user ID")
	ErrPasswordEmpty    = errors.New("password cannot be empty")
	ErrValidationFailed = errors.New("validation failed")
	ErrInvalidFormat    = errors.New("invalid format")
	ErrMissingField     = errors.New("required field is missing")
	ErrNoFieldsToUpdate = errors.New("no fields to update")
)

// Duplicate errors - errors for unique constraint violations
var (
	ErrEmailExists = errors.New("email already exists")
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

// GitHub Installation errors - errors related to GitHub App integration
var (
	ErrInstallationNotFound     = errors.New("GitHub installation not found")
	ErrInstallationExists       = errors.New("GitHub installation already exists")
	ErrInstallationRevoked      = errors.New("GitHub installation has been revoked or uninstalled")
	ErrInstallationUnauthorized = errors.New("unauthorized to access GitHub installation")
	ErrInvalidInstallationID    = errors.New("invalid installation ID")
	ErrAccountLoginRequired     = errors.New("account login is required")
	ErrUserIDRequired           = errors.New("user ID is required")
	ErrGitHubTokenGenerateFail  = errors.New("failed to generate GitHub token")
	ErrGitHubAPIFailed          = errors.New("GitHub API request failed")
	ErrInvalidState             = errors.New("invalid state parameter")
	ErrGitHubNotConfigured      = errors.New("GitHub integration is not configured")
	ErrRepositoryNotFound       = errors.New("repository not found or not accessible")
	ErrRepositoryAccessDenied   = errors.New("repository access denied (not granted to installation)")
)
