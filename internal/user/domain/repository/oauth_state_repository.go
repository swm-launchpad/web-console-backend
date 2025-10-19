package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
)

// OAuthStateRepository defines the interface for OAuth state data access
type OAuthStateRepository interface {
	// Create stores a new OAuth state
	Create(ctx context.Context, state *model.OAuthState) error

	// FindByState retrieves an OAuth state by its token
	FindByState(ctx context.Context, state string) (*model.OAuthState, error)

	// MarkAsConsumed marks a state as consumed (one-time use)
	MarkAsConsumed(ctx context.Context, state string, installationID int64) error

	// DeleteExpired removes expired states (cleanup)
	// Returns the number of deleted rows and any error
	DeleteExpired(ctx context.Context) (int64, error)
}
