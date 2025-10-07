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
//
// Note: This interface returns only container-level configuration. Project metadata
// (project_id, service_name, namespace, stable_window) should come from the Project
// bounded context.
type ContainerClient interface {
	// GetContainerConfig retrieves container-specific deployment configuration for a project.
	// This includes containers, environment variables, secrets, volume mounts, and ConfigMaps.
	// It does NOT include project metadata like project_id, service_name, namespace, or stable_window.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project
	//
	// Returns:
	//   - *dto.ContainerDeploymentConfig: Container configuration if found
	//   - error: An error if the operation fails or the project is not found
	//
	// Example usage:
	//   config, err := client.GetContainerConfig(ctx, 123)
	//   if err != nil {
	//       return err
	//   }
	//   for _, container := range config.Containers {
	//       fmt.Printf("Container: %s, Image: %s:%s\n",
	//           container.Name, container.ImageName, container.ImageTag)
	//   }
	GetContainerConfig(ctx context.Context, projectID uint) (*dto.ContainerDeploymentConfig, error)
}
