package deployment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeployment(t *testing.T) {
	projectID := uint(123)

	deployment := NewDeployment(projectID)

	assert.Equal(t, uint(0), deployment.DeploymentID())
	assert.Equal(t, projectID, deployment.ProjectID())
	assert.Equal(t, DeploymentStatusPending, deployment.Status())
	assert.Empty(t, deployment.Summary())
	assert.Empty(t, deployment.TektonRef())
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
		tektonRef := "pipeline-run-123"
		createdAt := time.Now().Add(-1 * time.Hour)
		startedAt := time.Now().Add(-30 * time.Minute)
		finishedAt := time.Now()

		deployment, err := ReconstructDeployment(
			deploymentID,
			projectID,
			status,
			summary,
			tektonRef,
			createdAt,
			startedAt,
			finishedAt,
		)

		require.NoError(t, err)
		assert.Equal(t, deploymentID, deployment.DeploymentID())
		assert.Equal(t, projectID, deployment.ProjectID())
		assert.Equal(t, status, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.Equal(t, tektonRef, deployment.TektonRef())
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
			time.Now(),
			time.Time{},
			time.Time{},
		)

		assert.Error(t, err)
		assert.Nil(t, deployment)
	})
}

func TestDeployment_SetDeploymentID(t *testing.T) {
	deployment := NewDeployment(123)
	assert.Equal(t, uint(0), deployment.DeploymentID())

	deployment.SetDeploymentID(456)
	assert.Equal(t, uint(456), deployment.DeploymentID())
}

func TestDeployment_Start(t *testing.T) {
	t.Run("successful start from pending", func(t *testing.T) {
		deployment := NewDeployment(123)
		tektonRef := "pipeline-run-123"

		err := deployment.Start(tektonRef)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())
		assert.Equal(t, tektonRef, deployment.TektonRef())
		assert.WithinDuration(t, time.Now(), deployment.StartedAt(), time.Second)
		assert.True(t, deployment.FinishedAt().IsZero())
		assert.False(t, deployment.IsCompleted())
	})

	t.Run("cannot start from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.Start("ref1")

		err := deployment.Start("ref2")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot start deployment")
		assert.Equal(t, DeploymentStatusRunning, deployment.Status())
		assert.Equal(t, "ref1", deployment.TektonRef())
	})

	t.Run("cannot start from success", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.Start("ref1")
		_ = deployment.Complete("success")

		err := deployment.Start("ref2")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot start deployment")
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
	})
}

func TestDeployment_Complete(t *testing.T) {
	t.Run("successful completion from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.Start("pipeline-run-123")
		summary := "Deployment completed successfully"

		err := deployment.Complete(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cannot complete from pending", func(t *testing.T) {
		deployment := NewDeployment(123)

		err := deployment.Complete("success")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot complete deployment")
		assert.Equal(t, DeploymentStatusPending, deployment.Status())
	})

	t.Run("cannot complete from failed", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.Start("ref1")
		_ = deployment.Fail("error occurred")

		err := deployment.Complete("success")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot complete deployment")
		assert.Equal(t, DeploymentStatusFailed, deployment.Status())
	})
}

func TestDeployment_Fail(t *testing.T) {
	t.Run("successful failure from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.Start("pipeline-run-123")
		summary := "Deployment failed due to build error"

		err := deployment.Fail(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusFailed, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cannot fail from pending", func(t *testing.T) {
		deployment := NewDeployment(123)

		err := deployment.Fail("error")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot fail deployment")
		assert.Equal(t, DeploymentStatusPending, deployment.Status())
	})

	t.Run("cannot fail from success", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.Start("ref1")
		_ = deployment.Complete("success")

		err := deployment.Fail("error")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot fail deployment")
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
	})
}

func TestDeployment_Cancel(t *testing.T) {
	t.Run("successful cancellation from pending", func(t *testing.T) {
		deployment := NewDeployment(123)
		summary := "Deployment cancelled by user"

		err := deployment.Cancel(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusCancelled, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("successful cancellation from running", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.Start("pipeline-run-123")
		summary := "Deployment cancelled by user"

		err := deployment.Cancel(summary)

		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusCancelled, deployment.Status())
		assert.Equal(t, summary, deployment.Summary())
		assert.WithinDuration(t, time.Now(), deployment.FinishedAt(), time.Second)
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cannot cancel from success", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.Start("ref1")
		_ = deployment.Complete("success")

		err := deployment.Cancel("cancel")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot cancel deployment")
		assert.Equal(t, DeploymentStatusSuccess, deployment.Status())
	})

	t.Run("cannot cancel from failed", func(t *testing.T) {
		deployment := NewDeployment(123)
		_ = deployment.Start("ref1")
		_ = deployment.Fail("error")

		err := deployment.Cancel("cancel")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot cancel deployment")
		assert.Equal(t, DeploymentStatusFailed, deployment.Status())
	})
}

func TestDeployment_IsCompleted(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Deployment)
		completed bool
	}{
		{
			name:      "pending is not completed",
			setup:     func(d *Deployment) {},
			completed: false,
		},
		{
			name: "running is not completed",
			setup: func(d *Deployment) {
				_ = d.Start("ref")
			},
			completed: false,
		},
		{
			name: "success is completed",
			setup: func(d *Deployment) {
				_ = d.Start("ref")
				_ = d.Complete("success")
			},
			completed: true,
		},
		{
			name: "failed is completed",
			setup: func(d *Deployment) {
				_ = d.Start("ref")
				_ = d.Fail("error")
			},
			completed: true,
		},
		{
			name: "cancelled is completed",
			setup: func(d *Deployment) {
				_ = d.Cancel("cancelled")
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

		// pending -> running
		err := deployment.Start("pipeline-run-123")
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

		// pending -> running
		err := deployment.Start("pipeline-run-123")
		require.NoError(t, err)

		// running -> failed
		err = deployment.Fail("Build failed")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusFailed, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cancellation from pending", func(t *testing.T) {
		deployment := NewDeployment(123)

		// pending -> cancelled
		err := deployment.Cancel("User cancelled")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusCancelled, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})

	t.Run("cancellation from running", func(t *testing.T) {
		deployment := NewDeployment(123)

		// pending -> running
		err := deployment.Start("pipeline-run-123")
		require.NoError(t, err)

		// running -> cancelled
		err = deployment.Cancel("User cancelled")
		require.NoError(t, err)
		assert.Equal(t, DeploymentStatusCancelled, deployment.Status())
		assert.True(t, deployment.IsCompleted())
	})
}
