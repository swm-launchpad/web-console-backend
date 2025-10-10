package repository

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
)

// DeploymentLockRepository defines the interface for deployment lock data persistence
// This follows the repository pattern and is part of the domain layer
type DeploymentLockRepository interface {
	// AcquireLock attempts to acquire a deployment lock for a project
	// Returns a new lock with the assigned token if successful
	// Returns ErrLockAlreadyAcquired if a non-expired lock already exists
	AcquireLock(ctx context.Context, projectID uint, expiresAt time.Time) (*deployment.DeploymentLock, error)

	// RenewLock extends the expiration time of an existing lock
	// The input lock must contain projectID, token, and the new expiresAt
	// Returns the renewed lock if successful
	// Returns ErrLockExpired if the lock has already expired
	// Returns ErrInvalidLockToken if the token doesn't match (stale request)
	RenewLock(ctx context.Context, lock *deployment.DeploymentLock) (*deployment.DeploymentLock, error)

	// ReleaseLock expires a deployment lock immediately
	// The input lock must contain projectID and token
	// This operation is idempotent - releasing an already expired lock is a no-op
	ReleaseLock(ctx context.Context, lock *deployment.DeploymentLock) error
}
