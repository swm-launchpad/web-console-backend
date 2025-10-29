package build_history

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewBuildHistory(t *testing.T) {
	containerID := uint(123)

	buildHistory := NewBuildHistory(containerID)

	assert.Equal(t, uint(0), buildHistory.BuildHistoryID)
	assert.Equal(t, containerID, buildHistory.ContainerID())
	assert.Equal(t, BuildHistoryStatusUntracked, buildHistory.Status())

	summary, exists := buildHistory.Summary()
	assert.False(t, exists)
	assert.Equal(t, "", summary)

	eventID, exists := buildHistory.TektonEventID()
	assert.False(t, exists)
	assert.Equal(t, "", eventID)

	runName, exists := buildHistory.TektonPipelineRunName()
	assert.False(t, exists)
	assert.Equal(t, "", runName)

	commitHash, exists := buildHistory.GitCommitHash()
	assert.False(t, exists)
	assert.Equal(t, "", commitHash)

	assert.WithinDuration(t, time.Now(), buildHistory.CreatedAt(), time.Second)

	startedAt, exists := buildHistory.StartedAt()
	assert.False(t, exists)
	assert.True(t, startedAt.IsZero())

	finishedAt, exists := buildHistory.FinishedAt()
	assert.False(t, exists)
	assert.True(t, finishedAt.IsZero())

	assert.False(t, buildHistory.IsCompleted())
}

func TestReconstructBuildHistory(t *testing.T) {
	t.Run("valid build history", func(t *testing.T) {
		buildHistoryID := uint(1)
		containerID := uint(123)
		status := BuildHistoryStatusSuccess
		summaryVal := "Build successful"
		summary := &summaryVal
		eventIDVal := "event-123"
		eventID := &eventIDVal
		runNameVal := "pipeline-run-123"
		runName := &runNameVal
		commitHashVal := "abc123def456"
		commitHash := &commitHashVal
		createdAt := time.Now().Add(-1 * time.Hour)
		startedAtVal := time.Now().Add(-30 * time.Minute)
		startedAt := &startedAtVal
		finishedAtVal := time.Now()
		finishedAt := &finishedAtVal

		buildHistory, err := ReconstructBuildHistory(
			buildHistoryID,
			containerID,
			status,
			summary,
			eventID,
			runName,
			commitHash,
			createdAt,
			startedAt,
			finishedAt,
		)

		require.NoError(t, err)
		assert.Equal(t, buildHistoryID, buildHistory.BuildHistoryID)
		assert.Equal(t, containerID, buildHistory.ContainerID())
		assert.Equal(t, status, buildHistory.Status())

		s, exists := buildHistory.Summary()
		assert.True(t, exists)
		assert.Equal(t, summaryVal, s)

		e, exists := buildHistory.TektonEventID()
		assert.True(t, exists)
		assert.Equal(t, eventIDVal, e)

		r, exists := buildHistory.TektonPipelineRunName()
		assert.True(t, exists)
		assert.Equal(t, runNameVal, r)

		c, exists := buildHistory.GitCommitHash()
		assert.True(t, exists)
		assert.Equal(t, commitHashVal, c)

		assert.Equal(t, createdAt, buildHistory.CreatedAt())

		st, exists := buildHistory.StartedAt()
		assert.True(t, exists)
		assert.Equal(t, startedAtVal, st)

		ft, exists := buildHistory.FinishedAt()
		assert.True(t, exists)
		assert.Equal(t, finishedAtVal, ft)

		assert.True(t, buildHistory.IsCompleted())
	})

	t.Run("invalid status", func(t *testing.T) {
		buildHistoryID := uint(1)
		containerID := uint(123)
		status := BuildHistoryStatus("invalid_status")
		createdAt := time.Now()

		buildHistory, err := ReconstructBuildHistory(
			buildHistoryID,
			containerID,
			status,
			nil,
			nil,
			nil,
			nil,
			createdAt,
			nil,
			nil,
		)

		require.Error(t, err)
		assert.Nil(t, buildHistory)
		assert.Equal(t, projecterrors.ErrInvalidBuildStatus, err)
	})
}

func TestBuildHistory_SetBuildHistoryID(t *testing.T) {
	buildHistory := NewBuildHistory(123)
	assert.Equal(t, uint(0), buildHistory.BuildHistoryID)

	buildHistory.SetBuildHistoryID(456)
	assert.Equal(t, uint(456), buildHistory.BuildHistoryID)
}

