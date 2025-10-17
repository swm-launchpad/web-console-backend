package deployment

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// Deployment represents a deployment entity
type Deployment struct {
	DeploymentID          uint
	projectID             uint
	status                DeploymentStatus
	summary               *string
	tektonEventID         *string
	tektonPipelineRunName *string
	createdAt             time.Time
	startedAt             *time.Time
	finishedAt            *time.Time
}

// NewDeployment creates a new deployment in untracked status
func NewDeployment(
	projectID uint,
) *Deployment {
	return &Deployment{
		projectID: projectID,
		status:    DeploymentStatusUntracked,
		createdAt: time.Now(),
	}
}

// ReconstructDeployment reconstructs a deployment from persistence
// This is used by the repository when loading from the database
func ReconstructDeployment(
	deploymentID uint,
	projectID uint,
	status DeploymentStatus,
	summary *string,
	tektonEventID *string,
	tektonPipelineRunName *string,
	createdAt time.Time,
	startedAt *time.Time,
	finishedAt *time.Time,
) (*Deployment, error) {
	if err := ValidateDeploymentStatus(status); err != nil {
		return nil, err
	}

	return &Deployment{
		DeploymentID:          deploymentID,
		projectID:             projectID,
		status:                status,
		summary:               summary,
		tektonEventID:         tektonEventID,
		tektonPipelineRunName: tektonPipelineRunName,
		createdAt:             createdAt,
		startedAt:             startedAt,
		finishedAt:            finishedAt,
	}, nil
}

// SetDeploymentID sets the deployment ID (used by repository after insert)
func (d *Deployment) SetDeploymentID(id uint) {
	d.DeploymentID = id
}

// ProjectID returns the project ID
func (d *Deployment) ProjectID() uint {
	return d.projectID
}

// CreatedAt returns the creation time
func (d *Deployment) CreatedAt() time.Time {
	return d.createdAt
}

// Status returns the deployment status
func (d *Deployment) Status() DeploymentStatus {
	return d.status
}

// Summary returns the deployment summary
// Returns ("", false) if summary is not set
func (d *Deployment) Summary() (string, bool) {
	if d.summary == nil {
		return "", false
	}
	return *d.summary, true
}

// TektonEventID returns the Tekton event ID
// Returns ("", false) if event ID is not set
func (d *Deployment) TektonEventID() (string, bool) {
	if d.tektonEventID == nil {
		return "", false
	}
	return *d.tektonEventID, true
}

// TektonPipelineRunName returns the Tekton PipelineRun name
// Returns ("", false) if PipelineRun name is not set
func (d *Deployment) TektonPipelineRunName() (string, bool) {
	if d.tektonPipelineRunName == nil {
		return "", false
	}
	return *d.tektonPipelineRunName, true
}

// StartedAt returns the deployment start time
// Returns (time.Time{}, false) if not started yet
func (d *Deployment) StartedAt() (time.Time, bool) {
	if d.startedAt == nil {
		return time.Time{}, false
	}
	return *d.startedAt, true
}

// FinishedAt returns the deployment finish time
// Returns (time.Time{}, false) if not finished yet
func (d *Deployment) FinishedAt() (time.Time, bool) {
	if d.finishedAt == nil {
		return time.Time{}, false
	}
	return *d.finishedAt, true
}

// InitTektonInfo sets Tekton metadata (event ID and PipelineRun name)
// Both parameters are optional (nil allowed)
// This can be called at any time to update Tekton information
func (d *Deployment) InitTektonInfo(
	tektonEventID *string,
	tektonPipelineRunName *string,
) error {
	if tektonEventID != nil {
		d.tektonEventID = tektonEventID
	}
	if tektonPipelineRunName != nil {
		d.tektonPipelineRunName = tektonPipelineRunName
	}
	return nil
}

