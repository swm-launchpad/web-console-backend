package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	domainerrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/error"
)

func TestTranslateError_WithRegistry(t *testing.T) {
	// Clear registry first
	ClearRegistry()

	// Manually register test errors for testing
	RegisterError(domainerrors.ErrUserNotFound, domainerrors.CodeUserNotFound, http.StatusNotFound, "User not found")
	RegisterError(domainerrors.ErrUserAlreadyExists, domainerrors.CodeUserAlreadyExists, http.StatusConflict, "User already exists")
	RegisterError(auth.ErrInvalidCredentials, "AUTH_001", http.StatusUnauthorized, "Invalid username or password")
	RegisterError(auth.ErrTokenExpired, "AUTH_002", http.StatusUnauthorized, "Token has expired")

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "User not found error",
			err:            domainerrors.ErrUserNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   domainerrors.CodeUserNotFound,
		},
		{
			name:           "Invalid credentials error",
			err:            auth.ErrInvalidCredentials,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH_001",
		},
		{
			name:           "Token expired error",
			err:            auth.ErrTokenExpired,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "AUTH_002",
		},
		{
			name:           "User already exists error",
			err:            domainerrors.ErrUserAlreadyExists,
			expectedStatus: http.StatusConflict,
			expectedCode:   domainerrors.CodeUserAlreadyExists,
		},
		{
			name:           "Nil error",
			err:            nil,
			expectedStatus: http.StatusOK,
			expectedCode:   "",
		},
		{
			name:           "Unregistered error",
			err:            errors.New("unknown error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   ErrCodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code, _ := TranslateError(tt.err)
			assert.Equal(t, tt.expectedStatus, status)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}
