package deployment

import (
	"fmt"
	"time"
)

// Deployment represents a deployment entity
type Deployment struct {
	deploymentID uint
	projectID    uint
	status       DeploymentStatus
	summary      string
	tektonRef    string
	createdAt    time.Time
	startedAt    time.Time
	finishedAt   time.Time
}

// NewDeployment creates a new deployment in pending status
func NewDeployment(
	projectID uint,
) *Deployment {
	return &Deployment{
		projectID: projectID,
		status:    DeploymentStatusPending,
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
	tektonRef string,
	createdAt time.Time,
	startedAt time.Time,
	finishedAt time.Time,
) (*Deployment, error) {
	if err := ValidateDeploymentStatus(status); err != nil {
		return nil, err
	}

	return &Deployment{
		deploymentID: deploymentID,
		projectID:    projectID,
		status:       status,
		summary:      summary,
		tektonRef:    tektonRef,
		createdAt:    createdAt,
		startedAt:    startedAt,
		finishedAt:   finishedAt,
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

// TektonRef returns the Tekton PipelineRun reference
func (d *Deployment) TektonRef() string {
	return d.tektonRef
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

// Start transitions the deployment to running status
// Sets the startedAt timestamp and tektonRef
func (d *Deployment) Start(tektonRef string) error {
	if d.status != DeploymentStatusPending {
		return fmt.Errorf("cannot start deployment: current status is %s, expected pending", d.status)
	}

	d.status = DeploymentStatusRunning
	d.tektonRef = tektonRef
	d.startedAt = time.Now()
	return nil
}

// Complete transitions the deployment to success status
// Sets the finishedAt timestamp and optional summary
func (d *Deployment) Complete(summary string) error {
	if d.status != DeploymentStatusRunning {
		return fmt.Errorf("cannot complete deployment: current status is %s, expected running", d.status)
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
		return fmt.Errorf("cannot fail deployment: current status is %s, expected running", d.status)
	}

	d.status = DeploymentStatusFailed
	d.summary = summary
	d.finishedAt = time.Now()
	return nil
}

// Cancel transitions the deployment to cancelled status
// Can be cancelled from pending or running status
func (d *Deployment) Cancel(summary string) error {
	if d.status != DeploymentStatusPending && d.status != DeploymentStatusRunning {
		return fmt.Errorf("cannot cancel deployment: current status is %s, expected pending or running", d.status)
	}

	d.status = DeploymentStatusCancelled
	d.summary = summary
	d.finishedAt = time.Now()
	return nil
}

// IsCompleted returns true if the deployment is in a terminal status
func (d *Deployment) IsCompleted() bool {
	return d.status == DeploymentStatusSuccess ||
		d.status == DeploymentStatusFailed ||
		d.status == DeploymentStatusCancelled
}
