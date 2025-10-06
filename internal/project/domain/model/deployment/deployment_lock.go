package deployment

import (
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// DeploymentLock represents a distributed lock for preventing concurrent deployments
// It uses fencing token (monotonically increasing per project) to prevent ABA problems and TTL-based expiration
// The token increases by 1 each time a new lock is acquired for the same project
type DeploymentLock struct {
	projectID uint
	token     uint64 // Fencing token: monotonically increasing value per project
	expiresAt time.Time
}

// NewDeploymentLock creates a new deployment lock without token (token will be assigned by Repository)
// ttl specifies how long the lock should remain valid
func NewDeploymentLock(projectID uint, ttl time.Duration) (*DeploymentLock, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectID
	}
	if ttl <= 0 {
		return nil, projecterrors.ErrInvalidLockTTL
	}

	now := time.Now()
	return &DeploymentLock{
		projectID: projectID,
		token:     0, // Token will be set by repository after DB insertion
		expiresAt: now.Add(ttl),
	}, nil
}

// ReconstructDeploymentLock reconstructs a deployment lock from persistence
// This is used when loading from the database
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

// Renew extends the lock expiration time
// Returns error if the lock is already expired
func (l *DeploymentLock) Renew(newTTL time.Duration) error {
	if newTTL <= 0 {
		return projecterrors.ErrInvalidLockTTL
	}
	if l.IsExpired() {
		return projecterrors.ErrLockExpired
	}

	l.expiresAt = time.Now().Add(newTTL)
	return nil
}

// SetToken sets the fencing token (called by Repository after calculating next token)
func (l *DeploymentLock) SetToken(token uint64) {
	l.token = token
}
