package deployment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewDeployment(t *testing.T) {
	projectID := uint(123)

	deployment := NewDeployment(projectID)

	assert.Equal(t, uint(0), deployment.DeploymentID)
	assert.Equal(t, projectID, deployment.ProjectID())
	assert.Equal(t, DeploymentStatusUntracked, deployment.Status())

	summary, exists := deployment.Summary()
	assert.False(t, exists)
	assert.Equal(t, "", summary)

	eventID, exists := deployment.TektonEventID()
	assert.False(t, exists)
	assert.Equal(t, "", eventID)

	runName, exists := deployment.TektonPipelineRunName()
	assert.False(t, exists)
	assert.Equal(t, "", runName)

	assert.WithinDuration(t, time.Now(), deployment.CreatedAt(), time.Second)

	startedAt, exists := deployment.StartedAt()
	assert.False(t, exists)
	assert.True(t, startedAt.IsZero())

	finishedAt, exists := deployment.FinishedAt()
	assert.False(t, exists)
	assert.True(t, finishedAt.IsZero())

	assert.False(t, deployment.IsCompleted())
}

func TestReconstructDeployment(t *testing.T) {
	t.Run("valid deployment", func(t *testing.T) {
		deploymentID := uint(1)
		projectID := uint(123)
		status := DeploymentStatusSuccess
		summaryVal := "Deployment successful"
		summary := &summaryVal
		eventIDVal := "event-123"
		eventID := &eventIDVal
		runNameVal := "pipeline-run-123"
		runName := &runNameVal
		createdAt := time.Now().Add(-1 * time.Hour)
		startedAtVal := time.Now().Add(-30 * time.Minute)
		startedAt := &startedAtVal
		finishedAtVal := time.Now()
		finishedAt := &finishedAtVal

		deployment, err := ReconstructDeployment(
			deploymentID,
			projectID,
			status,
			summary,
			eventID,
			runName,
			createdAt,
			startedAt,
			finishedAt,
		)

		require.NoError(t, err)
		assert.Equal(t, deploymentID, deployment.DeploymentID)
		assert.Equal(t, projectID, deployment.ProjectID())
		assert.Equal(t, status, deployment.Status())

		s, exists := deployment.Summary()
		assert.True(t, exists)
		assert.Equal(t, summaryVal, s)

		e, exists := deployment.TektonEventID()
		assert.True(t, exists)
		assert.Equal(t, eventIDVal, e)

		r, exists := deployment.TektonPipelineRunName()
		assert.True(t, exists)
		assert.Equal(t, runNameVal, r)

		assert.Equal(t, createdAt, deployment.CreatedAt())

		st, exists := deployment.StartedAt()
		assert.True(t, exists)
		assert.Equal(t, startedAtVal, st)

		ft, exists := deployment.FinishedAt()
		assert.True(t, exists)
		assert.Equal(t, finishedAtVal, ft)

		assert.True(t, deployment.IsCompleted())
	})

	t.Run("invalid status", func(t *testing.T) {
		deployment, err := ReconstructDeployment(
			1,
			123,
			DeploymentStatus("invalid"),
			nil,
			nil,
			nil,
			time.Now(),
			nil,
			nil,
		)

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentStatus)
		assert.Nil(t, deployment)
	})
}

func TestDeployment_SetDeploymentID(t *testing.T) {
	deployment := NewDeployment(123)
	assert.Equal(t, uint(0), deployment.DeploymentID)

	deployment.SetDeploymentID(456)
	assert.Equal(t, uint(456), deployment.DeploymentID)
}

func TestDeployment_InitTektonInfo(t *testing.T) {
	t.Run("set both event ID and run name", func(t *testing.T) {
		deployment := NewDeployment(123)
		eventIDVal := "event-123"
		eventID := &eventIDVal
		runNameVal := "pipeline-run-123"
		runName := &runNameVal

		err := deployment.InitTektonInfo(eventID, runName)

		require.NoError(t, err)
		e, exists := deployment.TektonEventID()
		assert.True(t, exists)
		assert.Equal(t, eventIDVal, e)

		r, exists := deployment.TektonPipelineRunName()
		assert.True(t, exists)
		assert.Equal(t, runNameVal, r)
	})

	t.Run("set only event ID", func(t *testing.T) {
		deployment := NewDeployment(123)
		eventIDVal := "event-456"
		eventID := &eventIDVal

		err := deployment.InitTektonInfo(eventID, nil)

		require.NoError(t, err)
		e, exists := deployment.TektonEventID()
		assert.True(t, exists)
		assert.Equal(t, eventIDVal, e)

		_, exists = deployment.TektonPipelineRunName()
		assert.False(t, exists)
	})

	t.Run("set only run name", func(t *testing.T) {
		deployment := NewDeployment(123)
		runNameVal := "pipeline-run-456"
		runName := &runNameVal

		err := deployment.InitTektonInfo(nil, runName)

		require.NoError(t, err)
		_, exists := deployment.TektonEventID()
		assert.False(t, exists)

		r, exists := deployment.TektonPipelineRunName()
		assert.True(t, exists)
		assert.Equal(t, runNameVal, r)
	})
}

