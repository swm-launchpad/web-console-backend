package response

import (
	"errors"
	"net/http"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	domainerrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/error"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

// Error code constants grouped by domain
const (
	// Authentication errors (AUTH_XXX)
	ErrCodeInvalidCredentials    = "AUTH_001"
	ErrCodeTokenExpired          = "AUTH_002"
	ErrCodeInvalidToken          = "AUTH_003"
	ErrCodeUnauthorized          = "AUTH_004"
	ErrCodeInvalidRefreshToken   = "AUTH_005"
	ErrCodeUserNotActive         = "AUTH_006"
	ErrCodeTokenGenerationFailed = "AUTH_007"
	ErrCodeMissingAuthHeader     = "AUTH_008"
	ErrCodeInvalidAuthFormat     = "AUTH_009"
	ErrCodeMissingToken          = "AUTH_010"

	// User errors (USER_XXX)
	ErrCodeUserNotFound      = "USER_001"
	ErrCodeUserAlreadyExists = "USER_002"
	ErrCodeInvalidUserID     = "USER_003"

	// Validation errors (VAL_XXX)
	ErrCodeValidationFailed = "VAL_001"
	ErrCodeInvalidFormat    = "VAL_002"
	ErrCodePasswordTooWeak  = "VAL_003"
	ErrCodePasswordMismatch = "VAL_004"
	ErrCodeMissingField     = "VAL_005"

	// System errors (SYS_XXX)
	ErrCodeInternalError = "SYS_001"
	ErrCodeDatabaseError = "SYS_002"
	ErrCodeServiceError  = "SYS_003"
	ErrCodeConfigError   = "SYS_004"
	ErrCodeNetworkError  = "SYS_005"
)

// ErrorMapping represents the mapping of an error to HTTP status and error code
type ErrorMapping struct {
	Status  int
	Code    string
	Message string
}

// errorMappings defines the mapping of domain errors to API errors
var errorMappings = map[error]ErrorMapping{
	// Auth errors from common/auth package
	auth.ErrInvalidCredentials: {
		Status:  http.StatusUnauthorized,
		Code:    ErrCodeInvalidCredentials,
		Message: "Invalid username or password",
	},
	auth.ErrTokenExpired: {
		Status:  http.StatusUnauthorized,
		Code:    ErrCodeTokenExpired,
		Message: "Token has expired",
	},
	auth.ErrInvalidToken: {
		Status:  http.StatusUnauthorized,
		Code:    ErrCodeInvalidToken,
		Message: "Invalid or expired token",
	},
	auth.ErrUnauthorized: {
		Status:  http.StatusUnauthorized,
		Code:    ErrCodeUnauthorized,
		Message: "Unauthorized access",
	},
	auth.ErrPasswordTooWeak: {
		Status:  http.StatusBadRequest,
		Code:    ErrCodePasswordTooWeak,
		Message: "Password does not meet security requirements",
	},
	auth.ErrPasswordMismatch: {
		Status:  http.StatusBadRequest,
		Code:    ErrCodePasswordMismatch,
		Message: "Password does not match",
	},
	auth.ErrInvalidRefreshToken: {
		Status:  http.StatusUnauthorized,
		Code:    ErrCodeInvalidRefreshToken,
		Message: "Invalid refresh token",
	},
	auth.ErrUserNotActive: {
		Status:  http.StatusForbidden,
		Code:    ErrCodeUserNotActive,
		Message: "User account is not active",
	},
	auth.ErrTokenGenerationFailed: {
		Status:  http.StatusInternalServerError,
		Code:    ErrCodeTokenGenerationFailed,
		Message: "Failed to generate authentication token",
	},

	// Domain authentication errors
	domainerrors.ErrInvalidCredentials: {
		Status:  http.StatusUnauthorized,
		Code:    ErrCodeInvalidCredentials,
		Message: "Invalid username or password",
	},
	domainerrors.ErrWeakPassword: {
		Status:  http.StatusBadRequest,
		Code:    ErrCodePasswordTooWeak,
		Message: "Password does not meet security requirements",
	},
	domainerrors.ErrInvalidEmail: {
		Status:  http.StatusBadRequest,
		Code:    ErrCodeInvalidFormat,
		Message: "Invalid email format",
	},

	// Domain user errors
	domainerrors.ErrUserNotFound: {
		Status:  http.StatusNotFound,
		Code:    ErrCodeUserNotFound,
		Message: "User not found",
	},
	domainerrors.ErrUserAlreadyExists: {
		Status:  http.StatusConflict,
		Code:    ErrCodeUserAlreadyExists,
		Message: "User already exists",
	},
	domainerrors.ErrUsernameExists: {
		Status:  http.StatusConflict,
		Code:    ErrCodeUserAlreadyExists,
		Message: "Username already exists",
	},
	domainerrors.ErrEmailExists: {
		Status:  http.StatusConflict,
		Code:    ErrCodeUserAlreadyExists,
		Message: "Email already exists",
	},
	domainerrors.ErrUserNotActive: {
		Status:  http.StatusForbidden,
		Code:    ErrCodeUserNotActive,
		Message: "User account is not active",
	},

	// Domain validation errors
	domainerrors.ErrUsernameRequired: {
		Status:  http.StatusBadRequest,
		Code:    ErrCodeMissingField,
		Message: "Username is required",
	},
	domainerrors.ErrPasswordRequired: {
		Status:  http.StatusBadRequest,
		Code:    ErrCodeMissingField,
		Message: "Password is required",
	},
	domainerrors.ErrEmailRequired: {
		Status:  http.StatusBadRequest,
		Code:    ErrCodeMissingField,
		Message: "Email is required",
	},
	domainerrors.ErrUsernameTooShort: {
		Status:  http.StatusBadRequest,
		Code:    ErrCodeValidationFailed,
		Message: "Username must be at least 3 characters long",
	},

	// Repository errors
	repository.ErrUserNotFound: {
		Status:  http.StatusNotFound,
		Code:    ErrCodeUserNotFound,
		Message: "User not found",
	},
	repository.ErrUserAlreadyExists: {
		Status:  http.StatusConflict,
		Code:    ErrCodeUserAlreadyExists,
		Message: "User already exists",
	},
}

// TranslateError converts a domain error to HTTP status and error code
func TranslateError(err error) (status int, code string, message string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	// Check if error exists in mapping
	if mapping, exists := errorMappings[err]; exists {
		return mapping.Status, mapping.Code, mapping.Message
	}

	// Check for wrapped errors
	for domainErr, mapping := range errorMappings {
		if errors.Is(err, domainErr) {
			return mapping.Status, mapping.Code, mapping.Message
		}
	}

	// Default to internal server error
	return http.StatusInternalServerError, ErrCodeInternalError, "An internal error occurred"
}

// GetErrorMessage returns a user-friendly error message based on error code
func GetErrorMessage(code string) string {
	switch code {
	// Auth errors
	case ErrCodeInvalidCredentials:
		return "Invalid username or password"
	case ErrCodeTokenExpired:
		return "Token has expired"
	case ErrCodeInvalidToken:
		return "Invalid or expired token"
	case ErrCodeUnauthorized:
		return "Unauthorized access"
	case ErrCodeMissingAuthHeader:
		return "Authorization header is required"
	case ErrCodeInvalidAuthFormat:
		return "Invalid authorization header format"
	case ErrCodeMissingToken:
		return "Token is required"

	// User errors
	case ErrCodeUserNotFound:
		return "User not found"
	case ErrCodeUserAlreadyExists:
		return "User already exists"
	case ErrCodeInvalidUserID:
		return "Invalid user ID"

	// Validation errors
	case ErrCodeValidationFailed:
		return "Validation failed"
	case ErrCodeInvalidFormat:
		return "Invalid format"
	case ErrCodePasswordTooWeak:
		return "Password does not meet security requirements"
	case ErrCodePasswordMismatch:
		return "Password does not match"
	case ErrCodeMissingField:
		return "Required field is missing"

	// System errors
	case ErrCodeInternalError:
		return "An internal error occurred"
	case ErrCodeDatabaseError:
		return "Database operation failed"
	case ErrCodeServiceError:
		return "Service temporarily unavailable"

	default:
		return "An error occurred"
	}
}
