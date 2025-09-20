package response

import (
	"sync"
)

// ErrorDefinition contains the API response details for an error
type ErrorDefinition struct {
	Code    string
	Status  int
	Message string
}

// errorRegistry stores the mapping of errors to their definitions
var (
	errorRegistry = make(map[error]*ErrorDefinition)
	registryMutex sync.RWMutex
)

// RegisterError registers an error with its API response details
// This should be called in init() functions of domain error packages
func RegisterError(err error, code string, status int, message string) {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	errorRegistry[err] = &ErrorDefinition{
		Code:    code,
		Status:  status,
		Message: message,
	}
}

// GetErrorDefinition retrieves the error definition for a given error
func GetErrorDefinition(err error) (*ErrorDefinition, bool) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()

	def, exists := errorRegistry[err]
	return def, exists
}

// ClearRegistry clears all registered errors (useful for testing)
func ClearRegistry() {
	registryMutex.Lock()
	defer registryMutex.Unlock()

	errorRegistry = make(map[error]*ErrorDefinition)
}
