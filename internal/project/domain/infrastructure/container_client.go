// Package infrastructure defines interfaces for external infrastructure dependencies.
// These interfaces abstract away external bounded contexts and services.
package infrastructure

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// ContainerClient defines the interface for communicating with the Container bounded context.
// This interface follows the Dependency Inversion Principle by defining the contract
// in the domain layer, allowing infrastructure implementations to depend on the domain.
//
// The Container bounded context is responsible for managing container configurations,
// environment variables, secrets, and volume mounts for deployments.
type ContainerClient interface {
	// GetAllContainerInfo retrieves all container deployment information for a project.
	// This includes containers, environment variables, secrets, volume mounts, and ConfigMaps.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project
	//
	// Returns:
	//   - *dto.ContainerDeploymentInfo: Complete deployment information if found
	//   - error: An error if the operation fails or the project is not found
	//
	// Example usage:
	//   deployInfo, err := client.GetAllContainerInfo(ctx, 123)
	//   if err != nil {
	//       return err
	//   }
	//   for _, container := range deployInfo.Containers {
	//       fmt.Printf("Container: %s, Image: %s:%s\n",
	//           container.Name, container.ImageName, container.ImageTag)
	//   }
	GetAllContainerInfo(ctx context.Context, projectID uint) (*dto.ContainerDeploymentInfo, error)
}
