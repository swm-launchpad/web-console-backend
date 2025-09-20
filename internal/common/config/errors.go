package config

import "errors"

// Configuration validation errors
var (
	ErrMissingJWTSecret    = errors.New("JWT_SECRET is required")
	ErrInvalidDBConfig     = errors.New("database configuration is invalid")
	ErrInvalidServerConfig = errors.New("server configuration is invalid")
)
