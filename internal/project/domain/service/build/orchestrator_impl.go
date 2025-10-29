package build

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
)

// orchestratorImpl implements Orchestrator
type orchestratorImpl struct {
	buildHistoryRepo repository.BuildHistoryRepository
	buildService     Builder
	logger           logger.Logger
}

// NewOrchestrator creates a new Orchestrator instance
func NewOrchestrator(
	buildHistoryRepo repository.BuildHistoryRepository,
	buildService Builder,
	log logger.Logger,
) Orchestrator {
	return &orchestratorImpl{
		buildHistoryRepo: buildHistoryRepo,
		buildService:     buildService,
		logger:           log,
	}
}

// buildTask represents a single build task with its associated data
type buildTask struct {
	index        int
	container    *dto.BuildContainerInfo
	buildHistory *build_history.BuildHistory
	buildResult  *BuildResult
	err          error
}

// BuildAndWait executes builds for all containers in parallel and waits for completion
func (o *orchestratorImpl) BuildAndWait(
	ctx context.Context,
	projectID uint,
	containers []*dto.BuildContainerInfo,
) ([]*BuildResult, error) {
	o.logger.Info(ctx, "Starting build orchestration",
		zap.Uint("project_id", projectID),
		zap.Int("container_count", len(containers)),
	)

	if len(containers) == 0 {
		o.logger.Warn(ctx, "No containers to build",
			zap.Uint("project_id", projectID),
		)
		return []*BuildResult{}, nil
	}

	// Phase 1: Create BUILD_HISTORY records for all containers
	buildHistories, err := o.createBuildHistories(ctx, containers)
	if err != nil {
		o.logger.Error(ctx, "Failed to create build histories",
			zap.Error(err),
			zap.Uint("project_id", projectID),
		)
		return nil, fmt.Errorf("failed to create build histories: %w", err)
	}

	// Phase 2: Execute builds in parallel
	results := o.executeBuildsConcurrently(ctx, projectID, containers, buildHistories)

	// Phase 3: Log summary
	o.logBuildSummary(projectID, results)

	return results, nil
}

// createBuildHistories creates BUILD_HISTORY records for all containers
func (o *orchestratorImpl) createBuildHistories(
	ctx context.Context,
	containers []*dto.BuildContainerInfo,
) ([]*build_history.BuildHistory, error) {
	buildHistories := make([]*build_history.BuildHistory, len(containers))
	createdCount := 0

	for i, container := range containers {
		// Create new BuildHistory in untracked status
		buildHist := build_history.NewBuildHistory(container.ContainerID)

		// Persist to database
		if err := o.buildHistoryRepo.Create(ctx, buildHist); err != nil {
			o.logger.Error(ctx, "Failed to create build history",
				zap.Error(err),
				zap.Uint("container_id", container.ContainerID),
				zap.String("container_name", container.Name),
			)

			// Best-effort cleanup: delete previously created records
			// to avoid orphaned BUILD_HISTORY entries
			o.cleanupBuildHistories(ctx, buildHistories[:createdCount])

			return nil, fmt.Errorf("failed to create build history for container %d: %w", container.ContainerID, err)
		}

		buildHistories[i] = buildHist
		createdCount++

		o.logger.Info(ctx, "Created build history record",
			zap.Uint("build_history_id", buildHist.BuildHistoryID),
			zap.Uint("container_id", container.ContainerID),
			zap.String("container_name", container.Name),
		)
	}

	return buildHistories, nil
}

// executeBuildsConcurrently spawns goroutines for each build and collects results
func (o *orchestratorImpl) executeBuildsConcurrently(
	ctx context.Context,
	_ uint, // projectID - unused but kept for consistency
	containers []*dto.BuildContainerInfo,
	buildHistories []*build_history.BuildHistory,
) []*BuildResult {
	var wg sync.WaitGroup
	resultChan := make(chan buildTask, len(containers))

	// Spawn goroutine for each container build
	for i := range containers {
		wg.Add(1)
		go o.buildContainerWorker(
			ctx,
			&wg,
			resultChan,
			i,
			containers[i],
			buildHistories[i],
		)
	}

	// Wait for all builds to complete and close channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results from channel
	return o.collectBuildResults(ctx, resultChan, len(containers))
}

