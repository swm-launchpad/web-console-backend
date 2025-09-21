package response

import (
	"errors"
	"net/http"

	autherrors "github.com/swm-launchpad/web-console-backend/internal/common/auth"
	config "github.com/swm-launchpad/web-console-backend/internal/common/config"
)

// CommonErrorMap provides mapping for common/infrastructure errors
var CommonErrorMap = map[error]ErrorMapping{
	// Auth package errors
	autherrors.ErrTokenExpired:          {StatusCode: http.StatusUnauthorized, Code: "TOKEN_EXPIRED", Message: "Token expired"},
	autherrors.ErrInvalidToken:          {StatusCode: http.StatusUnauthorized, Code: "INVALID_TOKEN", Message: "Invalid token"},
	autherrors.ErrUnauthorized:          {StatusCode: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: "Unauthorized"},
	autherrors.ErrInvalidRefreshToken:   {StatusCode: http.StatusUnauthorized, Code: "INVALID_REFRESH_TOKEN", Message: "Invalid refresh token"},
	autherrors.ErrTokenGenerationFailed: {StatusCode: http.StatusInternalServerError, Code: "TOKEN_GENERATION_FAILED", Message: "Failed to generate token"},
	autherrors.ErrPasswordMismatch:      {StatusCode: http.StatusUnauthorized, Code: "PASSWORD_MISMATCH", Message: "Password mismatch"},
	autherrors.ErrPasswordTooWeak:       {StatusCode: http.StatusBadRequest, Code: "PASSWORD_TOO_WEAK", Message: "Password is too weak"},
	autherrors.ErrMissingAuthHeader:     {StatusCode: http.StatusUnauthorized, Code: "MISSING_AUTH_HEADER", Message: "Missing authorization header"},
	autherrors.ErrInvalidAuthFormat:     {StatusCode: http.StatusBadRequest, Code: "INVALID_AUTH_FORMAT", Message: "Invalid authorization format"},
	autherrors.ErrMissingToken:          {StatusCode: http.StatusUnauthorized, Code: "MISSING_TOKEN", Message: "Missing token"},

	// Config package errors
	config.ErrMissingJWTSecret:    {StatusCode: http.StatusInternalServerError, Code: "MISSING_JWT_SECRET", Message: "Missing JWT secret"},
	config.ErrInvalidDBConfig:     {StatusCode: http.StatusInternalServerError, Code: "INVALID_DB_CONFIG", Message: "Invalid database configuration"},
	config.ErrInvalidServerConfig: {StatusCode: http.StatusInternalServerError, Code: "INVALID_SERVER_CONFIG", Message: "Invalid server configuration"},
}

// MapCommonError maps common package errors to ErrorMapping
func MapCommonError(err error) (ErrorMapping, bool) {
	for commonErr, mapping := range CommonErrorMap {
		if errors.Is(err, commonErr) {
			return mapping, true
		}
	}
	return ErrorMapping{}, false
}
