package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
)

type deploymentLockRepository struct {
	db      sqlc.DBTX
	queries *sqlc.Queries
}

// NewDeploymentLockRepository creates a new deployment lock repository instance
func NewDeploymentLockRepository(db sqlc.DBTX) repository.DeploymentLockRepository {
	return &deploymentLockRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// AcquireLock attempts to acquire a deployment lock for a project
// This operation is atomic and race-condition-free using INSERT ... ON DUPLICATE KEY UPDATE
// Returns a new lock with the assigned token if successful
// Returns ErrLockAlreadyAcquired if a non-expired lock already exists
func (r *deploymentLockRepository) AcquireLock(ctx context.Context, projectID uint, expiresAt time.Time) (*deployment.DeploymentLock, error) {
	if projectID == 0 {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	// Atomically try to insert new lock or update expired lock
	result, err := qtx.AcquireOrUpdateLock(ctx, sqlc.AcquireOrUpdateLockParams{
		ProjectID: uint32(projectID),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Check what happened:
	// - RowsAffected == 1: INSERT succeeded (new lock, token=1)
	// - RowsAffected == 2: UPDATE succeeded (expired lock renewed, token++)
	// - RowsAffected == 0: UPDATE failed (active lock exists)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	if rowsAffected == 0 {
		// Active lock already exists for this project
		return nil, projecterrors.ErrLockAlreadyAcquired
	}

	// Lock acquired successfully - retrieve the current token
	sqlcLock, err := qtx.GetDeploymentLock(ctx, uint32(projectID))
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Create new lock with the actual token from DB
	acquiredLock := deployment.ReconstructDeploymentLock(
		projectID,
		sqlcLock.Token,
		sqlcLock.ExpiresAt,
	)

	return acquiredLock, nil
}

// RenewLock extends the expiration time of an existing lock
// The input lock must contain projectID, token, and the new expiresAt
// Returns the renewed lock if successful
// Returns ErrLockExpired if the lock has already expired
// Returns ErrInvalidLockToken if the token doesn't match (stale request)
func (r *deploymentLockRepository) RenewLock(ctx context.Context, lock *deployment.DeploymentLock) (*deployment.DeploymentLock, error) {
	if lock == nil {
		return nil, projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	// Try to renew the lock - only succeeds if token matches and lock is not expired
	result, err := qtx.RenewDeploymentLock(ctx, sqlc.RenewDeploymentLockParams{
		ExpiresAt: lock.ExpiresAt(),
		ProjectID: uint32(lock.ProjectID()),
		Token:     lock.Token(),
	})
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	// Check if any row was updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, projecterrors.ErrDatabaseOperation
	}

	if rowsAffected == 0 {
		// Either the lock is expired or the token doesn't match
		// Check which case it is by fetching the current lock state
		sqlcLock, err := qtx.GetDeploymentLock(ctx, uint32(lock.ProjectID()))
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Lock doesn't exist at all
				return nil, projecterrors.ErrLockNotFound
			}
			return nil, projecterrors.ErrDatabaseOperation
		}

		// Check if it's a token mismatch (stale request)
		if sqlcLock.Token != lock.Token() {
			return nil, projecterrors.ErrInvalidLockToken
		}

		// Token matches but update failed - must be expired
		return nil, projecterrors.ErrLockExpired
	}

	// Return the renewed lock (same as input, confirmed successful)
	return lock, nil
}

// ReleaseLock expires a deployment lock immediately
// The input lock must contain projectID and token
// This operation is idempotent - releasing an already expired lock is a no-op
func (r *deploymentLockRepository) ReleaseLock(ctx context.Context, lock *deployment.DeploymentLock) error {
	if lock == nil {
		return projecterrors.ErrInvalidProjectData
	}

	qtx := r.queriesWithContext(ctx)

	// Set the lock's expiration to NOW - only if token matches
	_, err := qtx.ReleaseDeploymentLock(ctx, sqlc.ReleaseDeploymentLockParams{
		ProjectID: uint32(lock.ProjectID()),
		Token:     lock.Token(),
	})
	if err != nil {
		return projecterrors.ErrDatabaseOperation
	}

	// Note: We don't check RowsAffected here for idempotency
	// If the lock doesn't exist or token doesn't match, we still consider it released
	return nil
}

// queriesWithContext returns queries bound to transaction if available in context
func (r *deploymentLockRepository) queriesWithContext(ctx context.Context) *sqlc.Queries {
	if tx, ok := db.GetTx(ctx); ok && tx != nil {
		return r.queries.WithTx(tx)
	}
	return r.queries
}
