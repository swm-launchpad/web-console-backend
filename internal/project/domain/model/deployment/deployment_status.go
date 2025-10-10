package deployment

import (
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentStatusPending   DeploymentStatus = "pending"
	DeploymentStatusRunning   DeploymentStatus = "running"
	DeploymentStatusSuccess   DeploymentStatus = "success"
	DeploymentStatusFailed    DeploymentStatus = "failed"
	DeploymentStatusCancelled DeploymentStatus = "cancelled"
)

// String returns the string representation of the status
func (s DeploymentStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s DeploymentStatus) IsValid() bool {
	switch s {
	case DeploymentStatusPending,
		DeploymentStatusRunning,
		DeploymentStatusSuccess,
		DeploymentStatusFailed,
		DeploymentStatusCancelled:
		return true
	default:
		return false
	}
}

// ValidateDeploymentStatus validates the deployment status
func ValidateDeploymentStatus(s DeploymentStatus) error {
	if !s.IsValid() {
		return projecterrors.ErrInvalidDeploymentStatus
	}
	return nil
}
