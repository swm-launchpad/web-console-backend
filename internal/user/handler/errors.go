package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
)

// RespondWithError handles error with user domain error mapping
func RespondWithError(c *gin.Context, err error) {
	response.HandleErrorWithMapper(c, err, MapUserErrorCode)
}

// RespondWithErrorMessage handles error with custom message
func RespondWithErrorMessage(c *gin.Context, err error, customMessage string) {
	status, code, _ := response.TranslateError(err, MapUserErrorCode)
	response.Error(c, status, code, customMessage)
}

// MapUserErrorCode maps user domain errors to API error codes
func MapUserErrorCode(err error) (string, bool) {
	switch {
	// User domain errors
	case errors.Is(err, usererrors.ErrUserNotFound):
		return "USER_NOT_FOUND", true
	case errors.Is(err, usererrors.ErrUserAlreadyExists):
		return "USER_ALREADY_EXISTS", true
	case errors.Is(err, usererrors.ErrInvalidUserData):
		return "INVALID_USER_DATA", true
	case errors.Is(err, usererrors.ErrUserNotActive):
		return "USER_NOT_ACTIVE", true
	case errors.Is(err, usererrors.ErrCannotActivateDeletedUser):
		return "CANNOT_ACTIVATE_DELETED_USER", true
	case errors.Is(err, usererrors.ErrCannotDeleteUser):
		return "CANNOT_DELETE_USER", true

	// Authentication errors
	case errors.Is(err, usererrors.ErrInvalidCredentials):
		return "INVALID_CREDENTIALS", true
	case errors.Is(err, usererrors.ErrWeakPassword):
		return "WEAK_PASSWORD", true
	case errors.Is(err, usererrors.ErrInvalidEmail):
		return "INVALID_EMAIL", true

	// Validation errors
	case errors.Is(err, usererrors.ErrUsernameRequired):
		return "USERNAME_REQUIRED", true
	case errors.Is(err, usererrors.ErrPasswordRequired):
		return "PASSWORD_REQUIRED", true
	case errors.Is(err, usererrors.ErrEmailRequired):
		return "EMAIL_REQUIRED", true
	case errors.Is(err, usererrors.ErrUsernameTooShort):
		return "USERNAME_TOO_SHORT", true
	case errors.Is(err, usererrors.ErrInvalidUserID):
		return "INVALID_USER_ID", true
	case errors.Is(err, usererrors.ErrPasswordEmpty):
		return "PASSWORD_EMPTY", true
	case errors.Is(err, usererrors.ErrValidationFailed):
		return "VALIDATION_FAILED", true
	case errors.Is(err, usererrors.ErrInvalidFormat):
		return "INVALID_FORMAT", true
	case errors.Is(err, usererrors.ErrMissingField):
		return "MISSING_FIELD", true

	// Duplicate errors
	case errors.Is(err, usererrors.ErrUsernameExists):
		return "USERNAME_EXISTS", true
	case errors.Is(err, usererrors.ErrEmailExists):
		return "EMAIL_EXISTS", true

	default:
		return "", false
	}
}
