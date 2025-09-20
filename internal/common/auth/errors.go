package auth

import "errors"

// Token and authentication middleware errors
var (
	ErrTokenExpired          = errors.New("token has expired")
	ErrInvalidToken          = errors.New("invalid token")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrPasswordMismatch      = errors.New("password does not match")
	ErrPasswordTooWeak       = errors.New("password is too weak")
	ErrInvalidRefreshToken   = errors.New("invalid refresh token")
	ErrTokenGenerationFailed = errors.New("failed to generate token")

	// Middleware specific errors
	ErrMissingAuthHeader = errors.New("authorization header is required")
	ErrInvalidAuthFormat = errors.New("invalid authorization header format")
	ErrMissingToken      = errors.New("token is required")
)
