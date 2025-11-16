package errors

import "errors"

var (
	// ErrServiceNotFound is returned when a service is not found
	ErrServiceNotFound = errors.New("service not found")

	// ErrInvalidServiceName is returned when an invalid service name is provided
	ErrInvalidServiceName = errors.New("invalid service name")

	// ErrInvalidStatus is returned when an invalid status is provided
	ErrInvalidStatus = errors.New("invalid status")

	// ErrHealthCheckFailed is returned when a health check fails
	ErrHealthCheckFailed = errors.New("health check failed")

	// ErrHealthCheckTimeout is returned when a health check times out
	ErrHealthCheckTimeout = errors.New("health check timeout")

	// ErrIncidentNotFound is returned when an incident is not found
	ErrIncidentNotFound = errors.New("incident not found")

	// ErrInvalidDateRange is returned when an invalid date range is provided
	ErrInvalidDateRange = errors.New("invalid date range")

	// ErrRepositoryOperation is returned when a repository operation fails
	ErrRepositoryOperation = errors.New("repository operation failed")
)