// buildContainerWorker executes a single container build in a goroutine
func (o *orchestratorImpl) buildContainerWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	resultChan chan<- buildTask,
	index int,
	container *dto.BuildContainerInfo,
	buildHistory *build_history.BuildHistory,
) {
	defer wg.Done()

	o.logger.Info(ctx, "Starting build for container",
		zap.Int("index", index),
		zap.Uint("container_id", container.ContainerID),
		zap.String("container_name", container.Name),
		zap.Uint("build_history_id", buildHistory.BuildHistoryID),
	)

	// Execute build via BuildService
	result, err := o.buildService.BuildContainer(
		ctx,
		buildHistory.BuildHistoryID,
		container,
	)

	// Validate BuildService contract: NEVER return (nil, nil)
	// This catches bugs in BuildService implementations or mocks
	if result == nil && err == nil {
		o.logger.Error(ctx, "CRITICAL: BuildService contract violation - returned (nil, nil)",
			zap.Int("index", index),
			zap.Uint("container_id", container.ContainerID),
			zap.String("container_name", container.Name),
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
		)
		// Treat as a backend error - create synthetic error and result
		err = fmt.Errorf("BuildService contract violation: returned (nil, nil)")
		result = &BuildResult{
			BuildHistoryID: buildHistory.BuildHistoryID,
			Status:         "failed",
			ErrorMessage:   "Internal error: BuildService returned invalid response",
			ShouldBuild:    true,
		}
	}

	task := buildTask{
		index:        index,
		container:    container,
		buildHistory: buildHistory,
		buildResult:  result,
		err:          err,
	}

	if err != nil {
		o.logger.Error(ctx, "Build failed with error",
			zap.Error(err),
			zap.Int("index", index),
			zap.Uint("container_id", container.ContainerID),
			zap.String("container_name", container.Name),
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
		)
	} else if result != nil && result.Status != "success" {
		o.logger.Warn(ctx, "Build completed with non-success status",
			zap.String("status", result.Status),
			zap.String("error_message", result.ErrorMessage),
			zap.Int("index", index),
			zap.Uint("container_id", container.ContainerID),
			zap.String("container_name", container.Name),
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
		)
	} else if result != nil && result.Status == "success" {
		// Success case - safe to dereference result.LatestCommitHash
		o.logger.Info(ctx, "Build completed successfully",
			zap.Int("index", index),
			zap.Uint("container_id", container.ContainerID),
			zap.String("container_name", container.Name),
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.String("commit_hash", result.LatestCommitHash),
		)
	} else {
		// Unexpected case: err == nil but result == nil
		// This should not happen per BuildService contract, but handle defensively
		o.logger.Warn(ctx, "Build returned nil result without error",
			zap.Int("index", index),
			zap.Uint("container_id", container.ContainerID),
			zap.String("container_name", container.Name),
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
		)
	}

	resultChan <- task
}

