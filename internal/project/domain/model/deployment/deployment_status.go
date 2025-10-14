package deployment

import (
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// DeploymentStatus represents the status of a deployment
// Status is divided into backend-managed states and Tekton-managed states
type DeploymentStatus string

const (
	// Backend-managed states
	DeploymentStatusUntracked             DeploymentStatus = "untracked"               // Initial state, not tracked yet
	DeploymentStatusBackendTriggerFailed  DeploymentStatus = "backend_trigger_failed"  // Backend failed to trigger Tekton
	DeploymentStatusBackendTrackingFailed DeploymentStatus = "backend_tracking_failed" // Backend failed to track within 5 minutes
	DeploymentStatusBackendTrackingLost   DeploymentStatus = "backend_tracking_lost"   // Backend lost tracking (fatal errors)

	// Tekton-managed states
	DeploymentStatusRunning DeploymentStatus = "running" // Tekton: Running
	DeploymentStatusSuccess DeploymentStatus = "success" // Tekton: Success
	DeploymentStatusFailed  DeploymentStatus = "failed"  // Tekton: Failed
)

// String returns the string representation of the status
func (s DeploymentStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s DeploymentStatus) IsValid() bool {
	switch s {
	case DeploymentStatusUntracked,
		DeploymentStatusBackendTriggerFailed,
		DeploymentStatusBackendTrackingFailed,
		DeploymentStatusBackendTrackingLost,
		DeploymentStatusRunning,
		DeploymentStatusSuccess,
		DeploymentStatusFailed:
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
