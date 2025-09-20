package response

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	cerrors "github.com/swm-launchpad/web-console-backend/internal/common/errors"
)

func TestTranslateError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "NotFound Kind error",
			err:            cerrors.E(cerrors.NotFound, "test", errors.New("not found"), nil),
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "Unauthorized Kind error",
			err:            cerrors.E(cerrors.Unauthorized, "test", errors.New("unauthorized"), nil),
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
		},
		{
			name:           "Token expired error",
			err:            auth.ErrTokenExpired,
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "TOKEN_EXPIRED",
		},
		{
			name:           "Conflict Kind error",
			err:            cerrors.E(cerrors.Conflict, "test", errors.New("conflict"), nil),
			expectedStatus: http.StatusConflict,
			expectedCode:   "CONFLICT",
		},
		{
			name:           "Nil error",
			err:            nil,
			expectedStatus: http.StatusOK,
			expectedCode:   "",
		},
		{
			name:           "Unknown error",
			err:            errors.New("unknown error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
		{
			name:           "Internal Kind error",
			err:            cerrors.E(cerrors.Internal, "test", errors.New("internal"), nil),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "INTERNAL_ERROR",
		},
		{
			name:           "Invalid Kind error",
			err:            cerrors.E(cerrors.Invalid, "test", errors.New("validation"), nil),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code, _ := TranslateError(tt.err, nil)
			assert.Equal(t, tt.expectedStatus, status)
			assert.Equal(t, tt.expectedCode, code)
		})
	}
}

func TestHTTPStatusFromError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "Invalid Kind",
			err:            cerrors.E(cerrors.Invalid, "test", nil, nil),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "NotFound Kind",
			err:            cerrors.E(cerrors.NotFound, "test", nil, nil),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Conflict Kind",
			err:            cerrors.E(cerrors.Conflict, "test", nil, nil),
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "Unauthorized Kind",
			err:            cerrors.E(cerrors.Unauthorized, "test", nil, nil),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Forbidden Kind",
			err:            cerrors.E(cerrors.Forbidden, "test", nil, nil),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Unavailable Kind",
			err:            cerrors.E(cerrors.Unavailable, "test", nil, nil),
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "Timeout Kind",
			err:            cerrors.E(cerrors.Timeout, "test", nil, nil),
			expectedStatus: http.StatusGatewayTimeout,
		},
		{
			name:           "Internal Kind",
			err:            cerrors.E(cerrors.Internal, "test", nil, nil),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Plain error without Kind",
			err:            errors.New("plain error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := HTTPStatusFromError(tt.err)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}
