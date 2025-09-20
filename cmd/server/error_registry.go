package main

import (
	"net/http"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	domainerrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/error"
)

// initializeErrorRegistry registers all application errors with their response mappings
func initializeErrorRegistry() {
	// Register auth package errors
	response.RegisterError(auth.ErrInvalidCredentials, auth.CodeInvalidCredentials, http.StatusUnauthorized, "Invalid username or password")
	response.RegisterError(auth.ErrTokenExpired, auth.CodeTokenExpired, http.StatusUnauthorized, "Token has expired")
	response.RegisterError(auth.ErrInvalidToken, auth.CodeInvalidToken, http.StatusUnauthorized, "Invalid or expired token")
	response.RegisterError(auth.ErrUnauthorized, auth.CodeUnauthorized, http.StatusUnauthorized, "Unauthorized access")
	response.RegisterError(auth.ErrInvalidRefreshToken, auth.CodeInvalidRefreshToken, http.StatusUnauthorized, "Invalid refresh token")
	response.RegisterError(auth.ErrUserNotActive, auth.CodeUserNotActive, http.StatusForbidden, "User account is not active")
	response.RegisterError(auth.ErrTokenGenerationFailed, auth.CodeTokenGenerationFailed, http.StatusInternalServerError, "Failed to generate authentication token")
	response.RegisterError(auth.ErrPasswordTooWeak, auth.CodePasswordTooWeak, http.StatusBadRequest, "Password does not meet security requirements")
	response.RegisterError(auth.ErrPasswordMismatch, auth.CodePasswordMismatch, http.StatusBadRequest, "Password does not match")
	response.RegisterError(auth.ErrMissingAuthHeader, auth.CodeMissingAuthHeader, http.StatusUnauthorized, "Authorization header is required")
	response.RegisterError(auth.ErrInvalidAuthFormat, auth.CodeInvalidAuthFormat, http.StatusUnauthorized, "Invalid authorization header format")
	response.RegisterError(auth.ErrMissingToken, auth.CodeMissingToken, http.StatusUnauthorized, "Token is required")

	// Register user domain errors
	response.RegisterError(domainerrors.ErrUserNotFound, domainerrors.CodeUserNotFound, http.StatusNotFound, "User not found")
	response.RegisterError(domainerrors.ErrUserAlreadyExists, domainerrors.CodeUserAlreadyExists, http.StatusConflict, "User already exists")
	response.RegisterError(domainerrors.ErrInvalidUserData, domainerrors.CodeInvalidUserData, http.StatusBadRequest, "Invalid user data")
	response.RegisterError(domainerrors.ErrUserNotActive, domainerrors.CodeUserNotActive, http.StatusForbidden, "User is not active")
	response.RegisterError(domainerrors.ErrCannotActivateDeletedUser, domainerrors.CodeCannotActivateDeletedUser, http.StatusBadRequest, "Cannot activate deleted user")
	response.RegisterError(domainerrors.ErrCannotDeleteUser, domainerrors.CodeCannotDeleteUser, http.StatusBadRequest, "Cannot delete user")

	// Register authentication errors
	response.RegisterError(domainerrors.ErrInvalidCredentials, domainerrors.CodeInvalidCredentials, http.StatusUnauthorized, "Invalid username or password")
	response.RegisterError(domainerrors.ErrWeakPassword, domainerrors.CodeWeakPassword, http.StatusBadRequest, "Password does not meet security requirements")
	response.RegisterError(domainerrors.ErrInvalidEmail, domainerrors.CodeInvalidEmail, http.StatusBadRequest, "Invalid email format")

	// Register validation errors
	response.RegisterError(domainerrors.ErrUsernameRequired, domainerrors.CodeUsernameRequired, http.StatusBadRequest, "Username is required")
	response.RegisterError(domainerrors.ErrPasswordRequired, domainerrors.CodePasswordRequired, http.StatusBadRequest, "Password is required")
	response.RegisterError(domainerrors.ErrEmailRequired, domainerrors.CodeEmailRequired, http.StatusBadRequest, "Email is required")
	response.RegisterError(domainerrors.ErrUsernameTooShort, domainerrors.CodeUsernameTooShort, http.StatusBadRequest, "Username must be at least 3 characters long")
	response.RegisterError(domainerrors.ErrInvalidUserID, domainerrors.CodeInvalidUserID, http.StatusBadRequest, "Invalid user ID")
	response.RegisterError(domainerrors.ErrPasswordEmpty, domainerrors.CodePasswordEmpty, http.StatusBadRequest, "Password cannot be empty")

	// Register duplicate errors
	response.RegisterError(domainerrors.ErrUsernameExists, domainerrors.CodeUsernameExists, http.StatusConflict, "Username already exists")
	response.RegisterError(domainerrors.ErrEmailExists, domainerrors.CodeEmailExists, http.StatusConflict, "Email already exists")

	// Register common validation errors
	response.RegisterError(response.ErrValidationFailed, response.CodeValidationFailed, http.StatusBadRequest, "Validation failed")
	response.RegisterError(response.ErrInvalidFormat, response.CodeInvalidFormat, http.StatusBadRequest, "Invalid format")
	response.RegisterError(response.ErrMissingField, response.CodeMissingField, http.StatusBadRequest, "Required field is missing")
}
