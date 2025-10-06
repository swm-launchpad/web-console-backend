package deployment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReconstructDeploymentLock(t *testing.T) {
	t.Run("DeploymentLock 재구성", func(t *testing.T) {
		projectID := uint(1)
		token := uint64(123456)
		expiresAt := time.Now().Add(5 * time.Minute)

		lock := ReconstructDeploymentLock(projectID, token, expiresAt)

		assert.Equal(t, projectID, lock.ProjectID())
		assert.Equal(t, token, lock.Token())
		assert.Equal(t, expiresAt, lock.ExpiresAt())
	})
}

func TestDeploymentLock_IsExpired(t *testing.T) {
	t.Run("만료되지 않은 락", func(t *testing.T) {
		lock := ReconstructDeploymentLock(1, 123456, time.Now().Add(5*time.Minute))
		assert.False(t, lock.IsExpired())
	})

	t.Run("만료된 락", func(t *testing.T) {
		expiredLock := ReconstructDeploymentLock(
			1, 123456,
			time.Now().Add(-1*time.Second),
		)

		assert.True(t, expiredLock.IsExpired())
	})
}
