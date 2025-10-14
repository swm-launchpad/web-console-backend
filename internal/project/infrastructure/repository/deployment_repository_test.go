package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
)

// TestDeploymentRepository_ErrorHandling tests error handling with invalid inputs
// Note: All functional tests are in integration tests with real database
func TestDeploymentRepository_ErrorHandling(t *testing.T) {
	t.Run("Create with nil deployment", func(t *testing.T) {
		repo := NewDeploymentRepository(nil)
		err := repo.Create(context.Background(), nil)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("Save with nil deployment", func(t *testing.T) {
		repo := NewDeploymentRepository(nil)
		err := repo.Save(context.Background(), nil)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("Save with zero deployment ID", func(t *testing.T) {
		repo := NewDeploymentRepository(nil)
		d := deployment.NewDeployment(123)
		err := repo.Save(context.Background(), d)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("FindByID with zero ID", func(t *testing.T) {
		repo := NewDeploymentRepository(nil)
		_, err := repo.FindByID(context.Background(), 0)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("FindLatestByProjectID with zero ID", func(t *testing.T) {
		repo := NewDeploymentRepository(nil)
		_, err := repo.FindLatestByProjectID(context.Background(), 0)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("FindByProjectID with zero ID", func(t *testing.T) {
		repo := NewDeploymentRepository(nil)
		_, err := repo.FindByProjectID(context.Background(), 0, 10, 0)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})
}

func TestDeploymentStatusConversion(t *testing.T) {
	tests := []struct {
		name           string
		domainStatus   deployment.DeploymentStatus
		expectedDBType string
	}{
		{
			name:           "untracked status",
			domainStatus:   deployment.DeploymentStatusUntracked,
			expectedDBType: "untracked",
		},
		{
			name:           "backend_trigger_failed status",
			domainStatus:   deployment.DeploymentStatusBackendTriggerFailed,
			expectedDBType: "backend_trigger_failed",
		},
		{
			name:           "backend_tracking_failed status",
			domainStatus:   deployment.DeploymentStatusBackendTrackingFailed,
			expectedDBType: "backend_tracking_failed",
		},
		{
			name:           "backend_tracking_lost status",
			domainStatus:   deployment.DeploymentStatusBackendTrackingLost,
			expectedDBType: "backend_tracking_lost",
		},
		{
			name:           "running status",
			domainStatus:   deployment.DeploymentStatusRunning,
			expectedDBType: "running",
		},
		{
			name:           "success status",
			domainStatus:   deployment.DeploymentStatusSuccess,
			expectedDBType: "success",
		},
		{
			name:           "failed status",
			domainStatus:   deployment.DeploymentStatusFailed,
			expectedDBType: "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test domain to DB conversion
			dbStatus := deploymentStatusToDB(tt.domainStatus)
			assert.Equal(t, tt.expectedDBType, string(dbStatus))

			// Test round-trip conversion
			backToDomain := deploymentStatusFromDB(dbStatus)
			assert.Equal(t, tt.domainStatus, backToDomain)
		})
	}
}
