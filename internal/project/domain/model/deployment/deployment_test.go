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

	assert.Equal(t, uint(0), deployment.DeploymentID())
	assert.Equal(t, projectID, deployment.ProjectID())
	assert.Equal(t, DeploymentStatusUntracked, deployment.Status())
	assert.Empty(t, deployment.Summary())
	assert.Empty(t, deployment.TektonEventID())
	assert.Empty(t, deployment.TektonPipelineRunName())
	assert.WithinDuration(t, time.Now(), deployment.CreatedAt(), time.Second)
	assert.True(t, deployment.StartedAt().IsZero())
	assert.True(t, deployment.FinishedAt().IsZero())
	assert.False(t, deployment.IsCompleted())
}

func TestReconstructDeployment(t *testing.T) {
	t.Run("valid deployment", func(t *testing.T) {
		deploymentID := uint(1)
		projectID := uint(123)
		status := DeploymentStatusSuccess
		summary := "Deployment successful"
		tektonEventID := "event-123"
		tektonPipelineRunName := "pipeline-run-123"
		createdAt := time.Now().Add(-1 * time.Hour)
		startedAt := time.Now().Add(-30 * time.Minute)
		finishedAt := time.Now()

		deployment, err := ReconstructDeployment(
			deploymentID,
			projectID,
			status,
			summary,
			tektonEventID,
			tektonPipelineRunName,
			createdAt,
			startedAt,
			finishedAt,
		)

		require.NoError(t, err)
		assert.Equal(t, deploymentID, deployment.DeploymentID())
		assert.Equal(t, projectID, deployment.ProjectID())
		assert.Equal(t, status, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.Equal(t, tektonEventID, deployment.TektonEventID())
		assert.Equal(t, tektonPipelineRunName, deployment.TektonPipelineRunName())
		assert.Equal(t, createdAt, deployment.CreatedAt())
		assert.Equal(t, startedAt, deployment.StartedAt())
		assert.Equal(t, finishedAt, deployment.FinishedAt())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("invalid status", func(t *testing.T) {
		deployment, err := ReconstructDeployment(
			1,
			123,
			DeploymentStatus("invalid"),
			"",
			"",
			"",
			time.Now(),
			time.Time{},
			time.Time{},
		)

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentStatus)
		assert.Nil(t, deployment)
	})
}

func TestDeployment_SetDeploymentID(t *testing.T) {
	deployment := NewDeployment(123)
	assert.Equal(t, uint(0), deployment.DeploymentID())

	deployment.SetDeploymentID(456)
	assert.Equal(t, uint(456), deployment.DeploymentID())
}

func TestDeployment_MarkAsTriggerFailed(t *testing.T) {
	t.Run("successful transition from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)
		summary := "Failed to trigger Tekton API"

		err := deployment.MarkAsTriggerFailed(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTriggerFailed, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cannot transition from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")

		err := deployment.MarkAsTriggerFailed("error")

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentTransition)
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())
	})
}

func TestDeployment_MarkAsTracking(t *testing.T) {
	t.Run("successful tracking from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)
		eventID := "event-123"

		err := deployment.MarkAsTracking(eventID)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusUntracked, deployment.Status()) // Status remains untracked
		assert.Equal(t, eventID, deployment.TektonEventID())
		assert.False(t, deployment.IsCompleted())
	})

	t.Run("cannot set tracking from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")

		err := deployment.MarkAsTracking("event-456")

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentTransition)
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())
	})
}

func TestDeployment_MarkAsTrackingFailed(t *testing.T) {
	t.Run("successful transition from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsTracking("event-123")
		summary := "Failed to track within 5 minutes"

		err := deployment.MarkAsTrackingFailed(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingFailed, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})
}

func TestDeployment_MarkAsTrackingLost(t *testing.T) {
	t.Run("successful transition from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsTracking("event-123")
		summary := "Authentication failed"

		err := deployment.MarkAsTrackingLost(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingLost, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("successful transition from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")
		summary := "PipelineRun not found"

		err := deployment.MarkAsTrackingLost(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingLost, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cannot transition from success", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")
		_ = deployment.Complete("success")

		err := deployment.MarkAsTrackingLost("error")

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentTransition)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
	})
}

func TestDeployment_MarkAsRunning(t *testing.T) {
	t.Run("successful transition from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsTracking("event-123")
		pipelineRunName := "pipeline-run-123"

		err := deployment.MarkAsRunning(pipelineRunName)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())
		assert.Equal(t, pipelineRunName, deployment.TektonPipelineRunName())
		assert.WithinDuration(t, time.Now(), deployment.StartedAt(), time.Second)
		assert.True(t, deployment.FinishedAt().IsZero())
		assert.False(t, deployment.IsCompleted())
	})

	t.Run("cannot transition from success", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")
		_ = deployment.Complete("success")

		err := deployment.MarkAsRunning("pipeline-run-456")

		assert.ErrorIs(t, err, projecterrors.ErrInvalidDeploymentTransition)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
	})
}