func TestDeployment_UpdateRunningStatus(t *testing.T) {
	t.Run("successful update with summary and startedAt", func(t *testing.T) {
		deployment := NewDeployment(123)
		summaryVal := "running"
		summary := &summaryVal
		startedAtVal := time.Now()
		startedAt := &startedAtVal

		err := deployment.UpdateRunningStatus(summary, startedAt)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())

		s, exists := deployment.Summary()
		assert.True(t, exists)
		assert.Equal(t, summaryVal, s)

		st, exists := deployment.StartedAt()
		assert.True(t, exists)
		assert.Equal(t, startedAtVal, st)
	})

	t.Run("successful update with nil values", func(t *testing.T) {
		deployment := NewDeployment(123)

		err := deployment.UpdateRunningStatus(nil, nil)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())

		_, exists := deployment.Summary()
		assert.False(t, exists)

		_, exists = deployment.StartedAt()
		assert.False(t, exists)
	})

	t.Run("cannot update if already completed", func(t *testing.T) {
		deployment := NewDeployment(123)
		msg := "error"
		_ = deployment.UpdateBackendStatus(DeploymentStatusBackendTriggerFailed, &msg)

		err := deployment.UpdateRunningStatus(nil, nil)

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentTransition)
	})
}

func TestDeployment_UpdateCompleteStatus(t *testing.T) {
	t.Run("successful completion", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		summaryVal := "success"
		summary := &summaryVal
		finishedAt := time.Now()

		err := deployment.UpdateCompleteStatus(DeploymentStatusSuccess, summary, finishedAt)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())

		s, exists := deployment.Summary()
		assert.True(t, exists)
		assert.Equal(t, summaryVal, s)

		ft, exists := deployment.FinishedAt()
		assert.True(t, exists)
		assert.Equal(t, finishedAt, ft)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("can complete from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)
		finishedAt := time.Now()

		err := deployment.UpdateCompleteStatus(DeploymentStatusSuccess, nil, finishedAt)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("invalid status", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		finishedAt := time.Now()

		err := deployment.UpdateCompleteStatus(DeploymentStatusUntracked, nil, finishedAt)

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentStatus)
	})

	t.Run("cannot overwrite completed status with different status", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		finishedAt := time.Now()

		// First complete with success
		err := deployment.UpdateCompleteStatus(DeploymentStatusSuccess, nil, finishedAt)
		require.NoError(t, err)

		// Try to overwrite with failed
		err = deployment.UpdateCompleteStatus(DeploymentStatusFailed, nil, finishedAt)

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentTransition)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
	})

	t.Run("can update completed status with same status (idempotent)", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		finishedAt := time.Now()

		// First complete with success
		err := deployment.UpdateCompleteStatus(DeploymentStatusSuccess, nil, finishedAt)
		require.NoError(t, err)

		// Update again with same status (idempotent)
		newFinishedAt := time.Now()
		err = deployment.UpdateCompleteStatus(DeploymentStatusSuccess, nil, newFinishedAt)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
	})
}

