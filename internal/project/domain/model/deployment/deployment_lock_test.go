package deployment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewDeploymentLock(t *testing.T) {
	t.Run("정상적인 DeploymentLock 생성", func(t *testing.T) {
		projectID := uint(1)
		ttl := 5 * time.Minute

		lock, err := NewDeploymentLock(projectID, ttl)

		require.NoError(t, err)
		assert.NotNil(t, lock)
		assert.Equal(t, projectID, lock.ProjectID())
		assert.Equal(t, uint64(0), lock.Token(), "Token should be 0 before DB assignment")
		assert.False(t, lock.IsExpired())
		assert.WithinDuration(t, time.Now().Add(ttl), lock.ExpiresAt(), time.Second)
	})

	t.Run("projectID가 0인 경우 에러", func(t *testing.T) {
		lock, err := NewDeploymentLock(0, 5*time.Minute)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
		assert.Nil(t, lock)
	})

	t.Run("TTL이 0 이하인 경우 에러", func(t *testing.T) {
		lock, err := NewDeploymentLock(1, 0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidLockTTL, err)
		assert.Nil(t, lock)
	})

	t.Run("음수 TTL인 경우 에러", func(t *testing.T) {
		lock, err := NewDeploymentLock(1, -5*time.Minute)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidLockTTL, err)
		assert.Nil(t, lock)
	})
}

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
		lock, _ := NewDeploymentLock(1, 5*time.Minute)
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

func TestDeploymentLock_Renew(t *testing.T) {
	t.Run("정상적인 TTL 갱신", func(t *testing.T) {
		lock, _ := NewDeploymentLock(1, 5*time.Minute)
		originalExpiresAt := lock.ExpiresAt()

		time.Sleep(10 * time.Millisecond)
		err := lock.Renew(10 * time.Minute)

		require.NoError(t, err)
		assert.True(t, lock.ExpiresAt().After(originalExpiresAt))
		assert.WithinDuration(t, time.Now().Add(10*time.Minute), lock.ExpiresAt(), time.Second)
	})

	t.Run("음수 TTL로 갱신 시도", func(t *testing.T) {
		lock, _ := NewDeploymentLock(1, 5*time.Minute)

		err := lock.Renew(-1 * time.Minute)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidLockTTL, err)
	})

	t.Run("0 TTL로 갱신 시도", func(t *testing.T) {
		lock, _ := NewDeploymentLock(1, 5*time.Minute)

		err := lock.Renew(0)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrInvalidLockTTL, err)
	})

	t.Run("만료된 락 갱신 시도", func(t *testing.T) {
		expiredLock := ReconstructDeploymentLock(
			1, 123456,
			time.Now().Add(-1*time.Second),
		)

		err := expiredLock.Renew(5 * time.Minute)

		assert.Error(t, err)
		assert.Equal(t, projecterrors.ErrLockExpired, err)
	})
}

func TestDeploymentLock_SetToken(t *testing.T) {
	t.Run("Fencing Token 설정", func(t *testing.T) {
		lock, _ := NewDeploymentLock(1, 5*time.Minute)
		assert.Zero(t, lock.Token(), "Token should be 0 before DB assignment")

		fencingToken := uint64(12345)
		lock.SetToken(fencingToken)

		assert.Equal(t, fencingToken, lock.Token())
	})
}
