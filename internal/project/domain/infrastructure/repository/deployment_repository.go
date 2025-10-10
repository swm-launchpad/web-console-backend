package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
)

// DeploymentRepository defines the interface for deployment persistence
type DeploymentRepository interface {
	// Create creates a new deployment record
	// The deployment ID will be set after successful creation
	Create(ctx context.Context, d *deployment.Deployment) error

	// Save updates an existing deployment record
	// Used to update status, summary, timestamps, etc.
	Save(ctx context.Context, d *deployment.Deployment) error

	// FindByID finds a deployment by its ID
	// Returns ErrDeploymentNotFound if the deployment does not exist
	FindByID(ctx context.Context, deploymentID uint) (*deployment.Deployment, error)

	// FindLatestByProjectID finds the most recent deployment for a project
	// Returns ErrDeploymentNotFound if no deployments exist for the project
	FindLatestByProjectID(ctx context.Context, projectID uint) (*deployment.Deployment, error)

	// FindByProjectID finds all deployments for a project with pagination
	// Returns an empty slice if no deployments exist
	// Deployments are ordered by created_at DESC (newest first)
	FindByProjectID(ctx context.Context, projectID uint, limit, offset int) ([]*deployment.Deployment, error)
}