func TestDeployment_UpdateBackendStatus(t *testing.T) {
	t.Run("backend_trigger_failed from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)
		summaryVal := "trigger failed"
		summary := &summaryVal

		before := time.Now()
		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTriggerFailed, summary)
		after := time.Now()

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTriggerFailed, deployment.Status())

		s, exists := deployment.Summary()
		assert.True(t, exists)
		assert.Equal(t, summaryVal, s)

		ft, exists := deployment.FinishedAt()
		assert.True(t, exists)
		assert.True(t, !ft.Before(before))
		assert.True(t, !ft.After(after))
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("backend_tracking_lost from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		summaryVal := "tracking lost"
		summary := &summaryVal

		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTrackingLost, summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingLost, deployment.Status())
		assert.False(t, deployment.IsCompleted()) // backend_tracking_lost is NOT a terminal state
	})

	t.Run("invalid status", func(t *testing.T) {
		deployment := NewDeployment(123)

		err := deployment.UpdateBackendStatus(DeploymentStatusSuccess, nil)

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentStatus)
	})

	t.Run("backend_trigger_failed cannot transition from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)

		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTriggerFailed, nil)

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentTransition)
	})

	t.Run("backend_tracking_failed can transition from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)
		summaryVal := "tracking failed"
		summary := &summaryVal

		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTrackingFailed, summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingFailed, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("backend_tracking_failed can transition from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		summaryVal := "tracking failed during running"
		summary := &summaryVal

		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTrackingFailed, summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingFailed, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("backend_tracking_lost can transition from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)
		summaryVal := "tracking lost"
		summary := &summaryVal

		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTrackingLost, summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingLost, deployment.Status())
		assert.False(t, deployment.IsCompleted())
	})

	t.Run("backend_tracking_failed cannot transition from completed", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		_ = deployment.UpdateCompleteStatus(DeploymentStatusSuccess, nil, time.Now())

		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTrackingFailed, nil)

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentTransition)
	})

	t.Run("backend_tracking_lost cannot transition from completed", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		_ = deployment.UpdateCompleteStatus(DeploymentStatusSuccess, nil, time.Now())

		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTrackingLost, nil)

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentTransition)
	})

	t.Run("backend_tracking_lost does NOT set finishedAt (recoverable state)", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		summaryVal := "tracking lost"
		summary := &summaryVal

		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTrackingLost, summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingLost, deployment.Status())

		// finishedAt should NOT be set for recoverable backend_tracking_lost state
		_, exists := deployment.FinishedAt()
		assert.False(t, exists, "backend_tracking_lost should NOT set finishedAt")

		assert.False(t, deployment.IsCompleted())
	})

	t.Run("backend_tracking_failed DOES set finishedAt (terminal state)", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)
		summaryVal := "tracking failed"
		summary := &summaryVal

		before := time.Now()
		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTrackingFailed, summary)
		after := time.Now()

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingFailed, deployment.Status())

		// finishedAt SHOULD be set for terminal backend_tracking_failed state
		ft, exists := deployment.FinishedAt()
		assert.True(t, exists, "backend_tracking_failed SHOULD set finishedAt")
		assert.True(t, !ft.Before(before))
		assert.True(t, !ft.After(after))

		assert.True(t, deployment.IsCompleted())
	})

	t.Run("recovery from backend_tracking_lost to running clears finishedAt", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.UpdateRunningStatus(nil, nil)

		// Simulate network loss - deployment goes to backend_tracking_lost
		summaryVal := "tracking lost"
		err := deployment.UpdateBackendStatus(DeploymentStatusBackendTrackingLost, &summaryVal)
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingLost, deployment.Status())

		// Verify finishedAt is not set
		_, exists := deployment.FinishedAt()
		assert.False(t, exists)

		// Network recovers - deployment returns to running
		newSummary := "running again"
		err = deployment.UpdateRunningStatus(&newSummary, nil)
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())

		// Verify finishedAt is still not set
		_, exists = deployment.FinishedAt()
		assert.False(t, exists, "finishedAt should remain nil after recovery to running")

		assert.False(t, deployment.IsCompleted())
	})
}

func TestDeployment_IsCompleted(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Deployment)
		completed bool
	}{
		{
			name:      "untracked is not completed",
			setup:     func(d *Deployment) {},
			completed: false,
		},
		{
			name: "running is not completed",
			setup: func(d *Deployment) {
				_ = d.UpdateRunningStatus(nil, nil)
			},
			completed: false,
		},
		{
			name: "success is completed",
			setup: func(d *Deployment) {
				_ = d.UpdateRunningStatus(nil, nil)
				_ = d.UpdateCompleteStatus(DeploymentStatusSuccess, nil, time.Now())
			},
			completed: true,
		},
		{
			name: "failed is completed",
			setup: func(d *Deployment) {
				_ = d.UpdateRunningStatus(nil, nil)
				_ = d.UpdateCompleteStatus(DeploymentStatusFailed, nil, time.Now())
			},
			completed: true,
		},
		{
			name: "cancelled is completed",
			setup: func(d *Deployment) {
				_ = d.UpdateRunningStatus(nil, nil)
				_ = d.UpdateCompleteStatus(DeploymentStatusCancelled, nil, time.Now())
			},
			completed: true,
		},
		{
			name: "backend_trigger_failed is completed",
			setup: func(d *Deployment) {
				_ = d.UpdateBackendStatus(DeploymentStatusBackendTriggerFailed, nil)
			},
			completed: true,
		},
		{
			name: "backend_tracking_failed is completed",
			setup: func(d *Deployment) {
				_ = d.UpdateBackendStatus(DeploymentStatusBackendTrackingFailed, nil)
			},
			completed: true,
		},
		{
			name: "backend_tracking_lost is NOT completed (can be recovered)",
			setup: func(d *Deployment) {
				_ = d.UpdateBackendStatus(DeploymentStatusBackendTrackingLost, nil)
			},
			completed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := NewDeployment(123)
			tt.setup(deployment)

			assert.Equal(t, tt.completed, deployment.IsCompleted())
		})
	}
}
