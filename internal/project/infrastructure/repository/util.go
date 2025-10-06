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

// fromNullString converts sql.NullString to string
func fromNullString(n sql.NullString) string {
	if !n.Valid {
		return ""
	}
	return n.String
}

// timeToNullTime converts time.Time to sql.NullTime
// Zero time is considered as NULL
func timeToNullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: t, Valid: true}
}

// nullTimeToTime converts sql.NullTime to time.Time
// NULL is converted to zero time
func nullTimeToTime(n sql.NullTime) time.Time {
	if !n.Valid {
		return time.Time{}
	}
	return n.Time
}
