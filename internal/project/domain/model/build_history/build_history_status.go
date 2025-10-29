package build_history

import (
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// BuildHistoryStatus represents the status of a build
// Status is divided into backend-managed states and Tekton-managed states
type BuildHistoryStatus string

const (
	// Backend-managed states
	BuildHistoryStatusUntracked             BuildHistoryStatus = "untracked"               // Initial state, not tracked yet
	BuildHistoryStatusBackendTriggerFailed  BuildHistoryStatus = "backend_trigger_failed"  // Backend failed to trigger Tekton
	BuildHistoryStatusBackendTrackingFailed BuildHistoryStatus = "backend_tracking_failed" // Backend failed to track within 5 minutes
	BuildHistoryStatusBackendTrackingLost   BuildHistoryStatus = "backend_tracking_lost"   // Backend lost tracking (fatal errors)

	// Tekton-managed states
	BuildHistoryStatusRunning   BuildHistoryStatus = "running"   // Tekton: Running
	BuildHistoryStatusSuccess   BuildHistoryStatus = "success"   // Tekton: Success
	BuildHistoryStatusFailed    BuildHistoryStatus = "failed"    // Tekton: Failed
	BuildHistoryStatusCancelled BuildHistoryStatus = "cancelled" // Tekton: Cancelled
	BuildHistoryStatusSkipped   BuildHistoryStatus = "skipped"   // Tekton: Build skipped (no changes)
)

// String returns the string representation of the status
func (s BuildHistoryStatus) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s BuildHistoryStatus) IsValid() bool {
	switch s {
	case BuildHistoryStatusUntracked,
		BuildHistoryStatusBackendTriggerFailed,
		BuildHistoryStatusBackendTrackingFailed,
		BuildHistoryStatusBackendTrackingLost,
		BuildHistoryStatusRunning,
		BuildHistoryStatusSuccess,
		BuildHistoryStatusFailed,
		BuildHistoryStatusCancelled,
		BuildHistoryStatusSkipped:
		return true
	default:
		return false
	}
}

// ValidateBuildHistoryStatus validates the build history status
func ValidateBuildHistoryStatus(s BuildHistoryStatus) error {
	if !s.IsValid() {
		return projecterrors.ErrInvalidBuildStatus
	}
	return nil
}
