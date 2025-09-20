package response

import (
	"errors"
	"net/http"
)

// Error code constants for system-level errors only
// All other error codes are defined in their respective packages
const (
	// System errors (SYS_XXX)
	ErrCodeInternalError = "SYS_001"
	ErrCodeDatabaseError = "SYS_002"
	ErrCodeServiceError  = "SYS_003"
	ErrCodeConfigError   = "SYS_004"
	ErrCodeNetworkError  = "SYS_005"
)

// TranslateError converts a domain error to HTTP status and error code using the registry
func TranslateError(err error) (status int, code string, message string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	// Check if error exists in registry
	if def, exists := GetErrorDefinition(err); exists {
		return def.Status, def.Code, def.Message
	}

	// Check for wrapped errors
	currentErr := err
	for currentErr != nil {
		if def, exists := GetErrorDefinition(currentErr); exists {
			return def.Status, def.Code, def.Message
		}
		currentErr = errors.Unwrap(currentErr)
	}

	// Default to internal server error
	return http.StatusInternalServerError, ErrCodeInternalError, "An internal error occurred"
}
