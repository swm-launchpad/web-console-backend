package build

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// Orchestrator defines the interface for orchestrating multiple container builds.
// This service coordinates parallel builds for all containers in a project,
// managing goroutines, collecting results, and handling errors.
type Orchestrator interface {
	// BuildAndWait executes builds for all containers in parallel and waits for completion.
	// This method:
	//  1. Creates a BUILD_HISTORY record for each container (status=untracked)
	//  2. Spawns a goroutine for each container build (calls BuildService.BuildContainer)
	//  3. Uses WaitGroup to wait for all builds to complete
	//  4. Collects results via channels
	//  5. Returns all build results and any errors encountered
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadline control
	//   - projectID: The unique identifier of the project
	//   - containers: List of container configurations to build
	//
	// Returns:
	//   - []*BuildResult: Results for each build (in same order as input containers)
	//   - error: Returns error if orchestration setup fails
	//
	// Note: Even if some builds fail, all results are returned. Check each BuildResult.Status
	// and BuildResult.ErrorMessage to determine individual build outcomes.
	BuildAndWait(
		ctx context.Context,
		projectID uint,
		containers []*dto.BuildContainerInfo,
	) ([]*BuildResult, error)
}
