package service

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

// buildOrchestratorImpl implements BuildOrchestrator
type buildOrchestratorImpl struct {
	buildHistoryRepo repository.BuildHistoryRepository
	buildService     BuildService
	logger           logger.Logger
}

// NewBuildOrchestrator creates a new BuildOrchestrator instance
func NewBuildOrchestrator(
	buildHistoryRepo repository.BuildHistoryRepository,
	buildService BuildService,
	log logger.Logger,
) BuildOrchestrator {
	return &buildOrchestratorImpl{
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
func (o *buildOrchestratorImpl) BuildAndWait(
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
func (o *buildOrchestratorImpl) createBuildHistories(
	ctx context.Context,
	containers []*dto.BuildContainerInfo,
) ([]*build_history.BuildHistory, error) {
	buildHistories := make([]*build_history.BuildHistory, len(containers))

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
			return nil, fmt.Errorf("failed to create build history for container %d: %w", container.ContainerID, err)
		}

		buildHistories[i] = buildHist

		o.logger.Info(ctx, "Created build history record",
			zap.Uint("build_history_id", buildHist.BuildHistoryID),
			zap.Uint("container_id", container.ContainerID),
			zap.String("container_name", container.Name),
		)
	}

	return buildHistories, nil
}

// executeBuildsConcurrently spawns goroutines for each build and collects results
func (o *buildOrchestratorImpl) executeBuildsConcurrently(
	ctx context.Context,
	projectID uint,
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
	return o.collectBuildResults(resultChan, len(containers))
}

// buildContainerWorker executes a single container build in a goroutine
func (o *buildOrchestratorImpl) buildContainerWorker(
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
	} else {
		o.logger.Info(ctx, "Build completed successfully",
			zap.Int("index", index),
			zap.Uint("container_id", container.ContainerID),
			zap.String("container_name", container.Name),
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.String("commit_hash", result.LatestCommitHash),
		)
	}

	resultChan <- task
}

// collectBuildResults collects results from channel and orders them by index
func (o *buildOrchestratorImpl) collectBuildResults(
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
					o.logger.Warn(context.Background(), "build cancelled via context",
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
	for i := 0; i < expectedCount; i++ {
		results[i] = resultMap[i]
	}

	return results
}

// logBuildSummary logs a summary of all build results
func (o *buildOrchestratorImpl) logBuildSummary(
	projectID uint,
	results []*BuildResult,
) {
	successCount := 0
	failedCount := 0
	otherCount := 0

	for _, result := range results {
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
	o.logger.Info(ctx, "Build orchestration completed",
		zap.Uint("project_id", projectID),
		zap.Int("total_builds", len(results)),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount),
		zap.Int("other_count", otherCount),
	)
}