func TestBuildHistory_InitTektonInfo(t *testing.T) {
	buildHistory := NewBuildHistory(123)

	eventID := "event-123"
	runName := "pipeline-run-123"

	err := buildHistory.InitTektonInfo(&eventID, &runName)
	require.NoError(t, err)

	e, exists := buildHistory.TektonEventID()
	assert.True(t, exists)
	assert.Equal(t, eventID, e)

	r, exists := buildHistory.TektonPipelineRunName()
	assert.True(t, exists)
	assert.Equal(t, runName, r)
}

func TestBuildHistory_UpdateRunningStatus(t *testing.T) {
	t.Run("success transition from untracked", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		summaryVal := "Build is running"
		summary := &summaryVal
		startedAtVal := time.Now()
		startedAt := &startedAtVal

		err := buildHistory.UpdateRunningStatus(summary, startedAt)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusRunning, buildHistory.Status())

		s, exists := buildHistory.Summary()
		assert.True(t, exists)
		assert.Equal(t, summaryVal, s)

		st, exists := buildHistory.StartedAt()
		assert.True(t, exists)
		assert.Equal(t, startedAtVal, st)

		_, exists = buildHistory.FinishedAt()
		assert.False(t, exists)
	})

	t.Run("clears finishedAt when transitioning to running", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		// Simulate recovery scenario where finishedAt was set
		buildHistory.finishedAt = ptrTime(time.Now())

		err := buildHistory.UpdateRunningStatus(nil, nil)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusRunning, buildHistory.Status())
		_, exists := buildHistory.FinishedAt()
		assert.False(t, exists)
	})

	t.Run("cannot transition from completed status", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		_ = buildHistory.UpdateCompleteStatus(BuildHistoryStatusSuccess, nil, nil, time.Now())

		err := buildHistory.UpdateRunningStatus(nil, nil)

		require.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidBuildTransition, err)
		assert.Equal(t, BuildHistoryStatusSuccess, buildHistory.Status())
	})
}

func TestBuildHistory_UpdateCompleteStatus(t *testing.T) {
	t.Run("success status", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		summaryVal := "Build completed successfully"
		summary := &summaryVal
		commitHashVal := "abc123def456"
		commitHash := &commitHashVal
		finishedAt := time.Now()

		err := buildHistory.UpdateCompleteStatus(BuildHistoryStatusSuccess, summary, commitHash, finishedAt)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusSuccess, buildHistory.Status())

		s, exists := buildHistory.Summary()
		assert.True(t, exists)
		assert.Equal(t, summaryVal, s)

		c, exists := buildHistory.GitCommitHash()
		assert.True(t, exists)
		assert.Equal(t, commitHashVal, c)

		ft, exists := buildHistory.FinishedAt()
		assert.True(t, exists)
		assert.Equal(t, finishedAt, ft)

		assert.True(t, buildHistory.IsCompleted())
	})

	t.Run("failed status", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		summaryVal := "Build failed"
		summary := &summaryVal
		finishedAt := time.Now()

		err := buildHistory.UpdateCompleteStatus(BuildHistoryStatusFailed, summary, nil, finishedAt)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusFailed, buildHistory.Status())
		assert.True(t, buildHistory.IsCompleted())
	})

	t.Run("cancelled status", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		finishedAt := time.Now()

		err := buildHistory.UpdateCompleteStatus(BuildHistoryStatusCancelled, nil, nil, finishedAt)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusCancelled, buildHistory.Status())
		assert.True(t, buildHistory.IsCompleted())
	})

	t.Run("skipped status", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		summaryVal := "Build skipped (no changes)"
		summary := &summaryVal
		finishedAt := time.Now()

		err := buildHistory.UpdateCompleteStatus(BuildHistoryStatusSkipped, summary, nil, finishedAt)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusSkipped, buildHistory.Status())
		assert.True(t, buildHistory.IsCompleted())
	})

	t.Run("invalid status", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		finishedAt := time.Now()

		err := buildHistory.UpdateCompleteStatus(BuildHistoryStatusRunning, nil, nil, finishedAt)

		require.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidBuildStatus, err)
		assert.Equal(t, BuildHistoryStatusUntracked, buildHistory.Status())
	})

	t.Run("idempotent update with same status", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		finishedAt := time.Now()
		_ = buildHistory.UpdateCompleteStatus(BuildHistoryStatusSuccess, nil, nil, finishedAt)

		err := buildHistory.UpdateCompleteStatus(BuildHistoryStatusSuccess, nil, nil, finishedAt)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusSuccess, buildHistory.Status())
	})

	t.Run("cannot change from one completed status to another", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		finishedAt := time.Now()
		_ = buildHistory.UpdateCompleteStatus(BuildHistoryStatusSuccess, nil, nil, finishedAt)

		err := buildHistory.UpdateCompleteStatus(BuildHistoryStatusFailed, nil, nil, finishedAt)

		require.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidBuildTransition, err)
		assert.Equal(t, BuildHistoryStatusSuccess, buildHistory.Status())
	})
}

