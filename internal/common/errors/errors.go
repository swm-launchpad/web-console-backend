package errors

import (
	"errors"
	"fmt"
)

// Kind represents a category of errors that can be mapped to transport-specific codes
type Kind string

const (
	// Invalid indicates a validation error or bad request (400)
	Invalid Kind = "invalid"

	// NotFound indicates a resource was not found (404)
	NotFound Kind = "not_found"

	// Conflict indicates a conflicting request (409)
	Conflict Kind = "conflict"

	// Unauthorized indicates authentication is required or failed (401)
	Unauthorized Kind = "unauthorized"

	// Forbidden indicates the authenticated user lacks permission (403)
	Forbidden Kind = "forbidden"

	// Unavailable indicates a service is unavailable (503)
	Unavailable Kind = "unavailable"

	// Timeout indicates an operation timed out (504)
	Timeout Kind = "timeout"

	// Internal indicates an internal server error (500)
	Internal Kind = "internal"
)

// Error is a custom error type that wraps errors with additional context
type Error struct {
	Kind Kind           // The category of error
	Op   string         // The operation being performed
	Err  error          // The underlying error
	Meta map[string]any // Additional metadata
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	return e.Err.Error()
}

// Unwrap returns the underlying error for errors.Is/As support
func (e *Error) Unwrap() error {
	return e.Err
}

// E creates a new Error with the given parameters
func E(kind Kind, op string, err error, meta map[string]any) error {
	if err == nil {
		err = errors.New(string(kind))
	}
	return &Error{
		Kind: kind,
		Op:   op,
		Err:  err,
		Meta: meta,
	}
}

// KindOf returns the Kind of the given error
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}

	// Check wrapped errors
	for err != nil {
		var e *Error
		if errors.As(err, &e) {
			return e.Kind
		}
		err = errors.Unwrap(err)
	}

	return Internal
}

// MetaOf returns the metadata of the given error
func MetaOf(err error) map[string]any {
	var e *Error
	if errors.As(err, &e) {
		return e.Meta
	}
	return nil
}

// Is checks if the target error is of the same kind
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target
func As(err error, target any) bool {
	return errors.As(err, target)
}
