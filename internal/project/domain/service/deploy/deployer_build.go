package deploy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service/build"
)

func (s *deployService) BuildAndDeployProject(ctx context.Context, projectID uint) error {
	s.logger.Info(ctx, "build and deploy project started",
		zap.Uint("project_id", projectID),
	)

	// Phase 1: Validate project exists first (authorization/ownership check must come before resource checks)
	// This ensures we return ErrProjectNotFound for non-existent/unauthorized projects,
	// not ErrContainerConfigNotFound which would leak information about other users' projects
	_, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to find project",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return err // Returns ErrProjectNotFound if project doesn't exist
	}

	// Phase 2: Check container configuration (only after confirming project exists)
	containerConfig, err := s.containerClient.GetContainerBuildConfig(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to get container build config",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return projecterrors.ErrContainerConfigNotFound
	}

	if len(containerConfig.Containers) == 0 {
		s.logger.Error(ctx, "no containers found for project",
			zap.Uint("project_id", projectID),
		)
		return projecterrors.ErrContainerConfigNotFound
	}

	// Phase 3: Atomically change project status to 'building'
	err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Reload project with FOR UPDATE lock (need to lock for status update)
		proj, err := s.projectRepo.FindByIDForUpdate(txCtx, projectID)
		if err != nil {
			s.logger.Error(ctx, "failed to find project for update",
				zap.Uint("project_id", projectID),
				zap.Error(err),
			)
			return err
		}

		// Check if project is already deploying or building
		if proj.OperationStatus() != value.ProjectOperationStatusNothing {
			s.logger.Error(ctx, "project already in operation",
				zap.Uint("project_id", projectID),
				zap.String("operation_status", string(proj.OperationStatus())),
			)
			return projecterrors.ErrProjectAlreadyDeploying
		}

		// Change project status to building
		if err := proj.StartBuild(); err != nil {
			return err
		}

		if err := s.projectRepo.Save(txCtx, proj); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		s.logger.Error(ctx, "failed to set project status to building",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info(ctx, "project status set to building",
		zap.Uint("project_id", projectID),
	)

	// Phase 3: Start background goroutine for build and deploy
	// Create a detached context that preserves request_id and user_id for logging correlation
	// but doesn't inherit cancellation from the original request
	backgroundCtx := logger.DetachContext(ctx)
	go s.buildAndDeployInBackground(backgroundCtx, projectID)

	s.logger.Info(ctx, "build and deploy project initiated",
		zap.Uint("project_id", projectID),
	)

	return nil
}

