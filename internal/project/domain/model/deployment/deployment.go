package deployment

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// Deployment represents a deployment entity
type Deployment struct {
	deploymentID          uint
	projectID             uint
	status                DeploymentStatus
	summary               string
	tektonEventID         string
	tektonPipelineRunName string
	createdAt             time.Time
	startedAt             time.Time
	finishedAt            time.Time
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
	summary string,
	tektonEventID string,
	tektonPipelineRunName string,
	createdAt time.Time,
	startedAt time.Time,
	finishedAt time.Time,
) (*Deployment, error) {
	if err := ValidateDeploymentStatus(status); err != nil {
		return nil, err
	}

	return &Deployment{
		deploymentID:          deploymentID,
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

// DeploymentID returns the deployment ID
func (d *Deployment) DeploymentID() uint {
	return d.deploymentID
}

// SetDeploymentID sets the deployment ID (used by repository after insert)
func (d *Deployment) SetDeploymentID(id uint) {
	d.deploymentID = id
}

// ProjectID returns the project ID
func (d *Deployment) ProjectID() uint {
	return d.projectID
}

// Status returns the deployment status
func (d *Deployment) Status() DeploymentStatus {
	return d.status
}

// Summary returns the deployment summary
func (d *Deployment) Summary() string {
	return d.summary
}

// TektonEventID returns the Tekton event ID from API response
func (d *Deployment) TektonEventID() string {
	return d.tektonEventID
}

// TektonPipelineRunName returns the Tekton PipelineRun name
func (d *Deployment) TektonPipelineRunName() string {
	return d.tektonPipelineRunName
}

// CreatedAt returns the creation time
func (d *Deployment) CreatedAt() time.Time {
	return d.createdAt
}

// StartedAt returns the start time
func (d *Deployment) StartedAt() time.Time {
	return d.startedAt
}

// FinishedAt returns the finish time
func (d *Deployment) FinishedAt() time.Time {
	return d.finishedAt
}

// MarkAsTriggerFailed transitions the deployment to backend_trigger_failed status
// Called when backend fails to trigger Tekton API
func (d *Deployment) MarkAsTriggerFailed(summary string) error {
	if d.status != DeploymentStatusUntracked {
		return projecterrors.ErrInvalidDeploymentTransition
	}

	d.status = DeploymentStatusBackendTriggerFailed
	d.summary = summary
	d.finishedAt = time.Now()
	return nil
}

// MarkAsTracking sets the Tekton event ID after successful trigger
// Status remains untracked until we find the PipelineRun
func (d *Deployment) MarkAsTracking(eventID string) error {
	if d.status != DeploymentStatusUntracked {
		return projecterrors.ErrInvalidDeploymentTransition
	}

	d.tektonEventID = eventID
	return nil
}

// MarkAsTrackingFailed transitions the deployment to backend_tracking_failed status
// Called when backend fails to track the deployment within 5 minutes
func (d *Deployment) MarkAsTrackingFailed(summary string) error {
	if d.status != DeploymentStatusUntracked {
		return projecterrors.ErrInvalidDeploymentTransition
	}

	d.status = DeploymentStatusBackendTrackingFailed
	d.summary = summary
	d.finishedAt = time.Now()
	return nil
}

// MarkAsTrackingLost transitions the deployment to backend_tracking_lost status
// Called when fatal errors occur (auth failure, PipelineRun not found)
func (d *Deployment) MarkAsTrackingLost(summary string) error {
	if d.status != DeploymentStatusUntracked {
		return projecterrors.ErrInvalidDeploymentTransition
	}

	d.status = DeploymentStatusBackendTrackingLost
	d.summary = summary
	d.finishedAt = time.Now()
	return nil
}

// MarkAsRunning transitions the deployment to running status
// Sets the PipelineRun name and startedAt timestamp
func (d *Deployment) MarkAsRunning(pipelineRunName string) error {
	if d.status != DeploymentStatusUntracked {
		return projecterrors.ErrInvalidDeploymentTransition
	}

	d.status = DeploymentStatusRunning
	d.tektonPipelineRunName = pipelineRunName
	d.startedAt = time.Now()
	return nil
}

// Complete transitions the deployment to success status
// Sets the finishedAt timestamp and optional summary
func (d *Deployment) Complete(summary string) error {
	if d.status != DeploymentStatusRunning {
		return projecterrors.ErrCannotCompleteDeployment
	}

	d.status = DeploymentStatusSuccess
	d.summary = summary
	d.finishedAt = time.Now()
	return nil
}

// Fail transitions the deployment to failed status
// Sets the finishedAt timestamp and error summary
func (d *Deployment) Fail(summary string) error {
	if d.status != DeploymentStatusRunning {
		return projecterrors.ErrCannotFailDeployment
	}

	d.status = DeploymentStatusFailed
	d.summary = summary
	d.finishedAt = time.Now()
	return nil
}

// IsCompleted returns true if the deployment is in a terminal status
func (d *Deployment) IsCompleted() bool {
	return d.status == DeploymentStatusSuccess ||
		d.status == DeploymentStatusFailed ||
		d.status == DeploymentStatusBackendTriggerFailed ||
		d.status == DeploymentStatusBackendTrackingFailed ||
		d.status == DeploymentStatusBackendTrackingLost
}
