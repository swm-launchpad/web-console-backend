package integration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	projectinfra "github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

func TestDeploymentLockRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	testDB := helper.SetupTestDB(t)
	defer testDB.Cleanup()

	// Create repository
	lockRepo := projectinfra.NewDeploymentLockRepository(testDB.DB)
	ctx := context.Background()

	t.Run("AcquireLock - New lock with token=1", func(t *testing.T) {
		// Given
		pid := createTestProject(t, testDB)
		expiresAt := time.Now().Add(5 * time.Minute)

		// When
		acquiredLock, err := lockRepo.AcquireLock(ctx, pid, expiresAt)

		// Then
		require.NoError(t, err)
		assert.Equal(t, uint64(1), acquiredLock.Token(), "First acquisition should set token=1")
		assert.Equal(t, pid, acquiredLock.ProjectID())
		assert.False(t, acquiredLock.IsExpired())
	})

	t.Run("AcquireLock - Fail when active lock exists", func(t *testing.T) {
		// Given - First lock acquisition
		pid := createTestProject(t, testDB)
		expiresAt := time.Now().Add(5 * time.Minute)
		_, err := lockRepo.AcquireLock(ctx, pid, expiresAt)
		require.NoError(t, err)

		// When - Try to acquire again with active lock
		_, err = lockRepo.AcquireLock(ctx, pid, expiresAt)

		// Then
		assert.ErrorIs(t, err, projecterrors.ErrLockAlreadyAcquired)
	})

	t.Run("AcquireLock - Reacquire expired lock with incremented token", func(t *testing.T) {
		// Given - Create an expired lock
		pid := createTestProject(t, testDB)
		expiresAt1 := time.Now().Add(2 * time.Second)
		acquiredLock1, err := lockRepo.AcquireLock(ctx, pid, expiresAt1)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), acquiredLock1.Token())

		// Wait for expiration (generous buffer for clock skew between Go and MySQL)
		time.Sleep(2500 * time.Millisecond)

		// When - Acquire expired lock
		expiresAt2 := time.Now().Add(5 * time.Minute)
		acquiredLock2, err := lockRepo.AcquireLock(ctx, pid, expiresAt2)

		// Then
		require.NoError(t, err)
		assert.Equal(t, uint64(2), acquiredLock2.Token(), "Token should increment to 2")
		assert.False(t, acquiredLock2.IsExpired())
	})

	t.Run("AcquireLock - Race condition test (10 concurrent attempts)", func(t *testing.T) {
		// Given
		pid := createTestProject(t, testDB)
		const numGoroutines = 10
		var successCount atomic.Int32
		var alreadyAcquiredCount atomic.Int32
		var wg sync.WaitGroup

		// When - 10 goroutines try to acquire lock simultaneously
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				expiresAt := time.Now().Add(5 * time.Minute)
				_, err := lockRepo.AcquireLock(ctx, pid, expiresAt)
				if err == nil {
					successCount.Add(1)
				} else if errors.Is(err, projecterrors.ErrLockAlreadyAcquired) {
					alreadyAcquiredCount.Add(1)
				}
			}()
		}
		wg.Wait()

		// Then - Only one should succeed
		assert.Equal(t, int32(1), successCount.Load(), "Only one goroutine should acquire lock")
		assert.Equal(t, int32(numGoroutines-1), alreadyAcquiredCount.Load(), "Others should get ErrLockAlreadyAcquired")
	})

	t.Run("RenewLock - Successfully renew with valid token", func(t *testing.T) {
		// Given - Acquire lock first
		pid := createTestProject(t, testDB)
		expiresAt1 := time.Now().Add(2 * time.Second)
		acquiredLock, err := lockRepo.AcquireLock(ctx, pid, expiresAt1)
		require.NoError(t, err)
		originalToken := acquiredLock.Token()
		originalExpiry := acquiredLock.ExpiresAt()

		time.Sleep(100 * time.Millisecond)

		// When - Renew the lock with new expiry
		newExpiry := time.Now().Add(10 * time.Minute)
		renewLock := deployment.ReconstructDeploymentLock(pid, originalToken, newExpiry)
		renewedLock, err := lockRepo.RenewLock(ctx, renewLock)

		// Then
		require.NoError(t, err)
		// Verify expiry was extended (should be later than original)
		assert.True(t, renewedLock.ExpiresAt().After(originalExpiry))
	})

	t.Run("RenewLock - Fail with expired lock", func(t *testing.T) {
		// Given - Create and let lock expire
		pid := createTestProject(t, testDB)
		expiresAt := time.Now().Add(2 * time.Second)
		acquiredLock, err := lockRepo.AcquireLock(ctx, pid, expiresAt)
		require.NoError(t, err)
		token := acquiredLock.Token()

		// Wait for expiration (add extra time to account for MySQL NOW() vs Go time.Now() skew)
		time.Sleep(2500 * time.Millisecond)

		// When - Try to renew expired lock
		newExpiry := time.Now().Add(5 * time.Minute)
		renewLock := deployment.ReconstructDeploymentLock(pid, token, newExpiry)
		_, err = lockRepo.RenewLock(ctx, renewLock)

		// Then
		assert.ErrorIs(t, err, projecterrors.ErrLockExpired)
	})

	t.Run("RenewLock - Fail with stale token (fencing token protection)", func(t *testing.T) {
		// Given - Acquire lock with token=1
		pid := createTestProject(t, testDB)
		expiresAt1 := time.Now().Add(2 * time.Second)
		acquiredLock1, err := lockRepo.AcquireLock(ctx, pid, expiresAt1)
		require.NoError(t, err)
		staleToken := acquiredLock1.Token() // token=1

		// Wait for expiration and reacquire (token becomes 2)
		time.Sleep(2500 * time.Millisecond)
		expiresAt2 := time.Now().Add(5 * time.Minute)
		acquiredLock2, err := lockRepo.AcquireLock(ctx, pid, expiresAt2)
		require.NoError(t, err)
		assert.Equal(t, staleToken+1, acquiredLock2.Token(), "Token should have incremented")

		// When - Try to renew with old token (token=1)
		newExpiry := time.Now().Add(5 * time.Minute)
		renewLock := deployment.ReconstructDeploymentLock(pid, staleToken, newExpiry)
		_, err = lockRepo.RenewLock(ctx, renewLock)

		// Then - Should fail due to token mismatch
		assert.ErrorIs(t, err, projecterrors.ErrInvalidLockToken)
	})

	t.Run("RenewLock - Fail with non-existent lock", func(t *testing.T) {
		// Given - No lock exists
		pid := createTestProject(t, testDB)

		// When - Try to renew non-existent lock
		newExpiry := time.Now().Add(5 * time.Minute)
		renewLock := deployment.ReconstructDeploymentLock(pid, 99, newExpiry)
		_, err := lockRepo.RenewLock(ctx, renewLock)

		// Then
		assert.ErrorIs(t, err, projecterrors.ErrLockNotFound)
	})

	t.Run("ReleaseLock - Successfully release with valid token", func(t *testing.T) {
		// Given - Acquire lock
		pid := createTestProject(t, testDB)
		expiresAt := time.Now().Add(5 * time.Minute)
		acquiredLock, err := lockRepo.AcquireLock(ctx, pid, expiresAt)
		require.NoError(t, err)

		// When - Release lock
		err = lockRepo.ReleaseLock(ctx, acquiredLock)

		// Then
		require.NoError(t, err)

		// Verify lock can be acquired again (it was released)
		newExpiresAt := time.Now().Add(5 * time.Minute)
		_, err = lockRepo.AcquireLock(ctx, pid, newExpiresAt)
		assert.NoError(t, err, "Should be able to acquire after release")
	})

	t.Run("ReleaseLock - Idempotent with stale token", func(t *testing.T) {
		// Given - Acquire and release lock
		pid := createTestProject(t, testDB)
		expiresAt := time.Now().Add(5 * time.Minute)
		acquiredLock, err := lockRepo.AcquireLock(ctx, pid, expiresAt)
		require.NoError(t, err)

		err = lockRepo.ReleaseLock(ctx, acquiredLock)
		require.NoError(t, err)

		// When - Try to release again with same lock
		err = lockRepo.ReleaseLock(ctx, acquiredLock)

		// Then - Should succeed (idempotent)
		assert.NoError(t, err)
	})

	t.Run("ReleaseLock - Idempotent with non-existent lock", func(t *testing.T) {
		// Given - No lock exists
		pid := createTestProject(t, testDB)

		// When - Try to release non-existent lock
		nonExistentLock := deployment.ReconstructDeploymentLock(pid, 999, time.Now())
		err := lockRepo.ReleaseLock(ctx, nonExistentLock)

		// Then - Should succeed (idempotent)
		assert.NoError(t, err)
	})

	t.Run("Token Monotonicity - Tokens increment across acquisitions", func(t *testing.T) {
		// Given
		pid := createTestProject(t, testDB)
		var tokens []uint64

		// When - Acquire, expire, reacquire multiple times
		for i := 0; i < 5; i++ {
			expiresAt := time.Now().Add(2 * time.Second)
			acquiredLock, err := lockRepo.AcquireLock(ctx, pid, expiresAt)
			require.NoError(t, err)
			tokens = append(tokens, acquiredLock.Token())

			// Wait for expiration before next acquisition (generous buffer for clock skew)
			time.Sleep(2500 * time.Millisecond)
		}

		// Then - Tokens should be monotonically increasing
		for i := 1; i < len(tokens); i++ {
			assert.Greater(t, tokens[i], tokens[i-1], "Token %d should be greater than token %d", i, i-1)
		}
		assert.Equal(t, uint64(1), tokens[0], "First token should be 1")
		assert.Equal(t, uint64(5), tokens[4], "Fifth token should be 5")
	})

	t.Run("Concurrent Renewals - At least one renewal succeeds", func(t *testing.T) {
		// Given - Acquire lock
		pid := createTestProject(t, testDB)
		expiresAt := time.Now().Add(5 * time.Minute)
		acquiredLock, err := lockRepo.AcquireLock(ctx, pid, expiresAt)
		require.NoError(t, err)
		token := acquiredLock.Token()

		// When - Multiple goroutines try to renew with same token simultaneously
		const numGoroutines = 5
		var successCount atomic.Int32
		var wg sync.WaitGroup
		var startBarrier sync.WaitGroup
		startBarrier.Add(1)

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()

				// Wait for all goroutines to be ready
				startBarrier.Wait()

				newExpiry := time.Now().Add(10 * time.Minute)
				renewLock := deployment.ReconstructDeploymentLock(pid, token, newExpiry)
				_, err := lockRepo.RenewLock(ctx, renewLock)
				if err == nil {
					successCount.Add(1)
				}
				// Note: Some may get ErrLockExpired or ErrInvalidLockToken if they check after another goroutine updated
			}()
		}

		// Release all goroutines at once
		startBarrier.Done()
		wg.Wait()

		// Then - At least one renewal should succeed
		// Due to race conditions in concurrent UPDATEs, not all may succeed, but at least one should
		assert.GreaterOrEqual(t, successCount.Load(), int32(1), "At least one renewal should succeed")
	})

	t.Run("Invalid parameters return error", func(t *testing.T) {
		_, err := lockRepo.AcquireLock(ctx, 0, time.Now().Add(5*time.Minute))
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)

		_, err = lockRepo.RenewLock(ctx, nil)
		assert.ErrorIs(t, err, projecterrors.ErrInvalidProjectData)
	})
}

// createTestProject creates a test project and returns its ID
func createTestProject(t *testing.T, testDB *helper.TestDB) uint {
	t.Helper()

	query := `
		INSERT INTO PROJECTS (name, slug, status, cpu_limit, memory_limit, disk_limit, traffic_limit, created_at)
		VALUES (?, ?, 'active', 1000, 2048, 2048, 1048576, NOW())
	`

	// Use nanosecond timestamp for unique slugs
	slug := fmt.Sprintf("test-project-%d", time.Now().UnixNano())
	result, err := testDB.DB.Exec(query, "Test Project", slug)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)

	return uint(id)
}
