package auth

import "errors"

var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrTokenExpired          = errors.New("token has expired")
	ErrInvalidToken          = errors.New("invalid token")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrPasswordTooWeak       = errors.New("password is too weak")
	ErrPasswordMismatch      = errors.New("password does not match")
	ErrInvalidRefreshToken   = errors.New("invalid refresh token")
	ErrUserNotActive         = errors.New("user is not active")
	ErrTokenGenerationFailed = errors.New("failed to generate token")

	// Middleware specific errors
	ErrMissingAuthHeader = errors.New("authorization header is required")
	ErrInvalidAuthFormat = errors.New("invalid authorization header format")
	ErrMissingToken      = errors.New("token is required")
)
