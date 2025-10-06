package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
)

// DeploymentLockRepository defines the interface for deployment lock data persistence
// This follows the repository pattern and is part of the domain layer
type DeploymentLockRepository interface {
	// AcquireLock attempts to acquire a deployment lock for a project
	// Returns the lock with monotonically increasing token if successful
	// Returns ErrLockAlreadyAcquired if a non-expired lock already exists
	AcquireLock(ctx context.Context, lock *deployment.DeploymentLock) error

	// RenewLock extends the expiration time of an existing lock
	// Succeeds if request token >= DB token and lock is not expired
	// Ignored (no-op) if request token < DB token (stale request)
	RenewLock(ctx context.Context, projectID uint, token uint64, lock *deployment.DeploymentLock) error

	// ReleaseLock expires a deployment lock immediately
	// Succeeds if request token >= DB token
	// Ignored (no-op) if request token < DB token (stale request)
	ReleaseLock(ctx context.Context, projectID uint, token uint64) error
}
