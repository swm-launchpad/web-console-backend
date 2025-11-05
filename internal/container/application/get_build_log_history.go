package application

import (
	"context"
	"io"
	"time"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	projectrepo "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

// GetBuildLogHistoryInput represents the input for retrieving historical build logs
type GetBuildLogHistoryInput struct {
	UserID      uint
	ContainerID uint
	// Before optionally limits the query to logs before this timestamp (for hybrid WebSocket+HTTP mode)
	// Zero value means no filtering (query up to finishedAt + 5min buffer)
	Before time.Time
	// After optionally limits the query to logs after this timestamp (for burst loading)
	// Zero value means use startedAt as the start time
	After time.Time
	// Limit specifies the maximum number of log entries to return
	// Zero value will use default (1000). Valid range: 1-5000
	Limit int
}

// GetBuildLogHistoryUseCase handles retrieving historical build logs from Loki
type GetBuildLogHistoryUseCase struct {
	buildHistoryRepo  projectrepo.BuildHistoryRepository
	lokiClient        infrastructure.LokiClient
	permissionService service.PermissionService
	logger            logger.Logger
}

// NewGetBuildLogHistoryUseCase creates a new instance of GetBuildLogHistoryUseCase
func NewGetBuildLogHistoryUseCase(
	buildHistoryRepo projectrepo.BuildHistoryRepository,
	lokiClient infrastructure.LokiClient,
	permissionService service.PermissionService,
	log logger.Logger,
) *GetBuildLogHistoryUseCase {
	return &GetBuildLogHistoryUseCase{
		buildHistoryRepo:  buildHistoryRepo,
		lokiClient:        lokiClient,
		permissionService: permissionService,
		logger:            log,
	}
}

// Execute retrieves historical build logs for a container's latest completed build
// Returns an io.ReadCloser containing the Loki query_range response
// The caller is responsible for closing the returned ReadCloser
func (uc *GetBuildLogHistoryUseCase) Execute(ctx context.Context, input GetBuildLogHistoryInput) (io.ReadCloser, error) {
	// Check if user has access to the container (same pattern as CreateBuildLogToken)
	if err := uc.permissionService.CanUserAccessContainer(ctx, input.UserID, input.ContainerID); err != nil {
		uc.logger.Warn(ctx, "User permission denied for build log history",
			zap.Uint("user_id", input.UserID),
			zap.Uint("container_id", input.ContainerID),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "Retrieving historical build logs",
		zap.Uint("user_id", input.UserID),
		zap.Uint("container_id", input.ContainerID),
	)

	// First, try to find active builds (same pattern as StreamBuildLogsUseCase)
	activeBuilds, err := uc.buildHistoryRepo.FindActiveByContainerID(ctx, input.ContainerID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find active builds",
			zap.Uint("container_id", input.ContainerID),
			zap.Error(err),
		)
		return nil, err
	}

	var targetBuild interface {
		TektonPipelineRunName() (string, bool)
		StartedAt() (time.Time, bool)
		FinishedAt() (time.Time, bool)
		IsCompleted() bool
	}
	var buildHistoryID uint

	if len(activeBuilds) > 0 {
		// Active build found
		targetBuild = activeBuilds[0]
		buildHistoryID = activeBuilds[0].BuildHistoryID
		uc.logger.Info(ctx, "Using active build for historical logs",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", buildHistoryID),
		)
	} else {
		// No active build, find the latest build
		latestBuild, err := uc.buildHistoryRepo.FindLatestByContainerID(ctx, input.ContainerID)
		if err != nil {
			if err == projecterrors.ErrBuildHistoryNotFound {
				uc.logger.Warn(ctx, "No build history found for container",
					zap.Uint("container_id", input.ContainerID),
				)
				return nil, projecterrors.ErrBuildHistoryNotFound
			}
			uc.logger.Error(ctx, "Failed to find latest build",
				zap.Uint("container_id", input.ContainerID),
				zap.Error(err),
			)
			return nil, err
		}
		targetBuild = latestBuild
		buildHistoryID = latestBuild.BuildHistoryID
		uc.logger.Info(ctx, "Using latest build for historical logs",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", buildHistoryID),
		)
	}

	// Get PipelineRunName (required for all builds)
	prName, hasPRName := targetBuild.TektonPipelineRunName()
	if !hasPRName {
		uc.logger.Warn(ctx, "Build has no PipelineRunName",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", buildHistoryID),
		)
		return nil, projecterrors.ErrBuildHistoryNotFound
	}

	// Get build time range
	// startedAt is required, finishedAt is optional (for running builds)
	startedAt, hasStartedAt := targetBuild.StartedAt()
	if !hasStartedAt {
		uc.logger.Warn(ctx, "Build missing startedAt",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", buildHistoryID),
		)
		return nil, projecterrors.ErrBuildHistoryNotFound
	}

	// Use finishedAt if available (completed build), otherwise use current time (running build)
	finishedAt, hasFinishedAt := targetBuild.FinishedAt()
	isCompleted := targetBuild.IsCompleted()

	if !hasFinishedAt {
		// Build is still running, use current time
		finishedAt = time.Now()
		uc.logger.Info(ctx, "Found running build for historical logs (up to now)",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", buildHistoryID),
			zap.String("pipeline_run_name", prName),
			zap.Time("started_at", startedAt),
			zap.Bool("is_completed", isCompleted),
		)
	} else {
		uc.logger.Info(ctx, "Found completed build for historical logs",
			zap.Uint("container_id", input.ContainerID),
			zap.Uint("build_history_id", buildHistoryID),
			zap.String("pipeline_run_name", prName),
			zap.Time("started_at", startedAt),
			zap.Time("finished_at", finishedAt),
			zap.Bool("is_completed", isCompleted),
		)
	}

	// Validate and set limit (default: 1000, max: 5000)
	limit := input.Limit
	if limit == 0 {
		limit = 1000 // Default
	} else if limit < 0 || limit > 5000 {
		uc.logger.Warn(ctx, "Invalid limit parameter, using default",
			zap.Int("requested_limit", input.Limit),
			zap.Int("using_limit", 1000),
		)
		limit = 1000
	}

	// Determine query start time
	var queryStartTime time.Time
	if !input.After.IsZero() {
		// If After parameter is provided, use it as startTime (for burst loading)
		queryStartTime = input.After
		uc.logger.Info(ctx, "Using provided After timestamp as query start time",
			zap.Time("after", queryStartTime),
		)
	} else {
		// Use build's startedAt as start time
		queryStartTime = startedAt
	}

	// Determine query end time
	var endTime time.Time
	now := time.Now()

	if !input.Before.IsZero() {
		// If Before parameter is provided, use it as endTime (for hybrid WebSocket+HTTP mode)
		endTime = input.Before
		uc.logger.Info(ctx, "Using provided Before timestamp as query end time",
			zap.Time("before", endTime),
		)
	} else {
		// Use finishedAt + 5min buffer to capture delayed log flushes from Tekton pods
		// If build finished recently (< 5min ago), use current time instead to get latest logs
		endTime = finishedAt.Add(5 * time.Minute)
		if now.Before(endTime) {
			endTime = now
		}
	}

	uc.logger.Info(ctx, "Build log query time range",
		zap.Time("query_start", queryStartTime),
		zap.Time("query_end", endTime),
		zap.Int("limit", limit),
		zap.Duration("time_since_finish", now.Sub(finishedAt)),
		zap.Bool("before_parameter_provided", !input.Before.IsZero()),
		zap.Bool("after_parameter_provided", !input.After.IsZero()),
	)

	// Query historical logs from Loki via HTTP, excluding ecr-repository-check task
	excludeTasks := []string{"ecr-repository-check"}
	logData, err := uc.lokiClient.QueryPipelineRunLogsHTTP(ctx, prName, excludeTasks, queryStartTime, endTime, limit)
	if err != nil {
		uc.logger.Error(ctx, "Failed to query logs from Loki",
			zap.Uint("container_id", input.ContainerID),
			zap.String("pipeline_run_name", prName),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "Successfully retrieved historical build logs",
		zap.Uint("container_id", input.ContainerID),
		zap.String("pipeline_run_name", prName),
	)

	return logData, nil
}
