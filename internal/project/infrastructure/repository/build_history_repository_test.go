package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
)

// TestBuildHistoryRepository_ErrorHandling tests error handling with invalid inputs
// Note: All functional tests are in integration tests with real database
func TestBuildHistoryRepository_ErrorHandling(t *testing.T) {
	testLogger := logger.NewForTest()

	t.Run("Create with nil build history", func(t *testing.T) {
		repo := NewBuildHistoryRepository(nil, testLogger)
		err := repo.Create(context.Background(), nil)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("Save with nil build history", func(t *testing.T) {
		repo := NewBuildHistoryRepository(nil, testLogger)
		err := repo.Save(context.Background(), nil)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("Save with zero build history ID", func(t *testing.T) {
		repo := NewBuildHistoryRepository(nil, testLogger)
		b := build_history.NewBuildHistory(123)
		err := repo.Save(context.Background(), b)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("FindByID with zero ID", func(t *testing.T) {
		repo := NewBuildHistoryRepository(nil, testLogger)
		_, err := repo.FindByID(context.Background(), 0)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("FindLatestByContainerID with zero ID", func(t *testing.T) {
		repo := NewBuildHistoryRepository(nil, testLogger)
		_, err := repo.FindLatestByContainerID(context.Background(), 0)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("FindByContainerID with zero ID", func(t *testing.T) {
		repo := NewBuildHistoryRepository(nil, testLogger)
		_, err := repo.FindByContainerID(context.Background(), 0, 10, 0)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("FindActiveByContainerID with zero ID", func(t *testing.T) {
		repo := NewBuildHistoryRepository(nil, testLogger)
		_, err := repo.FindActiveByContainerID(context.Background(), 0)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("FindByTektonPipelineRunName with empty string", func(t *testing.T) {
		repo := NewBuildHistoryRepository(nil, testLogger)
		_, err := repo.FindByTektonPipelineRunName(context.Background(), "")
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})
}

func TestBuildHistoryStatusConversion(t *testing.T) {
	tests := []struct {
		name           string
		domainStatus   build_history.BuildHistoryStatus
		expectedDBType string
	}{
		{
			name:           "untracked status",
			domainStatus:   build_history.BuildHistoryStatusUntracked,
			expectedDBType: "untracked",
		},
		{
			name:           "backend_trigger_failed status",
			domainStatus:   build_history.BuildHistoryStatusBackendTriggerFailed,
			expectedDBType: "backend_trigger_failed",
		},
		{
			name:           "backend_tracking_failed status",
			domainStatus:   build_history.BuildHistoryStatusBackendTrackingFailed,
			expectedDBType: "backend_tracking_failed",
		},
		{
			name:           "backend_tracking_lost status",
			domainStatus:   build_history.BuildHistoryStatusBackendTrackingLost,
			expectedDBType: "backend_tracking_lost",
		},
		{
			name:           "running status",
			domainStatus:   build_history.BuildHistoryStatusRunning,
			expectedDBType: "running",
		},
		{
			name:           "success status",
			domainStatus:   build_history.BuildHistoryStatusSuccess,
			expectedDBType: "success",
		},
		{
			name:           "failed status",
			domainStatus:   build_history.BuildHistoryStatusFailed,
			expectedDBType: "failed",
		},
		{
			name:           "cancelled status",
			domainStatus:   build_history.BuildHistoryStatusCancelled,
			expectedDBType: "cancelled",
		},
		{
			name:           "skipped status",
			domainStatus:   build_history.BuildHistoryStatusSkipped,
			expectedDBType: "skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test domain to DB conversion
			dbStatus := buildHistoryStatusToDB(tt.domainStatus)
			assert.Equal(t, tt.expectedDBType, string(dbStatus))

			// Test round-trip conversion
			backToDomain := buildHistoryStatusFromDB(dbStatus)
			assert.Equal(t, tt.domainStatus, backToDomain)
		})
	}
}
