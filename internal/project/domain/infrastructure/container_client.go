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

	// GetContainerBuildConfig retrieves container-specific build configuration for a project.
	// This includes container build information such as template, git configuration,
	// build variables, and other build-related settings.
	// It does NOT include project metadata like project_id, service_name, or namespace.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project
	//
	// Returns:
	//   - *dto.ContainerBuildConfig: Container build configuration if found
	//   - error: An error if the operation fails or the project is not found
	//
	// Example usage:
	//   config, err := client.GetContainerBuildConfig(ctx, 123)
	//   if err != nil {
	//       return err
	//   }
	//   for _, container := range config.Containers {
	//       fmt.Printf("Container: %s, Branch: %s, NeedsBuild: %v\n",
	//           container.Name, container.GitBranch, container.NeedsBuild)
	//   }
	GetContainerBuildConfig(ctx context.Context, projectID uint) (*dto.ContainerBuildConfig, error)

	// GetContainerConfigs retrieves both build and deployment configurations in a single call.
	// This method executes a single database query and transforms the result into both formats,
	// ensuring perfect snapshot consistency and eliminating the risk of configuration divergence
	// between build and deployment phases (P1 Badge fix).
	//
	// This method should be preferred over calling GetContainerBuildConfig and GetContainerConfig
	// separately when you need both configurations, as it:
	// - Reduces database queries by 50% (1 query instead of 2)
	// - Guarantees snapshot consistency (no time gap between queries)
	// - Prevents configuration divergence (e.g., deploying containers that weren't built)
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project
	//
	// Returns:
	//   - *dto.ContainerBuildConfig: Container build configuration if found
	//   - *dto.ContainerDeploymentConfig: Container deployment configuration if found
	//   - error: An error if the operation fails or the project is not found
	//
	// Example usage:
	//   buildConfig, deployConfig, err := client.GetContainerConfigs(ctx, 123)
	//   if err != nil {
	//       return err
	//   }
	//   // Both configs are guaranteed to be from the same database snapshot
	//   fmt.Printf("Build containers: %d, Deploy containers: %d\n",
	//       len(buildConfig.Containers), len(deployConfig.Containers))
	GetContainerConfigs(ctx context.Context, projectID uint) (*dto.ContainerBuildConfig, *dto.ContainerDeploymentConfig, error)
}