// collectBuildResults collects results from channel and orders them by index
func (o *orchestratorImpl) collectBuildResults(
	ctx context.Context,
	resultChan <-chan buildTask,
	expectedCount int,
) []*BuildResult {
	// Use map to preserve ordering
	resultMap := make(map[int]*BuildResult, expectedCount)

	// Collect all results from channel
	for task := range resultChan {
		if task.err != nil {
			// If BuildResult is present, preserve it even with error
			// BuildService returns both result and error for terminal states
			// (monitoring timeout, terminal failure with metadata, etc.)
			if task.buildResult != nil {
				resultMap[task.index] = task.buildResult
			} else {
				// Only create fallback when service returned nil BuildResult
				// Check if this is a context cancellation
				if errors.Is(task.err, context.Canceled) || errors.Is(task.err, context.DeadlineExceeded) {
					// Context cancellation - use backend_tracking_lost status (non-terminal)
					// This allows higher layers to distinguish cancellation from real failure
					// BuildHistory remains in non-terminal state for future reconciliation
					o.logger.Warn(ctx, "build cancelled via context",
						zap.Int("index", task.index),
						zap.Uint("container_id", task.container.ContainerID),
						zap.Uint("build_history_id", task.buildHistory.BuildHistoryID),
						zap.Error(task.err),
					)
					// Return concrete BuildResult with backend_tracking_lost status
					// This prevents panic in downstream consumers (e.g., BuildPostProcessor)
					resultMap[task.index] = &BuildResult{
						BuildHistoryID: task.buildHistory.BuildHistoryID,
						Status:         "backend_tracking_lost",
						ErrorMessage:   fmt.Sprintf("Context cancelled during build monitoring: %v", task.err),
						ShouldBuild:    true,
					}
				} else {
					// Real failure with no BuildResult - create fallback
					resultMap[task.index] = &BuildResult{
						BuildHistoryID: task.buildHistory.BuildHistoryID,
						Status:         "failed",
						ErrorMessage:   task.err.Error(),
						ShouldBuild:    true,
					}
				}
			}
		} else {
			resultMap[task.index] = task.buildResult
		}
	}

	// Convert map to ordered slice
	results := make([]*BuildResult, expectedCount)
	for i := range expectedCount {
		results[i] = resultMap[i]
	}

	return results
}

// logBuildSummary logs a summary of all build results
func (o *orchestratorImpl) logBuildSummary(
	projectID uint,
	results []*BuildResult,
) {
	successCount := 0
	failedCount := 0
	otherCount := 0
	nilCount := 0

	for _, result := range results {
		// Handle nil results defensively
		if result == nil {
			nilCount++
			continue
		}

		switch result.Status {
		case "success":
			successCount++
		case "failed":
			failedCount++
		default:
			// Includes "backend_tracking_lost", "skipped", etc.
			otherCount++
		}
	}

	// Use background context for logging summary
	ctx := context.Background()

	logFields := []zap.Field{
		zap.Uint("project_id", projectID),
		zap.Int("total_builds", len(results)),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount),
		zap.Int("other_count", otherCount),
	}

	// Only log nil_count if there are nil results (unexpected case)
	if nilCount > 0 {
		logFields = append(logFields, zap.Int("nil_count", nilCount))
	}

	o.logger.Info(ctx, "Build orchestration completed", logFields...)
}

// cleanupBuildHistories performs best-effort cleanup of created BUILD_HISTORY records
// This is called when BUILD_HISTORY creation fails partway through to avoid orphaned records
func (o *orchestratorImpl) cleanupBuildHistories(
	ctx context.Context,
	buildHistories []*build_history.BuildHistory,
) {
	if len(buildHistories) == 0 {
		return
	}

	o.logger.Warn(ctx, "Cleaning up BUILD_HISTORY records after partial failure",
		zap.Int("count", len(buildHistories)),
	)

	// Best-effort deletion - log errors but don't fail
	// These records are in "untracked" status and won't affect builds
	for _, buildHist := range buildHistories {
		if buildHist == nil {
			continue
		}

		// Mark as backend_trigger_failed to indicate cleanup
		// This is appropriate since orchestration was aborted before Tekton was triggered
		summary := "Orchestration aborted during BUILD_HISTORY creation"
		if err := buildHist.UpdateBackendStatus(
			build_history.BuildHistoryStatusBackendTriggerFailed,
			&summary,
		); err != nil {
			// Log but continue - best effort only
			o.logger.Warn(ctx, "Failed to update BUILD_HISTORY status during cleanup",
				zap.Error(err),
				zap.Uint("build_history_id", buildHist.BuildHistoryID),
			)
			continue
		}

		if err := o.buildHistoryRepo.Save(ctx, buildHist); err != nil {
			// Log but continue - best effort only
			o.logger.Warn(ctx, "Failed to save BUILD_HISTORY during cleanup",
				zap.Error(err),
				zap.Uint("build_history_id", buildHist.BuildHistoryID),
			)
		} else {
			o.logger.Info(ctx, "Marked BUILD_HISTORY as backend_trigger_failed during cleanup",
				zap.Uint("build_history_id", buildHist.BuildHistoryID),
			)
		}
	}
}