func TestBuildHistory_UpdateBackendStatus(t *testing.T) {
	t.Run("backend_trigger_failed from untracked", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		summaryVal := "Failed to trigger Tekton"
		summary := &summaryVal

		err := buildHistory.UpdateBackendStatus(BuildHistoryStatusBackendTriggerFailed, summary)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusBackendTriggerFailed, buildHistory.Status())

		s, exists := buildHistory.Summary()
		assert.True(t, exists)
		assert.Equal(t, summaryVal, s)

		ft, exists := buildHistory.FinishedAt()
		assert.True(t, exists)
		assert.WithinDuration(t, time.Now(), ft, time.Second)

		assert.True(t, buildHistory.IsCompleted())
	})

	t.Run("backend_trigger_failed cannot transition from running", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		_ = buildHistory.UpdateRunningStatus(nil, nil)

		err := buildHistory.UpdateBackendStatus(BuildHistoryStatusBackendTriggerFailed, nil)

		require.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidBuildTransition, err)
		assert.Equal(t, BuildHistoryStatusRunning, buildHistory.Status())
	})

	t.Run("backend_tracking_failed from running", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		_ = buildHistory.UpdateRunningStatus(nil, nil)

		summaryVal := "Tracking failed"
		summary := &summaryVal

		err := buildHistory.UpdateBackendStatus(BuildHistoryStatusBackendTrackingFailed, summary)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusBackendTrackingFailed, buildHistory.Status())

		ft, exists := buildHistory.FinishedAt()
		assert.True(t, exists)
		assert.WithinDuration(t, time.Now(), ft, time.Second)

		assert.True(t, buildHistory.IsCompleted())
	})

	t.Run("backend_tracking_failed cannot transition from completed", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		_ = buildHistory.UpdateCompleteStatus(BuildHistoryStatusSuccess, nil, nil, time.Now())

		err := buildHistory.UpdateBackendStatus(BuildHistoryStatusBackendTrackingFailed, nil)

		require.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidBuildTransition, err)
	})

	t.Run("backend_tracking_lost from running (recoverable)", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)
		_ = buildHistory.UpdateRunningStatus(nil, nil)

		summaryVal := "Tracking lost"
		summary := &summaryVal

		err := buildHistory.UpdateBackendStatus(BuildHistoryStatusBackendTrackingLost, summary)

		require.NoError(t, err)
		assert.Equal(t, BuildHistoryStatusBackendTrackingLost, buildHistory.Status())

		// backend_tracking_lost should NOT set finishedAt (recoverable)
		_, exists := buildHistory.FinishedAt()
		assert.False(t, exists)

		assert.False(t, buildHistory.IsCompleted())
	})

	t.Run("invalid backend status", func(t *testing.T) {
		buildHistory := NewBuildHistory(123)

		err := buildHistory.UpdateBackendStatus(BuildHistoryStatusRunning, nil)

		require.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidBuildStatus, err)
	})
}

func TestBuildHistory_IsCompleted(t *testing.T) {
	tests := []struct {
		name      string
		status    BuildHistoryStatus
		completed bool
	}{
		{"untracked", BuildHistoryStatusUntracked, false},
		{"running", BuildHistoryStatusRunning, false},
		{"success", BuildHistoryStatusSuccess, true},
		{"failed", BuildHistoryStatusFailed, true},
		{"cancelled", BuildHistoryStatusCancelled, true},
		{"skipped", BuildHistoryStatusSkipped, true},
		{"backend_trigger_failed", BuildHistoryStatusBackendTriggerFailed, true},
		{"backend_tracking_failed", BuildHistoryStatusBackendTrackingFailed, true},
		{"backend_tracking_lost", BuildHistoryStatusBackendTrackingLost, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buildHistory := NewBuildHistory(123)
			buildHistory.status = tt.status

			assert.Equal(t, tt.completed, buildHistory.IsCompleted())
		})
	}
}

// Helper function to create pointer to time.Time
func ptrTime(t time.Time) *time.Time {
	return &t
}
