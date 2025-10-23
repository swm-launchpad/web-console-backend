package response

import (
	"net/http"
	"testing"

	autherrors "github.com/swm-launchpad/web-console-backend/internal/common/auth"
	config "github.com/swm-launchpad/web-console-backend/internal/common/config"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestMapCommonError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantFound      bool
		wantStatusCode int
		wantCode       string
		wantMessage    string
	}{
		// Auth errors
		{
			name:           "ErrTokenExpired",
			err:            autherrors.ErrTokenExpired,
			wantFound:      true,
			wantStatusCode: http.StatusUnauthorized,
			wantCode:       "TOKEN_EXPIRED",
			wantMessage:    "Token expired",
		},
		{
			name:           "ErrInvalidToken",
			err:            autherrors.ErrInvalidToken,
			wantFound:      true,
			wantStatusCode: http.StatusUnauthorized,
			wantCode:       "INVALID_TOKEN",
			wantMessage:    "Invalid token",
		},
		{
			name:           "ErrUnauthorized",
			err:            autherrors.ErrUnauthorized,
			wantFound:      true,
			wantStatusCode: http.StatusUnauthorized,
			wantCode:       "UNAUTHORIZED",
			wantMessage:    "Unauthorized",
		},

		// Config errors
		{
			name:           "ErrMissingJWTSecret",
			err:            config.ErrMissingJWTSecret,
			wantFound:      true,
			wantStatusCode: http.StatusInternalServerError,
			wantCode:       "MISSING_JWT_SECRET",
			wantMessage:    "Missing JWT secret",
		},
		{
			name:           "ErrInvalidDBConfig",
			err:            config.ErrInvalidDBConfig,
			wantFound:      true,
			wantStatusCode: http.StatusInternalServerError,
			wantCode:       "INVALID_DB_CONFIG",
			wantMessage:    "Invalid database configuration",
		},

		// Project domain errors (cross-domain usage)
		{
			name:           "ErrProjectNotFound",
			err:            projecterrors.ErrProjectNotFound,
			wantFound:      true,
			wantStatusCode: http.StatusNotFound,
			wantCode:       "PROJECT_NOT_FOUND",
			wantMessage:    "Project not found",
		},
		{
			name:           "ErrSlugInvalidLength",
			err:            projecterrors.ErrSlugInvalidLength,
			wantFound:      true,
			wantStatusCode: http.StatusBadRequest,
			wantCode:       "INVALID_SLUG_FORMAT",
			wantMessage:    "Invalid slug format",
		},
		{
			name:           "ErrSlugInvalidFormat",
			err:            projecterrors.ErrSlugInvalidFormat,
			wantFound:      true,
			wantStatusCode: http.StatusBadRequest,
			wantCode:       "INVALID_SLUG_FORMAT",
			wantMessage:    "Invalid slug format",
		},

		// Volume errors (part of project domain)
		{
			name:           "ErrVolumeNotFound",
			err:            projecterrors.ErrVolumeNotFound,
			wantFound:      true,
			wantStatusCode: http.StatusNotFound,
			wantCode:       "VOLUME_NOT_FOUND",
			wantMessage:    "Volume not found",
		},

		// Unmapped error
		{
			name:           "Unmapped error",
			err:            projecterrors.ErrInvalidProjectData,
			wantFound:      false,
			wantStatusCode: 0,
			wantCode:       "",
			wantMessage:    "",
		},

		// Nil error
		{
			name:           "Nil error",
			err:            nil,
			wantFound:      false,
			wantStatusCode: 0,
			wantCode:       "",
			wantMessage:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, found := mapCommonError(tt.err)

			if found != tt.wantFound {
				t.Errorf("mapCommonError() found = %v, want %v", found, tt.wantFound)
				return
			}

			if !found {
				return
			}

			if mapping.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %v, want %v", mapping.StatusCode, tt.wantStatusCode)
			}
			if mapping.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", mapping.Code, tt.wantCode)
			}
			if mapping.Message != tt.wantMessage {
				t.Errorf("Message = %v, want %v", mapping.Message, tt.wantMessage)
			}
		})
	}
}

// TestCrossDomainErrorMapping tests that project domain errors are properly
// mapped in common error mapper for cross-domain usage
func TestCrossDomainErrorMapping(t *testing.T) {
	// This test verifies the fix for the issue where ContainerHandler
	// calls ProjectService.GetProjectBySlug and receives project domain errors
	// that need to be properly mapped to HTTP responses

	tests := []struct {
		name           string
		err            error
		wantStatusCode int
		wantCode       string
	}{
		{
			name:           "Project not found should return 404",
			err:            projecterrors.ErrProjectNotFound,
			wantStatusCode: http.StatusNotFound,
			wantCode:       "PROJECT_NOT_FOUND",
		},
		{
			name:           "Invalid slug length should return 400",
			err:            projecterrors.ErrSlugInvalidLength,
			wantStatusCode: http.StatusBadRequest,
			wantCode:       "INVALID_SLUG_FORMAT",
		},
		{
			name:           "Invalid slug format should return 400",
			err:            projecterrors.ErrSlugInvalidFormat,
			wantStatusCode: http.StatusBadRequest,
			wantCode:       "INVALID_SLUG_FORMAT",
		},
		{
			name:           "Volume not found should return 404",
			err:            projecterrors.ErrVolumeNotFound,
			wantStatusCode: http.StatusNotFound,
			wantCode:       "VOLUME_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, found := mapCommonError(tt.err)

			if !found {
				t.Errorf("Expected error to be mapped in common error map, but it was not found")
				return
			}

			if mapping.StatusCode != tt.wantStatusCode {
				t.Errorf("StatusCode = %v, want %v", mapping.StatusCode, tt.wantStatusCode)
			}

			if mapping.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", mapping.Code, tt.wantCode)
			}
		})
	}
}
