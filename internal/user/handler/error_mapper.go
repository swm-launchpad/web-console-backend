package handler

import (
	"net/http"

	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
)

// userErrorMap provides mapping from user domain errors to response information
var userErrorMap = map[error]response.ErrorMapping{
	// Repository errors
	usererrors.ErrUserNotFound:      {StatusCode: http.StatusNotFound, Code: "USER_NOT_FOUND", Message: "User not found"},
	usererrors.ErrUserAlreadyExists: {StatusCode: http.StatusConflict, Code: "USER_ALREADY_EXISTS", Message: "User already exists"},

	// Domain errors
	usererrors.ErrInvalidUserData:           {StatusCode: http.StatusBadRequest, Code: "INVALID_USER_DATA", Message: "Invalid user data"},
	usererrors.ErrUserNotActive:             {StatusCode: http.StatusForbidden, Code: "USER_NOT_ACTIVE", Message: "User is not active"},
	usererrors.ErrCannotActivateDeletedUser: {StatusCode: http.StatusBadRequest, Code: "CANNOT_ACTIVATE_DELETED_USER", Message: "Cannot activate deleted user"},
	usererrors.ErrCannotDeleteUser:          {StatusCode: http.StatusBadRequest, Code: "CANNOT_DELETE_USER", Message: "Cannot delete user"},

	// Authentication errors
	usererrors.ErrInvalidCredentials: {StatusCode: http.StatusUnauthorized, Code: "INVALID_CREDENTIALS", Message: "Invalid credentials"},
	usererrors.ErrWeakPassword:       {StatusCode: http.StatusBadRequest, Code: "WEAK_PASSWORD", Message: "Password is too weak"},
	usererrors.ErrInvalidEmail:       {StatusCode: http.StatusBadRequest, Code: "INVALID_EMAIL", Message: "Invalid email format"},

	// Infrastructure errors
	usererrors.ErrDatabaseUnavailable:   {StatusCode: http.StatusServiceUnavailable, Code: "DATABASE_UNAVAILABLE", Message: "Database unavailable"},
	usererrors.ErrDatabaseOperation:     {StatusCode: http.StatusInternalServerError, Code: "DATABASE_OPERATION_FAILED", Message: "Database operation failed"},
	usererrors.ErrTokenGenerationFailed: {StatusCode: http.StatusInternalServerError, Code: "TOKEN_GENERATION_FAILED", Message: "Failed to generate token"},

	// Validation errors
	usererrors.ErrUsernameRequired: {StatusCode: http.StatusBadRequest, Code: "USERNAME_REQUIRED", Message: "Username is required"},
	usererrors.ErrPasswordRequired: {StatusCode: http.StatusBadRequest, Code: "PASSWORD_REQUIRED", Message: "Password is required"},
	usererrors.ErrEmailRequired:    {StatusCode: http.StatusBadRequest, Code: "EMAIL_REQUIRED", Message: "Email is required"},
	usererrors.ErrUsernameTooShort: {StatusCode: http.StatusBadRequest, Code: "USERNAME_TOO_SHORT", Message: "Username must be at least 3 characters long"},
	usererrors.ErrInvalidUserID:    {StatusCode: http.StatusBadRequest, Code: "INVALID_USER_ID", Message: "Invalid user ID"},
	usererrors.ErrPasswordEmpty:    {StatusCode: http.StatusBadRequest, Code: "PASSWORD_EMPTY", Message: "Password cannot be empty"},
	usererrors.ErrValidationFailed: {StatusCode: http.StatusBadRequest, Code: "VALIDATION_FAILED", Message: "Validation failed"},
	usererrors.ErrInvalidFormat:    {StatusCode: http.StatusBadRequest, Code: "INVALID_FORMAT", Message: "Invalid format"},
	usererrors.ErrMissingField:     {StatusCode: http.StatusBadRequest, Code: "MISSING_FIELD", Message: "Required field is missing"},
	usererrors.ErrNoFieldsToUpdate: {StatusCode: http.StatusBadRequest, Code: "NO_FIELDS_TO_UPDATE", Message: "No fields to update"},

	// Duplicate errors
	usererrors.ErrUsernameExists: {StatusCode: http.StatusConflict, Code: "USERNAME_EXISTS", Message: "Username already exists"},
	usererrors.ErrEmailExists:    {StatusCode: http.StatusConflict, Code: "EMAIL_EXISTS", Message: "Email already exists"},

	// GitHub Installation errors
	usererrors.ErrInstallationNotFound:     {StatusCode: http.StatusNotFound, Code: "INSTALLATION_NOT_FOUND", Message: "GitHub installation not found"},
	usererrors.ErrInstallationExists:       {StatusCode: http.StatusConflict, Code: "INSTALLATION_EXISTS", Message: "GitHub installation already exists"},
	usererrors.ErrInstallationRevoked:      {StatusCode: http.StatusGone, Code: "INSTALLATION_REVOKED", Message: "GitHub installation has been revoked. Please reconnect your GitHub account"},
	usererrors.ErrInstallationUnauthorized: {StatusCode: http.StatusForbidden, Code: "INSTALLATION_UNAUTHORIZED", Message: "Unauthorized to access GitHub installation"},
	usererrors.ErrInvalidInstallationID:    {StatusCode: http.StatusBadRequest, Code: "INVALID_INSTALLATION_ID", Message: "Invalid installation ID"},
	usererrors.ErrAccountLoginRequired:     {StatusCode: http.StatusBadRequest, Code: "ACCOUNT_LOGIN_REQUIRED", Message: "Account login is required"},
	usererrors.ErrUserIDRequired:           {StatusCode: http.StatusBadRequest, Code: "USER_ID_REQUIRED", Message: "User ID is required"},
	usererrors.ErrGitHubTokenGenerateFail:  {StatusCode: http.StatusInternalServerError, Code: "GITHUB_TOKEN_GENERATION_FAILED", Message: "Failed to generate GitHub token"},
	usererrors.ErrGitHubAPIFailed:          {StatusCode: http.StatusBadGateway, Code: "GITHUB_API_FAILED", Message: "GitHub API request failed"},
	usererrors.ErrInvalidState:             {StatusCode: http.StatusBadRequest, Code: "INVALID_STATE", Message: "Invalid state parameter"},
}

// mapUserError provides error mapping for user domain
func mapUserError(err error) (response.ErrorMapping, bool) {
	if err == nil {
		return response.ErrorMapping{}, false
	}

	m, ok := userErrorMap[err]
	return m, ok
}
