package deployment

import (
	"time"
)

// DeploymentLock represents a distributed lock for preventing concurrent deployments
// It uses fencing token (monotonically increasing per project) to prevent ABA problems and TTL-based expiration
// The token increases by 1 each time a new lock is acquired for the same project
//
// Locks are ONLY created through the DeploymentLockRepository:
// - AcquireLock: Acquires a new lock or reacquires an expired lock
// - RenewLock: Extends expiration of an existing lock
// - ReleaseLock: Immediately expires a lock
type DeploymentLock struct {
	projectID uint
	token     uint64 // Fencing token: monotonically increasing value per project
	expiresAt time.Time
}

// ReconstructDeploymentLock reconstructs a deployment lock from persistence
// This is used by the repository when loading from the database
func ReconstructDeploymentLock(
	projectID uint,
	token uint64,
	expiresAt time.Time,
) *DeploymentLock {
	return &DeploymentLock{
		projectID: projectID,
		token:     token,
		expiresAt: expiresAt,
	}
}

// ProjectID returns the project ID
func (l *DeploymentLock) ProjectID() uint {
	return l.projectID
}

// Token returns the lock token
func (l *DeploymentLock) Token() uint64 {
	return l.token
}

// ExpiresAt returns the expiration time
func (l *DeploymentLock) ExpiresAt() time.Time {
	return l.expiresAt
}

// IsExpired checks if the lock has expired
func (l *DeploymentLock) IsExpired() bool {
	return time.Now().After(l.expiresAt)
}
