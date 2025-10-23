package response

import (
	"net/http"

	autherrors "github.com/swm-launchpad/web-console-backend/internal/common/auth"
	config "github.com/swm-launchpad/web-console-backend/internal/common/config"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// ErrorMapping contains complete error mapping information
type ErrorMapping struct {
	StatusCode int
	Code       string
	Message    string
}

// ErrorMapper is a function type for domain-specific error mapping
type ErrorMapper func(error) (ErrorMapping, bool)

// commonErrorMap provides mapping for common/infrastructure errors
// and frequently used cross-domain errors
var commonErrorMap = map[error]ErrorMapping{
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

	// Project domain errors (cross-domain usage)
	// These errors are used by other bounded contexts (e.g., container, volume handlers)
	// when they interact with project resources
	projecterrors.ErrProjectNotFound:   {StatusCode: http.StatusNotFound, Code: "PROJECT_NOT_FOUND", Message: "Project not found"},
	projecterrors.ErrSlugInvalidLength: {StatusCode: http.StatusBadRequest, Code: "INVALID_SLUG_FORMAT", Message: "Invalid slug format"},
	projecterrors.ErrSlugInvalidFormat: {StatusCode: http.StatusBadRequest, Code: "INVALID_SLUG_FORMAT", Message: "Invalid slug format"},

	// Volume errors (part of project domain, used cross-domain)
	projecterrors.ErrVolumeNotFound: {StatusCode: http.StatusNotFound, Code: "VOLUME_NOT_FOUND", Message: "Volume not found"},
}

// mapCommonError maps common package errors to ErrorMapping (internal use)
func mapCommonError(err error) (ErrorMapping, bool) {
	if err == nil {
		return ErrorMapping{}, false
	}

	m, ok := commonErrorMap[err]
	return m, ok
}
