package build_history

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// BuildHistory represents a build history entity
type BuildHistory struct {
	BuildHistoryID        uint
	containerID           uint
	status                BuildHistoryStatus
	summary               *string
	tektonEventID         *string
	tektonPipelineRunName *string
	gitCommitHash         *string
	createdAt             time.Time
	startedAt             *time.Time
	finishedAt            *time.Time
}

// NewBuildHistory creates a new build history in untracked status
func NewBuildHistory(
	containerID uint,
) *BuildHistory {
	return &BuildHistory{
		containerID: containerID,
		status:      BuildHistoryStatusUntracked,
		createdAt:   time.Now(),
	}
}

// ReconstructBuildHistory reconstructs a build history from persistence
// This is used by the repository when loading from the database
func ReconstructBuildHistory(
	buildHistoryID uint,
	containerID uint,
	status BuildHistoryStatus,
	summary *string,
	tektonEventID *string,
	tektonPipelineRunName *string,
	gitCommitHash *string,
	createdAt time.Time,
	startedAt *time.Time,
	finishedAt *time.Time,
) (*BuildHistory, error) {
	if err := ValidateBuildHistoryStatus(status); err != nil {
		return nil, err
	}

	return &BuildHistory{
		BuildHistoryID:        buildHistoryID,
		containerID:           containerID,
		status:                status,
		summary:               summary,
		tektonEventID:         tektonEventID,
		tektonPipelineRunName: tektonPipelineRunName,
		gitCommitHash:         gitCommitHash,
		createdAt:             createdAt,
		startedAt:             startedAt,
		finishedAt:            finishedAt,
	}, nil
}

// SetBuildHistoryID sets the build history ID (used by repository after insert)
func (b *BuildHistory) SetBuildHistoryID(id uint) {
	b.BuildHistoryID = id
}

// ContainerID returns the container ID
func (b *BuildHistory) ContainerID() uint {
	return b.containerID
}

// CreatedAt returns the creation time
func (b *BuildHistory) CreatedAt() time.Time {
	return b.createdAt
}

// Status returns the build history status
func (b *BuildHistory) Status() BuildHistoryStatus {
	return b.status
}

// Summary returns the build summary
// Returns ("", false) if summary is not set
func (b *BuildHistory) Summary() (string, bool) {
	if b.summary == nil {
		return "", false
	}
	return *b.summary, true
}

// TektonEventID returns the Tekton event ID
// Returns ("", false) if event ID is not set
func (b *BuildHistory) TektonEventID() (string, bool) {
	if b.tektonEventID == nil {
		return "", false
	}
	return *b.tektonEventID, true
}

// TektonPipelineRunName returns the Tekton PipelineRun name
// Returns ("", false) if PipelineRun name is not set
func (b *BuildHistory) TektonPipelineRunName() (string, bool) {
	if b.tektonPipelineRunName == nil {
		return "", false
	}
	return *b.tektonPipelineRunName, true
}

// GitCommitHash returns the git commit hash
// Returns ("", false) if commit hash is not set
func (b *BuildHistory) GitCommitHash() (string, bool) {
	if b.gitCommitHash == nil {
		return "", false
	}
	return *b.gitCommitHash, true
}

// StartedAt returns the build start time
// Returns (time.Time{}, false) if not started yet
func (b *BuildHistory) StartedAt() (time.Time, bool) {
	if b.startedAt == nil {
		return time.Time{}, false
	}
	return *b.startedAt, true
}

// FinishedAt returns the build finish time
// Returns (time.Time{}, false) if not finished yet
func (b *BuildHistory) FinishedAt() (time.Time, bool) {
	if b.finishedAt == nil {
		return time.Time{}, false
	}
	return *b.finishedAt, true
}

// InitTektonInfo sets Tekton metadata (event ID and PipelineRun name)
// Both parameters are optional (nil allowed)
// This can be called at any time to update Tekton information
func (b *BuildHistory) InitTektonInfo(
	tektonEventID *string,
	tektonPipelineRunName *string,
) error {
	if tektonEventID != nil {
		b.tektonEventID = tektonEventID
	}
	if tektonPipelineRunName != nil {
		b.tektonPipelineRunName = tektonPipelineRunName
	}
	return nil
}

