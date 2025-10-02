package infrastructure

import "strings"

// isDuplicateError checks if the error is a duplicate entry error
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "unique")
}
