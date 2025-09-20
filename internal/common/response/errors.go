package response

import "errors"

// Common validation errors used across the application
var (
	ErrValidationFailed = errors.New("validation failed")
	ErrInvalidFormat    = errors.New("invalid format")
	ErrMissingField     = errors.New("required field is missing")
)

// Error codes for common validation errors
const (
	CodeValidationFailed = "COM_001"
	CodeInvalidFormat    = "COM_002"
	CodeMissingField     = "COM_003"
)
