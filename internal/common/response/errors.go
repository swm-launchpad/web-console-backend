package response

import (
	"errors"
	"net/http"

	autherrors "github.com/swm-launchpad/web-console-backend/internal/common/auth"
	config "github.com/swm-launchpad/web-console-backend/internal/common/config"
	cerrors "github.com/swm-launchpad/web-console-backend/internal/common/errors"
)

// HTTPStatusFromError converts error Kind to HTTP status code
func HTTPStatusFromError(err error) int {
	switch cerrors.KindOf(err) {
	case cerrors.Invalid:
		return http.StatusBadRequest
	case cerrors.NotFound:
		return http.StatusNotFound
	case cerrors.Conflict:
		return http.StatusConflict
	case cerrors.Unauthorized:
		return http.StatusUnauthorized
	case cerrors.Forbidden:
		return http.StatusForbidden
	case cerrors.Unavailable:
		return http.StatusServiceUnavailable
	case cerrors.Timeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// ErrorCodeFromError generates client-facing error code based on the error
// This only handles common errors. Domain-specific errors should be mapped
// in their respective handlers.
func ErrorCodeFromError(err error) string {
	// Check for common errors that apply across all domains
	switch {

	// Auth package errors
	case errors.Is(err, autherrors.ErrTokenExpired):
		return "TOKEN_EXPIRED"
	case errors.Is(err, autherrors.ErrInvalidToken):
		return "INVALID_TOKEN"
	case errors.Is(err, autherrors.ErrUnauthorized):
		return "UNAUTHORIZED"
	case errors.Is(err, autherrors.ErrInvalidRefreshToken):
		return "INVALID_REFRESH_TOKEN"
	case errors.Is(err, autherrors.ErrTokenGenerationFailed):
		return "TOKEN_GENERATION_FAILED"
	case errors.Is(err, autherrors.ErrPasswordMismatch):
		return "PASSWORD_MISMATCH"
	case errors.Is(err, autherrors.ErrPasswordTooWeak):
		return "PASSWORD_TOO_WEAK"
	case errors.Is(err, autherrors.ErrMissingAuthHeader):
		return "MISSING_AUTH_HEADER"
	case errors.Is(err, autherrors.ErrInvalidAuthFormat):
		return "INVALID_AUTH_FORMAT"
	case errors.Is(err, autherrors.ErrMissingToken):
		return "MISSING_TOKEN"

	// Config package errors
	case errors.Is(err, config.ErrMissingJWTSecret):
		return "MISSING_JWT_SECRET"
	case errors.Is(err, config.ErrInvalidDBConfig):
		return "INVALID_DB_CONFIG"
	case errors.Is(err, config.ErrInvalidServerConfig):
		return "INVALID_SERVER_CONFIG"

	// Default to Kind-based code
	default:
		return errorCodeFromKind(cerrors.KindOf(err))
	}
}

// errorCodeFromKind generates error code from error Kind
func errorCodeFromKind(kind cerrors.Kind) string {
	switch kind {
	case cerrors.Invalid:
		return "INVALID_REQUEST"
	case cerrors.NotFound:
		return "NOT_FOUND"
	case cerrors.Conflict:
		return "CONFLICT"
	case cerrors.Unauthorized:
		return "UNAUTHORIZED"
	case cerrors.Forbidden:
		return "FORBIDDEN"
	case cerrors.Unavailable:
		return "SERVICE_UNAVAILABLE"
	case cerrors.Timeout:
		return "TIMEOUT"
	default:
		return "INTERNAL_ERROR"
	}
}

// TranslateError converts an error to HTTP status and error code
// using the provided domain-specific error mapper if available
func TranslateError(err error, errorMapper func(error) (string, bool)) (status int, code string, message string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	status = HTTPStatusFromError(err)

	// Try domain-specific error mapping first
	if errorMapper != nil {
		if mappedCode, ok := errorMapper(err); ok {
			code = mappedCode
		} else {
			code = ErrorCodeFromError(err)
		}
	} else {
		code = ErrorCodeFromError(err)
	}

	message = err.Error()

	return status, code, message
}
