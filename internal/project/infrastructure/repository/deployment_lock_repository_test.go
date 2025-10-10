package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// TestDeploymentLockRepository_ErrorHandling tests error handling with invalid inputs
// Note: All functional tests are in integration tests with real database
func TestDeploymentLockRepository_ErrorHandling(t *testing.T) {
	t.Run("AcquireLock with invalid projectID", func(t *testing.T) {
		repo := NewDeploymentLockRepository(nil)
		_, err := repo.AcquireLock(context.Background(), 0, time.Now().Add(5*time.Minute))
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("RenewLock with nil lock", func(t *testing.T) {
		repo := NewDeploymentLockRepository(nil)
		_, err := repo.RenewLock(context.Background(), nil)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})

	t.Run("ReleaseLock with nil lock", func(t *testing.T) {
		repo := NewDeploymentLockRepository(nil)
		err := repo.ReleaseLock(context.Background(), nil)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})
}
