package repository

import (
	"database/sql"
	"strings"
	"time"
)

// isDuplicateError checks if the error is a duplicate entry error
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "unique")
}

// toNullString converts a string to sql.NullString
func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// stringPtrToNullString converts *string to sql.NullString
// nil pointer is considered as NULL
func stringPtrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// nullStringToStringPtr converts sql.NullString to *string
// NULL is converted to nil pointer
func nullStringToStringPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	return &n.String
}

// timePtrToNullTime converts *time.Time to sql.NullTime
// nil pointer is considered as NULL
func timePtrToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// nullTimeToTimePtr converts sql.NullTime to *time.Time
// NULL is converted to nil pointer
func nullTimeToTimePtr(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	return &n.Time
}