// UpdateRunningStatus transitions the deployment to running status
// Both summary and startedAt are optional
// Can only be called if deployment is not yet completed
// Clears finishedAt when transitioning to running (defensive cleanup for recovery scenarios)
func (d *Deployment) UpdateRunningStatus(
	summary *string,
	startedAt *time.Time,
) error {
	if d.IsCompleted() {
		return projecterrors.ErrInvalidDeploymentTransition
	}

	d.status = DeploymentStatusRunning
	if summary != nil {
		d.summary = summary
	}
	if startedAt != nil {
		d.startedAt = startedAt
	}
	// Clear finishedAt when transitioning to running
	// This handles recovery from backend_tracking_lost or similar scenarios
	d.finishedAt = nil
	return nil
}

// UpdateCompleteStatus transitions the deployment to a completion status
// status must be one of: success, failed, cancelled
// summary is optional, finishedAt is required
// If already in a completion status (success/failed/cancelled), only allows idempotent update with the same status
func (d *Deployment) UpdateCompleteStatus(
	status DeploymentStatus,
	summary *string,
	finishedAt time.Time,
) error {
	// Validate status is a completion status
	if status != DeploymentStatusSuccess &&
		status != DeploymentStatusFailed &&
		status != DeploymentStatusCancelled {
		return projecterrors.ErrInvalidDeploymentStatus
	}

	// If already in a completion status, only allow idempotent update (same status)
	if d.status == DeploymentStatusSuccess ||
		d.status == DeploymentStatusFailed ||
		d.status == DeploymentStatusCancelled {
		if d.status != status {
			return projecterrors.ErrInvalidDeploymentTransition
		}
	}

	d.status = status
	if summary != nil {
		d.summary = summary
	}
	d.finishedAt = &finishedAt
	return nil
}

// UpdateBackendStatus transitions the deployment to a backend error status
// status must be one of: backend_trigger_failed, backend_tracking_failed, backend_tracking_lost
// summary is optional
// finishedAt is set to current time for terminal states (backend_trigger_failed, backend_tracking_failed)
// but NOT for backend_tracking_lost (recoverable state)
func (d *Deployment) UpdateBackendStatus(
	status DeploymentStatus,
	summary *string,
) error {
	// Validate status is a backend error status
	if status != DeploymentStatusBackendTriggerFailed &&
		status != DeploymentStatusBackendTrackingFailed &&
		status != DeploymentStatusBackendTrackingLost {
		return projecterrors.ErrInvalidDeploymentStatus
	}

	// Validate state transitions for each backend status
	switch status {
	case DeploymentStatusBackendTriggerFailed:
		// Can only transition from untracked (Tekton trigger itself failed)
		if d.status != DeploymentStatusUntracked {
			return projecterrors.ErrInvalidDeploymentTransition
		}
	case DeploymentStatusBackendTrackingFailed:
		// Can transition from any non-completed state (permanent tracking failure)
		if d.IsCompleted() {
			return projecterrors.ErrInvalidDeploymentTransition
		}
	case DeploymentStatusBackendTrackingLost:
		// Can transition from any non-completed state (temporary tracking loss)
		if d.IsCompleted() {
			return projecterrors.ErrInvalidDeploymentTransition
		}
	}

	d.status = status
	if summary != nil {
		d.summary = summary
	}

	// Only set finishedAt for terminal backend states
	// backend_tracking_lost is recoverable and should not have finishedAt set
	if status == DeploymentStatusBackendTriggerFailed ||
		status == DeploymentStatusBackendTrackingFailed {
		now := time.Now()
		d.finishedAt = &now
	}

	return nil
}

// IsCompleted returns true if the deployment is in a terminal status
// backend_tracking_lost is NOT a terminal status as it can be recovered via re-monitoring
func (d *Deployment) IsCompleted() bool {
	return d.status == DeploymentStatusSuccess ||
		d.status == DeploymentStatusFailed ||
		d.status == DeploymentStatusCancelled ||
		d.status == DeploymentStatusBackendTriggerFailed ||
		d.status == DeploymentStatusBackendTrackingFailed
}