// UpdateRunningStatus transitions the build to running status
// summary and startedAt are optional
// Can only be called if build is not yet completed
// Clears finishedAt when transitioning to running (defensive cleanup for recovery scenarios)
func (b *BuildHistory) UpdateRunningStatus(
	summary *string,
	startedAt *time.Time,
) error {
	if b.IsCompleted() {
		return projecterrors.ErrInvalidBuildTransition
	}

	b.status = BuildHistoryStatusRunning
	if summary != nil {
		b.summary = summary
	}
	if startedAt != nil {
		b.startedAt = startedAt
	}
	// Clear finishedAt when transitioning to running
	// This handles recovery from backend_tracking_lost or similar scenarios
	b.finishedAt = nil
	return nil
}

// UpdateCompleteStatus transitions the build to a completion status
// status must be one of: success, failed, cancelled, skipped
// summary, gitCommitHash are optional, finishedAt is required
// If already in a completion status (success/failed/cancelled/skipped), only allows idempotent update with the same status
func (b *BuildHistory) UpdateCompleteStatus(
	status BuildHistoryStatus,
	summary *string,
	gitCommitHash *string,
	finishedAt time.Time,
) error {
	// Validate status is a completion status
	if status != BuildHistoryStatusSuccess &&
		status != BuildHistoryStatusFailed &&
		status != BuildHistoryStatusCancelled &&
		status != BuildHistoryStatusSkipped {
		return projecterrors.ErrInvalidBuildStatus
	}

	// If already in a completion status, only allow idempotent update (same status)
	if b.status == BuildHistoryStatusSuccess ||
		b.status == BuildHistoryStatusFailed ||
		b.status == BuildHistoryStatusCancelled ||
		b.status == BuildHistoryStatusSkipped {
		if b.status != status {
			return projecterrors.ErrInvalidBuildTransition
		}
	}

	b.status = status
	if summary != nil {
		b.summary = summary
	}
	if gitCommitHash != nil {
		b.gitCommitHash = gitCommitHash
	}
	b.finishedAt = &finishedAt
	return nil
}

// UpdateBackendStatus transitions the build to a backend error status
// status must be one of: backend_trigger_failed, backend_tracking_failed, backend_tracking_lost
// summary is optional
// finishedAt is set to current time for terminal states (backend_trigger_failed, backend_tracking_failed)
// but NOT for backend_tracking_lost (recoverable state)
func (b *BuildHistory) UpdateBackendStatus(
	status BuildHistoryStatus,
	summary *string,
) error {
	// Validate status is a backend error status
	if status != BuildHistoryStatusBackendTriggerFailed &&
		status != BuildHistoryStatusBackendTrackingFailed &&
		status != BuildHistoryStatusBackendTrackingLost {
		return projecterrors.ErrInvalidBuildStatus
	}

	// Validate state transitions for each backend status
	switch status {
	case BuildHistoryStatusBackendTriggerFailed:
		// Can only transition from untracked (Tekton trigger itself failed)
		if b.status != BuildHistoryStatusUntracked {
			return projecterrors.ErrInvalidBuildTransition
		}
	case BuildHistoryStatusBackendTrackingFailed:
		// Can transition from any non-completed state (permanent tracking failure)
		if b.IsCompleted() {
			return projecterrors.ErrInvalidBuildTransition
		}
	case BuildHistoryStatusBackendTrackingLost:
		// Can transition from any non-completed state (temporary tracking loss)
		if b.IsCompleted() {
			return projecterrors.ErrInvalidBuildTransition
		}
	}

	b.status = status
	if summary != nil {
		b.summary = summary
	}

	// Only set finishedAt for terminal backend states
	// backend_tracking_lost is recoverable and should not have finishedAt set
	if status == BuildHistoryStatusBackendTriggerFailed ||
		status == BuildHistoryStatusBackendTrackingFailed {
		now := time.Now()
		b.finishedAt = &now
	}

	return nil
}

// IsCompleted returns true if the build is in a terminal status
// backend_tracking_lost is NOT a terminal status as it can be recovered via re-monitoring
func (b *BuildHistory) IsCompleted() bool {
	return b.status == BuildHistoryStatusSuccess ||
		b.status == BuildHistoryStatusFailed ||
		b.status == BuildHistoryStatusCancelled ||
		b.status == BuildHistoryStatusSkipped ||
		b.status == BuildHistoryStatusBackendTriggerFailed ||
		b.status == BuildHistoryStatusBackendTrackingFailed
}
