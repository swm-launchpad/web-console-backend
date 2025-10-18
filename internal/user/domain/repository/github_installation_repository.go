package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
)

// GitHubInstallationRepository defines the interface for GitHub installation data access
type GitHubInstallationRepository interface {
	// Create creates a new GitHub installation record
	Create(ctx context.Context, installation *model.GitHubInstallation) error

	// FindByInstallationID retrieves a GitHub installation by its installation ID
	FindByInstallationID(ctx context.Context, installationID int64) (*model.GitHubInstallation, error)

	// FindByUserID retrieves GitHub installations for a specific user
	FindByUserID(ctx context.Context, userID uint) ([]*model.GitHubInstallation, error)

	// Update updates an existing GitHub installation
	Update(ctx context.Context, installation *model.GitHubInstallation) error

	// MarkAsRevoked marks a GitHub installation as revoked
	MarkAsRevoked(ctx context.Context, installationID int64) error

	// Delete soft deletes a GitHub installation
	Delete(ctx context.Context, installationID int64) error

	// ExistsByInstallationID checks if an installation exists
	ExistsByInstallationID(ctx context.Context, installationID int64) (bool, error)
}