func TestDeployment_Complete(t *testing.T) {
	t.Run("successful completion from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")
		summary := "Deployment completed successfully"

		err := deployment.Complete(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cannot complete from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)

		err := deployment.Complete("success")

		assert.ErrorIs(t, err, projecterrors.ErrCannotCompleteDeployment)
		assert.Equal(t, DeploymentStatusUntracked, deployment.Status())
	})

	t.Run("cannot complete from failed", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")
		_ = deployment.Fail("error occurred")

		err := deployment.Complete("success")

		assert.ErrorIs(t, err, projecterrors.ErrCannotCompleteDeployment)
		assert.Equal(t, DeploymentStatusFailed, deployment.Status())
	})
}

func TestDeployment_Fail(t *testing.T) {
	t.Run("successful failure from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")
		summary := "Deployment failed due to build error"

		err := deployment.Fail(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusFailed, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cannot fail from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)

		err := deployment.Fail("error")

		assert.ErrorIs(t, err, projecterrors.ErrCannotFailDeployment)
		assert.Equal(t, DeploymentStatusUntracked, deployment.Status())
	})

	t.Run("cannot fail from success", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")
		_ = deployment.Complete("success")

		err := deployment.Fail("error")

		assert.ErrorIs(t, err, projecterrors.ErrCannotFailDeployment)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
	})
}

func TestDeployment_Cancel(t *testing.T) {
	t.Run("successful cancellation from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")
		summary := "Deployment cancelled by user"

		err := deployment.Cancel(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusCancelled, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cannot cancel from untracked", func(t *testing.T) {
		deployment := NewDeployment(123)

		err := deployment.Cancel("cancelled")

		assert.ErrorIs(t, err, projecterrors.ErrCannotCancelDeployment)
		assert.Equal(t, DeploymentStatusUntracked, deployment.Status())
	})

	t.Run("cannot cancel from success", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.MarkAsRunning("pipeline-run-123")
		_ = deployment.Complete("success")

		err := deployment.Cancel("cancelled")

		assert.ErrorIs(t, err, projecterrors.ErrCannotCancelDeployment)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
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
				_ = d.MarkAsRunning("pipeline-run")
			},
			completed: false,
		},
		{
			name: "success is completed",
			setup: func(d *Deployment) {
				_ = d.MarkAsRunning("pipeline-run")
				_ = d.Complete("success")
			},
			completed: true,
		},
		{
			name: "failed is completed",
			setup: func(d *Deployment) {
				_ = d.MarkAsRunning("pipeline-run")
				_ = d.Fail("error")
			},
			completed: true,
		},
		{
			name: "cancelled is completed",
			setup: func(d *Deployment) {
				_ = d.MarkAsRunning("pipeline-run")
				_ = d.Cancel("cancelled by user")
			},
			completed: true,
		},
		{
			name: "backend_trigger_failed is completed",
			setup: func(d *Deployment) {
				_ = d.MarkAsTriggerFailed("trigger failed")
			},
			completed: true,
		},
		{
			name: "backend_tracking_failed is completed",
			setup: func(d *Deployment) {
				_ = d.MarkAsTracking("event-123")
				_ = d.MarkAsTrackingFailed("tracking failed")
			},
			completed: true,
		},
		{
			name: "backend_tracking_lost is completed",
			setup: func(d *Deployment) {
				_ = d.MarkAsTracking("event-123")
				_ = d.MarkAsTrackingLost("tracking lost")
			},
			completed: true,
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

func TestDeployment_StateTransitions(t *testing.T) {
	t.Run("full success flow", func(t *testing.T) {
		deployment := NewDeployment(123)

		// untracked -> tracking (set event ID)
		err := deployment.MarkAsTracking("event-123")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusUntracked, deployment.Status())
		assert.Equal(t, "event-123", deployment.TektonEventID())

		// untracked -> running
		err = deployment.MarkAsRunning("pipeline-run-123")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())
		assert.False(t, deployment.IsCompleted())

		// running -> success
		err = deployment.Complete("Deployment successful")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("full failure flow", func(t *testing.T) {
		deployment := NewDeployment(123)

		// untracked -> running
		err := deployment.MarkAsRunning("pipeline-run-123")
		require.NoError(t, err)

		// running -> failed
		err = deployment.Fail("Build failed")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusFailed, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("backend trigger failure", func(t *testing.T) {
		deployment := NewDeployment(123)

		// untracked -> backend_trigger_failed
		err := deployment.MarkAsTriggerFailed("Failed to trigger Tekton")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTriggerFailed, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("backend tracking failure", func(t *testing.T) {
		deployment := NewDeployment(123)

		// Set event ID
		_ = deployment.MarkAsTracking("event-123")

		// untracked -> backend_tracking_failed
		err := deployment.MarkAsTrackingFailed("Failed to track within 5 minutes")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingFailed, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("backend tracking lost", func(t *testing.T) {
		deployment := NewDeployment(123)

		// Set event ID
		_ = deployment.MarkAsTracking("event-123")

		// untracked -> backend_tracking_lost
		err := deployment.MarkAsTrackingLost("Authentication failed")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusBackendTrackingLost, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cancellation flow", func(t *testing.T) {
		deployment := NewDeployment(123)

		// untracked -> running
		err := deployment.MarkAsRunning("pipeline-run-123")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())

		// running -> cancelled
		err = deployment.Cancel("User cancelled deployment")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusCancelled, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})
}
