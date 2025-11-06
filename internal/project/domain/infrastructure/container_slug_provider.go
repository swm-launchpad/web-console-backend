package infrastructure

import "context"

// ContainerSlugProvider provides container slug information from the Container bounded context.
// This interface allows Project BC to query container information without direct dependency
// on Container BC domain layer, following clean architecture principles.
type ContainerSlugProvider interface {
	// GetContainerSlugsByProjectID retrieves all container slugs (including soft-deleted)
	// for a given project. Used during project cleanup to remove all container resources.
	GetContainerSlugsByProjectID(ctx context.Context, projectID uint) ([]string, error)
}