// buildAndDeployInBackground executes build and deploy operations in the background
func (s *deployService) buildAndDeployInBackground(ctx context.Context, projectID uint) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error(ctx, "PANIC in buildAndDeployInBackground",
				zap.Uint("project_id", projectID),
				zap.Any("panic", r),
			)
			// Attempt to reset project status on panic
			msg := fmt.Sprintf("Panic during build and deploy: %v", r)
			s.handleBuildError(ctx, projectID, &msg)
		}
	}()

	s.logger.Info(ctx, "background build and deploy started",
		zap.Uint("project_id", projectID),
	)

	// Step 1: Get unified container configuration (single source of truth)
	// This executes a SINGLE database query and returns a unified snapshot containing
	// all information needed for both build and deployment operations.
	// By using a unified config, we eliminate the possibility of snapshot divergence by design
	// (P1 Badge fix improvement: divergence cannot happen, not just detected and handled).
	unifiedConfig, err := s.containerClient.GetUnifiedContainerConfig(ctx, projectID)
	if err != nil {
		s.logger.Error(ctx, "failed to get unified container config in background",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Failed to get container configuration: %v", err)
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	s.logger.Info(ctx, "retrieved unified container configuration",
		zap.Uint("project_id", projectID),
		zap.Int("container_count", len(unifiedConfig.Containers)),
	)

	// Step 1.5: Convert unified config to build format
	// This conversion extracts build-specific fields while maintaining the unified snapshot
	buildConfig := build.ConvertToBuildConfig(unifiedConfig)
	if buildConfig == nil || len(buildConfig.Containers) == 0 {
		s.logger.Error(ctx, "no containers in build config after conversion",
			zap.Uint("project_id", projectID),
		)
		msg := "No containers found for build"
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	// Guard: Verify containers still exist (race condition: container deleted between validation and here)
	if len(buildConfig.Containers) == 0 {
		s.logger.Error(ctx, "no containers found for build after status flip",
			zap.Uint("project_id", projectID),
			zap.String("reason", "containers were deleted between initial validation and background execution"),
		)
		msg := "No containers found for build (deleted after status flip)"
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	// Convert []BuildContainerInfo to []*BuildContainerInfo
	containerPointers := make([]*dto.BuildContainerInfo, len(buildConfig.Containers))
	for i := range buildConfig.Containers {
		containerPointers[i] = &buildConfig.Containers[i]
	}

	// Step 2: Execute builds in parallel using build.Orchestrator
	buildResults, err := s.buildOrchestrator.BuildAndWait(ctx, projectID, containerPointers)
	if err != nil {
		s.logger.Error(ctx, "build orchestration failed",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		msg := fmt.Sprintf("Build orchestration failed: %v", err)
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	// Step 3: Check if all builds succeeded
	if hasFailedBuilds(buildResults) {
		s.logger.Error(ctx, "one or more builds failed",
			zap.Uint("project_id", projectID),
		)
		msg := "One or more container builds failed"
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	s.logger.Info(ctx, "all builds completed successfully",
		zap.Uint("project_id", projectID),
	)

	// Step 4: Update container information after successful builds
	// If any update fails, we must abort to avoid deploying with stale metadata
	for i, result := range buildResults {
		if result == nil {
			s.logger.Error(ctx, "build result is nil, aborting post-build updates",
				zap.Uint("project_id", projectID),
				zap.Int("index", i),
			)
			msg := fmt.Sprintf("Build result is nil for container at index %d", i)
			s.handleBuildError(ctx, projectID, &msg)
			return
		}

		err := s.buildPostProcessor.UpdateContainerAfterBuild(
			ctx,
			containerPointers[i].ContainerID,
			result,
			containerPointers[i],
		)
		if err != nil {
			// Check if container changed during build (parameters drift)
			if errors.Is(err, projecterrors.ErrContainerChangedDuringBuild) {
				s.logger.Error(ctx, "container changed during build, cannot proceed with deployment",
					zap.Uint("project_id", projectID),
					zap.Uint("container_id", containerPointers[i].ContainerID),
					zap.String("reason", "parameters changed mid-build, built image is stale"),
					zap.String("mitigation", "needs_build flag remains true, rebuild required"),
				)
				msg := fmt.Sprintf("Container %d changed during build, deployment aborted. Rebuild required.", containerPointers[i].ContainerID)
				s.handleBuildError(ctx, projectID, &msg)
				return
			}

			// Other update errors
			s.logger.Error(ctx, "failed to update container after build, aborting",
				zap.Uint("project_id", projectID),
				zap.Uint("container_id", containerPointers[i].ContainerID),
				zap.Error(err),
			)
			msg := fmt.Sprintf("Failed to update container %d after build: %v", containerPointers[i].ContainerID, err)
			s.handleBuildError(ctx, projectID, &msg)
			return
		}
	}

	s.logger.Info(ctx, "container updates completed, proceeding to deployment",
		zap.Uint("project_id", projectID),
	)

	// Step 4.5: Convert unified config to deployment format with build results
	// This conversion extracts deployment-specific fields from the unified snapshot
	// and updates image tags based on build results.
	// By using the unified config as the source of truth, we eliminate the possibility
	// of divergence by design - no mapping, no validation loops, just pure conversion.
	// (P1 Badge fix improvement: ~70 lines of complex mapping logic replaced with 1 line)
	deploymentConfig := build.ConvertToDeployConfig(unifiedConfig, buildResults)
	if deploymentConfig == nil || len(deploymentConfig.Containers) == 0 {
		s.logger.Error(ctx, "no containers in deployment config after conversion",
			zap.Uint("project_id", projectID),
		)
		msg := "No containers found for deployment"
		s.handleBuildError(ctx, projectID, &msg)
		return
	}

	s.logger.Info(ctx, "converted unified config to deployment format",
		zap.Uint("project_id", projectID),
		zap.Int("deploy_container_count", len(deploymentConfig.Containers)),
	)

	// Step 5: Proceed to deployment using the converted snapshot
	// The snapshot maintains consistent configuration (captured before builds)
	// with updated image tags from build results
	s.logger.Info(ctx, "proceeding to deployment",
		zap.Uint("project_id", projectID),
	)

	// Call deployProjectInternal with the captured deployment configuration
	// This method will handle the deployment, monitoring, and project status updates
	if err := s.deployProjectInternal(ctx, projectID, deploymentConfig); err != nil {
		s.logger.Error(ctx, "build and deploy background operation completed with deployment failure",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
		// deployProjectInternal handles its own cleanup via handleDeployFailure
		// Builds were successful, but deployment failed
		return
	}

	s.logger.Info(ctx, "build and deploy background operation completed successfully",
		zap.Uint("project_id", projectID),
	)
}

// handleBuildError handles build failure by resetting project status to 'nothing'
func (s *deployService) handleBuildError(ctx context.Context, projectID uint, summary *string) {
	s.logger.Error(ctx, "handling build error",
		zap.Uint("project_id", projectID),
		zap.Stringp("summary", summary),
	)

	// Create a fresh context with timeout for cleanup operations
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Reset project operation status to 'nothing'
	err := s.txManager.RunInTx(cleanupCtx, func(txCtx context.Context) error {
		proj, err := s.projectRepo.FindByIDForUpdate(txCtx, projectID)
		if err != nil {
			return fmt.Errorf("failed to find project: %w", err)
		}

		// Only reset if project is in 'building' status
		if proj.OperationStatus() == value.ProjectOperationStatusBuilding {
			if err := proj.CompleteBuild(); err != nil {
				return fmt.Errorf("failed to complete build operation: %w", err)
			}

			if err := s.projectRepo.Save(txCtx, proj); err != nil {
				return fmt.Errorf("failed to save project: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		s.logger.Error(ctx, "handleBuildError failed",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)
	} else {
		s.logger.Info(ctx, "project status reset to nothing",
			zap.Uint("project_id", projectID),
		)
	}
}

// hasFailedBuilds checks if any build result indicates failure
func hasFailedBuilds(results []*build.BuildResult) bool {
	for _, result := range results {
		if result == nil {
			// Nil result is treated as failure
			return true
		}
		// Check for non-success terminal states
		if result.Status != "success" && result.Status != "skipped" {
			return true
		}
	}
	return false
}
