package value

import "fmt"

// ServiceStatus represents the operational status of a service
type ServiceStatus string

const (
	StatusOperational ServiceStatus = "operational"
	StatusDegraded    ServiceStatus = "degraded"
	StatusDown        ServiceStatus = "down"
)

// IsValid checks if the status is valid
func (s ServiceStatus) IsValid() bool {
	switch s {
	case StatusOperational, StatusDegraded, StatusDown:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (s ServiceStatus) String() string {
	return string(s)
}

// NewServiceStatus creates and validates a ServiceStatus
func NewServiceStatus(status string) (ServiceStatus, error) {
	s := ServiceStatus(status)
	if !s.IsValid() {
		return "", fmt.Errorf("invalid service status: %s", status)
	}
	return s, nil
}

// IsHealthy returns true if the status indicates a healthy service
func (s ServiceStatus) IsHealthy() bool {
	return s == StatusOperational
}

// IsUnhealthy returns true if the status indicates an unhealthy service
func (s ServiceStatus) IsUnhealthy() bool {
	return s == StatusDown
}
