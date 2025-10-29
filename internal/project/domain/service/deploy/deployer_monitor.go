package deploy

import (
	"context"
	"time"

	"go.uber.org/zap"
)

func (s *deployService) monitorDeployment(ctx context.Context, projectID uint, deploymentID uint) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error(ctx, "PANIC in monitorDeployment",
				zap.Uint("project_id", projectID),
				zap.Uint("deployment_id", deploymentID),
				zap.Any("panic", r),
			)
		}
	}()

	// Polling interval
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d, _ := s.refreshDeploymentStatus(ctx, uint64(deploymentID))

			if d != nil && d.IsCompleted() {
				// Deployment completed - exit monitoring
				return
			}
		}
	}
}

// deployProjectInternal deploys a project using the provided container configuration.
// This method performs the deployment and monitors it directly (no background goroutine).
//
// The method is used by BuildAndDeployProject to deploy after builds complete.
// It accepts container configuration as a parameter to ensure consistency with the
// built artifacts - the configuration represents the state when builds were initiated.
//
// Process:
//  1. Atomically set project status to 'deploying' and create Deployment record
//  2. Gather deployment configuration (volumes, project metadata)
//  3. Trigger deployment via Tekton API
//  4. Monitor deployment directly with 10-second polling (30-minute timeout)
//  5. Update deployment and project status when terminal state is reached
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - projectID: The unique identifier of the project
//   - containerConfig: Container configuration snapshot captured before builds
//
// Returns:
//   - error: Returns error if deployment cannot be initiated or monitoring fails
//
// Note: This method does NOT spawn a goroutine - it performs monitoring synchronously.
// This allows the caller (buildAndDeployInBackground) to maintain a single goroutine
// for the entire build+deploy flow.
