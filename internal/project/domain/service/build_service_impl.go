package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
	"go.uber.org/zap"
)

// buildServiceImpl implements the BuildService interface
type buildServiceImpl struct {
	buildHistoryRepo             repository.BuildHistoryRepository
	tektonBuildClient            infrastructure.TektonBuildClient
	kubeBuildClient              infrastructure.KubeBuildClient
	logger                       logger.Logger
	pollingInterval              time.Duration // Polling interval for monitoring build status
	findPipelineRunRetryInterval time.Duration // Retry interval for finding PipelineRun by EventID
	findPipelineRunTotalTimeout  time.Duration // Total timeout for finding PipelineRun
	findPipelineRunMaxRetries    int           // Maximum retry attempts for finding PipelineRun
}

// NewBuildService creates a new BuildService instance
func NewBuildService(
	buildHistoryRepo repository.BuildHistoryRepository,
	tektonBuildClient infrastructure.TektonBuildClient,
	kubeBuildClient infrastructure.KubeBuildClient,
	logger logger.Logger,
) BuildService {
	return &buildServiceImpl{
		buildHistoryRepo:             buildHistoryRepo,
		tektonBuildClient:            tektonBuildClient,
		kubeBuildClient:              kubeBuildClient,
		logger:                       logger,
		pollingInterval:              30 * time.Second, // Default: poll every 30 seconds
		findPipelineRunRetryInterval: 10 * time.Second, // Default: retry every 10 seconds
		findPipelineRunTotalTimeout:  5 * time.Minute,  // Default: 5 minutes total timeout
		findPipelineRunMaxRetries:    300,              // Default: maximum 300 retry attempts
	}
}

// BuildContainer executes a build for a single container
// This method is designed to be called in a goroutine and will block until the build completes or times out
func (s *buildServiceImpl) BuildContainer(
	ctx context.Context,
	buildHistoryID uint,
	container *dto.BuildContainerInfo,
) (*BuildResult, error) {
	s.logger.Info(ctx, "build service started",
		zap.Uint("build_history_id", buildHistoryID),
		zap.Uint("container_id", container.ContainerID),
		zap.String("container_name", container.Name),
	)

	// Load the build history record
	buildHistory, err := s.buildHistoryRepo.FindByID(ctx, buildHistoryID)
	if err != nil {
		s.logger.Error(ctx, "failed to load build history",
			zap.Uint("build_history_id", buildHistoryID),
			zap.Error(err),
		)
		return nil, err
	}

	// Step 1: Trigger Tekton build pipeline
	buildRequest, err := s.prepareBuildRequest(container)
	if err != nil {
		s.logger.Error(ctx, "failed to prepare build request",
			zap.Uint("build_history_id", buildHistoryID),
			zap.Error(err),
		)

		// Update build history to backend_trigger_failed
		summary := fmt.Sprintf("Failed to prepare build request: %v", err)
		if updateErr := buildHistory.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTriggerFailed, &summary); updateErr != nil {
			s.logger.Error(ctx, "failed to update build history status",
				zap.Uint("build_history_id", buildHistoryID),
				zap.Error(updateErr),
			)
			return nil, fmt.Errorf("failed to update build history status: %w", updateErr)
		}

		if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
			s.logger.Error(ctx, "failed to persist build history",
				zap.Uint("build_history_id", buildHistoryID),
				zap.Error(saveErr),
			)
			return nil, fmt.Errorf("failed to persist terminal state (backend_trigger_failed): %w", saveErr)
		}

		return &BuildResult{
			BuildHistoryID: buildHistoryID,
			Status:         "failed",
			ErrorMessage:   summary,
		}, err
	}

	s.logger.Info(ctx, "triggering tekton build",
		zap.Uint("build_history_id", buildHistoryID),
		zap.String("image_name", buildRequest.ImageName),
		zap.String("git_branch", buildRequest.GitHubBranch),
	)

	buildResponse, err := s.tektonBuildClient.TriggerBuild(ctx, buildRequest)
	if err != nil {
		s.logger.Error(ctx, "failed to trigger tekton build",
			zap.Uint("build_history_id", buildHistoryID),
			zap.Error(err),
		)

		// Update build history to backend_trigger_failed
		summary := fmt.Sprintf("Failed to trigger Tekton build: %v", err)
		if updateErr := buildHistory.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTriggerFailed, &summary); updateErr != nil {
			s.logger.Error(ctx, "failed to update build history status",
				zap.Uint("build_history_id", buildHistoryID),
				zap.Error(updateErr),
			)
			return nil, fmt.Errorf("failed to update build history status: %w", updateErr)
		}

		if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
			s.logger.Error(ctx, "failed to persist build history",
				zap.Uint("build_history_id", buildHistoryID),
				zap.Error(saveErr),
			)
			return nil, fmt.Errorf("failed to persist terminal state (backend_trigger_failed): %w", saveErr)
		}

		return &BuildResult{
			BuildHistoryID: buildHistoryID,
			Status:         "failed",
			ErrorMessage:   summary,
		}, err
	}

	s.logger.Info(ctx, "tekton build triggered successfully",
		zap.Uint("build_history_id", buildHistoryID),
		zap.String("event_id", buildResponse.EventID),
	)

	// Update build history with Tekton event ID
	// EventID is CRITICAL for tracking and recovery - without it, the build cannot be monitored
	if updateErr := buildHistory.InitTektonInfo(&buildResponse.EventID, nil); updateErr != nil {
		s.logger.Error(ctx, "CRITICAL: failed to update build history with event ID - build tracking impossible without EventID",
			zap.Uint("build_history_id", buildHistoryID),
			zap.String("event_id", buildResponse.EventID),
			zap.Error(updateErr),
		)

		// Update to backend_tracking_failed since we cannot track this build
		summary := fmt.Sprintf("Failed to save EventID to build history: %v. EventID is required for tracking.", updateErr)
		if statusErr := buildHistory.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingFailed, &summary); statusErr != nil {
			s.logger.Error(ctx, "failed to update build history to tracking_failed",
				zap.Uint("build_history_id", buildHistoryID),
				zap.Error(statusErr),
			)
			return nil, fmt.Errorf("failed to update build history status: %w", statusErr)
		}

		if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
			s.logger.Error(ctx, "failed to persist terminal state (backend_tracking_failed)",
				zap.Uint("build_history_id", buildHistoryID),
				zap.Error(saveErr),
			)
			return nil, fmt.Errorf("failed to persist terminal state (backend_tracking_failed): %w", saveErr)
		}

		return &BuildResult{
			BuildHistoryID: buildHistoryID,
			Status:         "failed",
			ErrorMessage:   summary,
		}, updateErr
	}

	if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
		s.logger.Error(ctx, "CRITICAL: failed to persist event ID to build history - build tracking impossible without EventID",
			zap.Uint("build_history_id", buildHistoryID),
			zap.String("event_id", buildResponse.EventID),
			zap.Error(saveErr),
		)

		// Update to backend_tracking_failed since we cannot track this build
		summary := fmt.Sprintf("Failed to persist EventID to build history: %v. EventID is required for tracking.", saveErr)
		if statusErr := buildHistory.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingFailed, &summary); statusErr != nil {
			s.logger.Error(ctx, "failed to update build history to tracking_failed",
				zap.Uint("build_history_id", buildHistoryID),
				zap.Error(statusErr),
			)
			return nil, fmt.Errorf("failed to update build history status: %w", statusErr)
		}

		if saveErr2 := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr2 != nil {
			s.logger.Error(ctx, "failed to persist terminal state (backend_tracking_failed)",
				zap.Uint("build_history_id", buildHistoryID),
				zap.Error(saveErr2),
			)
			return nil, fmt.Errorf("failed to persist terminal state (backend_tracking_failed): %w", saveErr2)
		}

		return &BuildResult{
			BuildHistoryID: buildHistoryID,
			Status:         "failed",
			ErrorMessage:   summary,
		}, saveErr
	}

	// Step 2: Find PipelineRun name by EventID (with timeout and retries)
	pipelineRunName, err := s.findPipelineRunWithRetry(ctx, buildHistory, buildResponse.EventID)
	if err != nil {
		s.logger.Error(ctx, "failed to find PipelineRun",
			zap.Uint("build_history_id", buildHistoryID),
			zap.String("event_id", buildResponse.EventID),
			zap.Error(err),
		)

		// Special case: Context cancellation should not mark build as terminal failure
		// This matches the Option B approach in monitorBuildStatus (line 376-388)
		// The build state remains untouched and can be reconciled later via force refresh or batch jobs
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			s.logger.Info(ctx, "build context cancelled during PipelineRun lookup - leaving state untouched",
				zap.Uint("build_history_id", buildHistoryID),
				zap.String("event_id", buildResponse.EventID),
			)
			return nil, err
		}

		// Note: findPipelineRunWithRetry may have already updated BUILD_HISTORY to backend_tracking_failed
		// If so, the entity is already mutated but might not be persisted (if Save failed)
		summary := fmt.Sprintf("Failed to find PipelineRun within %v: %v", s.findPipelineRunTotalTimeout, err)

		// Update buildHistory if not yet in terminal state
		if !buildHistory.IsCompleted() {
			if updateErr := buildHistory.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingFailed, &summary); updateErr != nil {
				s.logger.Error(ctx, "failed to update build history status",
					zap.Uint("build_history_id", buildHistoryID),
					zap.Error(updateErr),
				)
				return nil, fmt.Errorf("failed to update build history status: %w", updateErr)
			}
		} else {
			s.logger.Info(ctx, "build history already in terminal state from findPipelineRunWithRetry",
				zap.Uint("build_history_id", buildHistoryID),
				zap.String("current_status", string(buildHistory.Status())),
			)
		}

		// ALWAYS save to DB, even if already completed
		// This handles the case where findPipelineRunWithRetry mutated the entity but failed to persist:
		// - If findPipelineRunWithRetry succeeded in saving: this is an idempotent update (safe)
		// - If findPipelineRunWithRetry failed to save: this retry ensures DB consistency
		// Without this, the entity would be marked completed in-memory but DB would remain 'untracked'
		if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
			s.logger.Error(ctx, "failed to persist build history to DB",
				zap.Uint("build_history_id", buildHistoryID),
				zap.Error(saveErr),
			)
			return nil, fmt.Errorf("failed to persist terminal state (backend_tracking_failed): %w", saveErr)
		}

		return &BuildResult{
			BuildHistoryID: buildHistoryID,
			Status:         "failed",
			ErrorMessage:   summary,
		}, err
	}

	s.logger.Info(ctx, "found PipelineRun",
		zap.Uint("build_history_id", buildHistoryID),
		zap.String("pipeline_run_name", pipelineRunName),
	)

	// Update build history with PipelineRun name
	if updateErr := buildHistory.InitTektonInfo(nil, &pipelineRunName); updateErr != nil {
		s.logger.Warn(ctx, "failed to update build history with PipelineRun name",
			zap.Uint("build_history_id", buildHistoryID),
			zap.String("pipeline_run_name", pipelineRunName),
			zap.Error(updateErr),
		)
	} else if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
		s.logger.Warn(ctx, "failed to persist PipelineRun name to build history",
			zap.Uint("build_history_id", buildHistoryID),
			zap.Error(saveErr),
		)
	}

	// Step 3: Monitor PipelineRun status every 30 seconds
	result, err := s.monitorBuildStatus(ctx, buildHistory, pipelineRunName)
	if err != nil {
		s.logger.Error(ctx, "build monitoring failed",
			zap.Uint("build_history_id", buildHistoryID),
			zap.String("pipeline_run_name", pipelineRunName),
			zap.Error(err),
		)
		return result, err
	}

	s.logger.Info(ctx, "build completed",
		zap.Uint("build_history_id", buildHistoryID),
		zap.String("status", result.Status),
		zap.String("commit_hash", result.LatestCommitHash),
	)

	return result, nil
}

// prepareBuildRequest converts BuildContainerInfo to TektonBuildRequest
func (s *buildServiceImpl) prepareBuildRequest(container *dto.BuildContainerInfo) (*dto.TektonBuildRequest, error) {
	// Validate template requirement
	// Template is required for build pipeline (apply-dockerfile-config task)
	if container.TemplateBody == nil || *container.TemplateBody == "" {
		return nil, fmt.Errorf("template is required for build but not configured for container %s (ID: %d)", container.Slug, container.ContainerID)
	}

	// Convert template_config map to JSON
	var templateConfigJSON json.RawMessage
	if container.TemplateConfig != nil {
		configBytes, err := json.Marshal(container.TemplateConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal template_config: %w", err)
		}
		templateConfigJSON = configBytes
	}

	// Convert build_vars map to JSON
	var buildEnvJSON json.RawMessage
	if container.BuildVars != nil {
		buildVarsBytes, err := json.Marshal(container.BuildVars)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal build_vars: %w", err)
		}
		buildEnvJSON = buildVarsBytes
	}

	// Determine force_build flag
	// Backend always sends needs_build status to Tekton, and Tekton makes the final decision
	forceBuild := "false"
	if container.NeedsBuild {
		forceBuild = "true"
	}

	request := &dto.TektonBuildRequest{
		ProjectID:            fmt.Sprintf("%d", container.ProjectID),
		ContainerID:          fmt.Sprintf("%d", container.ContainerID),
		ImageName:            container.Slug,
		GitHubURL:            container.GitRepositoryURL,
		GitHubBranch:         container.GitBranch,
		DirectoryPath:        stringPtrToString(container.GitDirectoryPath),
		ForceBuild:           forceBuild,
		LastBuildCommitHash:  stringPtrToString(container.LastBuiltCommitHash),
		Template:             stringPtrToString(container.TemplateBody),
		DockerfileConfigJSON: templateConfigJSON,
		BuildEnvJSON:         buildEnvJSON,
		RegistryURL:          "", // Will be set by TektonBuildClient from env var
		InstallationID:       int64PtrToString(container.InstallationID),
	}

	return request, nil
}

// findPipelineRunWithRetry attempts to find PipelineRun by EventID with retries
// Retries periodically for up to totalTimeout or maxRetries attempts, whichever comes first
func (s *buildServiceImpl) findPipelineRunWithRetry(
	ctx context.Context,
	buildHistory *build_history.BuildHistory,
	eventID string,
) (string, error) {
	totalTimeout := s.findPipelineRunTotalTimeout
	maxRetriesLimit := s.findPipelineRunMaxRetries

	calculatedRetries := int(totalTimeout / s.findPipelineRunRetryInterval)
	maxRetries := calculatedRetries
	if maxRetries > maxRetriesLimit {
		maxRetries = maxRetriesLimit
	}

	ticker := time.NewTicker(s.findPipelineRunRetryInterval)
	defer ticker.Stop()

	timeout := time.After(totalTimeout)

	for attempt := 0; attempt < maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			// Permanent failure: timeout expired without finding PipelineRun
			msg := fmt.Sprintf("Timeout waiting for PipelineRun (%v) for EventID %s", totalTimeout, eventID)
			s.logger.Error(ctx, "timeout waiting for PipelineRun",
				zap.Uint("build_history_id", buildHistory.BuildHistoryID),
				zap.String("event_id", eventID),
				zap.Duration("timeout", totalTimeout),
			)
			if updateErr := buildHistory.UpdateBackendStatus(
				build_history.BuildHistoryStatusBackendTrackingFailed, &msg); updateErr != nil {
				s.logger.Warn(ctx, "failed to update build history to tracking_failed",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.String("event_id", eventID),
					zap.Error(updateErr),
				)
			} else if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
				s.logger.Warn(ctx, "failed to persist tracking_failed state",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Error(saveErr),
				)
			}
			return "", fmt.Errorf("timeout waiting for PipelineRun (%v)", totalTimeout)
		case <-ticker.C:
			pipelineRunName, err := s.kubeBuildClient.FindPipelineRunNameByEventID(ctx, eventID)
			if err == nil {
				return pipelineRunName, nil
			}

			// Distinguish between "not found yet" (retriable) vs other errors (connection/auth issues)
			if !errors.Is(err, projecterrors.ErrKubePipelineRunNotFound) {
				// Transient error (network, authentication) → backend_tracking_lost
				msg := fmt.Sprintf("Failed to find PipelineRun by EventID %s: %v", eventID, err)
				if updateErr := buildHistory.UpdateBackendStatus(
					build_history.BuildHistoryStatusBackendTrackingLost, &msg); updateErr != nil {
					s.logger.Warn(ctx, "failed to update build history to tracking_lost",
						zap.Uint("build_history_id", buildHistory.BuildHistoryID),
						zap.String("event_id", eventID),
						zap.Error(updateErr),
					)
				} else if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
					s.logger.Warn(ctx, "failed to persist tracking_lost state",
						zap.Uint("build_history_id", buildHistory.BuildHistoryID),
						zap.Error(saveErr),
					)
				}
			}

			// Log retry attempt (for both not-found and transient errors)
			if attempt%3 == 0 { // Log every 30 seconds
				s.logger.Info(ctx, "waiting for PipelineRun to be created",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.String("event_id", eventID),
					zap.Int("attempt", attempt+1),
					zap.Int("max_attempts", maxRetries),
				)
			}
		}
	}

	// Permanent failure: exhausted all retry attempts without finding PipelineRun
	msg := fmt.Sprintf("PipelineRun not found after %d attempts for EventID %s", maxRetries, eventID)
	s.logger.Error(ctx, "exhausted retries waiting for PipelineRun",
		zap.Uint("build_history_id", buildHistory.BuildHistoryID),
		zap.String("event_id", eventID),
		zap.Int("max_attempts", maxRetries),
	)
	if updateErr := buildHistory.UpdateBackendStatus(
		build_history.BuildHistoryStatusBackendTrackingFailed, &msg); updateErr != nil {
		s.logger.Warn(ctx, "failed to update build history to tracking_failed",
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.String("event_id", eventID),
			zap.Error(updateErr),
		)
	} else if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
		s.logger.Warn(ctx, "failed to persist tracking_failed state",
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.Error(saveErr),
		)
	}
	return "", fmt.Errorf("PipelineRun not found after %d attempts", maxRetries)
}

// monitorBuildStatus monitors the PipelineRun status periodically
// Returns when the build reaches a terminal state or times out
func (s *buildServiceImpl) monitorBuildStatus(
	ctx context.Context,
	buildHistory *build_history.BuildHistory,
	pipelineRunName string,
) (*BuildResult, error) {
	const (
		totalTimeout = 30 * time.Minute // 30 minutes total
	)

	ticker := time.NewTicker(s.pollingInterval)
	defer ticker.Stop()

	timeout := time.After(totalTimeout)

	// Initial status check (don't wait for first tick)
	result, err := s.checkBuildStatus(ctx, buildHistory, pipelineRunName)
	// Return immediately if terminal state reached, even if error is present
	// This ensures we don't keep retrying after marking BuildHistory as terminal
	if result != nil {
		return result, err // Terminal state reached
	}
	// Only retry on non-terminal errors
	if err != nil {
		s.logger.Warn(ctx, "initial status check failed, will retry",
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.Error(err),
		)
	}

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - monitoring goroutine exits but PipelineRun continues in Tekton
			// The BuildHistory record may remain in running/backend_tracking_lost state temporarily
			// This is acceptable because:
			// 1. Users can trigger force refresh to update the status
			// 2. Periodic batch jobs will reconcile stale records
			// 3. Tekton PipelineRun continues executing and will eventually complete
			// This matches the DeploymentService pattern (deploy_service.go:938)
			s.logger.Info(ctx, "build monitoring context cancelled",
				zap.Uint("build_history_id", buildHistory.BuildHistoryID),
				zap.String("pipeline_run_name", pipelineRunName),
			)
			return nil, ctx.Err()

		case <-timeout:
			s.logger.Error(ctx, "build monitoring timeout",
				zap.Uint("build_history_id", buildHistory.BuildHistoryID),
				zap.String("pipeline_run_name", pipelineRunName),
			)

			// Update build history to backend_tracking_failed
			summary := "Build monitoring timeout (30 minutes)"
			if err := buildHistory.UpdateBackendStatus(build_history.BuildHistoryStatusBackendTrackingFailed, &summary); err != nil {
				s.logger.Error(ctx, "failed to update build history status",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Error(err),
				)
				return nil, fmt.Errorf("failed to update build history status: %w", err)
			}

			if err := s.buildHistoryRepo.Save(ctx, buildHistory); err != nil {
				s.logger.Error(ctx, "failed to save build history",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Error(err),
				)
				return nil, fmt.Errorf("failed to persist timeout state: %w", err)
			}

			return &BuildResult{
				BuildHistoryID: buildHistory.BuildHistoryID,
				Status:         "failed",
				ErrorMessage:   summary,
			}, fmt.Errorf("build monitoring timeout")

		case <-ticker.C:
			result, err := s.checkBuildStatus(ctx, buildHistory, pipelineRunName)
			// Return immediately if terminal state reached, even if error is present
			// This ensures we don't keep retrying after marking BuildHistory as terminal
			if result != nil {
				return result, err // Terminal state reached
			}
			// Only retry on non-terminal errors
			if err != nil {
				s.logger.Warn(ctx, "status check failed, will retry",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Error(err),
				)
				continue // Continue monitoring
			}
		}
	}
}

// checkBuildStatus checks the current status of the PipelineRun and updates BUILD_HISTORY
// Returns BuildResult if terminal state is reached, nil if still running
func (s *buildServiceImpl) checkBuildStatus(
	ctx context.Context,
	buildHistory *build_history.BuildHistory,
	pipelineRunName string,
) (*BuildResult, error) {
	pipelineRun, err := s.kubeBuildClient.GetPipelineRunStatus(ctx, pipelineRunName)
	if err != nil {
		// Distinguish between "PipelineRun deleted" (terminal) vs transient errors (retriable)
		if errors.Is(err, projecterrors.ErrKubePipelineRunNotFound) {
			// PipelineRun was deleted from Kubernetes → terminal failure
			msg := fmt.Sprintf("PipelineRun %s not found in Kubernetes", pipelineRunName)
			if updateErr := buildHistory.UpdateBackendStatus(
				build_history.BuildHistoryStatusBackendTrackingFailed, &msg); updateErr != nil {
				s.logger.Error(ctx, "failed to update build history status",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Error(updateErr),
				)
				return nil, fmt.Errorf("failed to update build history status: %w", updateErr)
			}

			if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
				s.logger.Error(ctx, "failed to persist build history",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Error(saveErr),
				)
				return nil, fmt.Errorf("failed to persist terminal state (backend_tracking_failed): %w", saveErr)
			}

			return &BuildResult{
				BuildHistoryID: buildHistory.BuildHistoryID,
				Status:         "failed",
				ErrorMessage:   msg,
			}, fmt.Errorf("PipelineRun deleted")
		}

		// Transient error (network, authentication) → backend_tracking_lost
		msg := fmt.Sprintf("Failed to get PipelineRun status: %v", err)
		if updateErr := buildHistory.UpdateBackendStatus(
			build_history.BuildHistoryStatusBackendTrackingLost, &msg); updateErr != nil {
			s.logger.Warn(ctx, "failed to update build history to tracking_lost",
				zap.Uint("build_history_id", buildHistory.BuildHistoryID),
				zap.Error(updateErr),
			)
		} else if saveErr := s.buildHistoryRepo.Save(ctx, buildHistory); saveErr != nil {
			s.logger.Warn(ctx, "failed to persist tracking_lost state",
				zap.Uint("build_history_id", buildHistory.BuildHistoryID),
				zap.Error(saveErr),
			)
		}

		return nil, fmt.Errorf("failed to get PipelineRun status: %w", err)
	}

	s.logger.Debug(ctx, "PipelineRun status checked",
		zap.Uint("build_history_id", buildHistory.BuildHistoryID),
		zap.String("status", pipelineRun.Status),
		zap.String("reason", pipelineRun.Reason),
	)

	// Handle different statuses
	switch pipelineRun.Status {
	case "Unknown":
		// Build is still running
		if buildHistory.Status() != build_history.BuildHistoryStatusRunning {
			summary := "Build is running"
			// Use Tekton's StartTime if available, fallback to time.Now()
			var startedAt *time.Time
			if pipelineRun.StartTime != nil {
				startedAt = pipelineRun.StartTime
			} else {
				now := time.Now()
				startedAt = &now
			}
			if err := buildHistory.UpdateRunningStatus(&summary, startedAt); err != nil {
				// Log error but continue monitoring - non-terminal state update failure is not critical
				s.logger.Warn(ctx, "failed to update build history to running status",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Error(err),
				)
				return nil, nil
			}

			if err := s.buildHistoryRepo.Save(ctx, buildHistory); err != nil {
				// Log error but continue monitoring - non-terminal state persistence failure is not critical
				s.logger.Warn(ctx, "failed to save build history running status",
					zap.Uint("build_history_id", buildHistory.BuildHistoryID),
					zap.Error(err),
				)
			}
		}
		return nil, nil // Continue monitoring

	case "True":
		// Build succeeded
		return s.handleBuildSuccess(ctx, buildHistory, pipelineRun)

	case "False":
		// Build failed
		return s.handleBuildFailure(ctx, buildHistory, pipelineRun)

	default:
		// Unknown status
		s.logger.Warn(ctx, "unknown PipelineRun status",
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.String("status", pipelineRun.Status),
		)
		return nil, nil // Continue monitoring
	}
}

// handleBuildSuccess handles successful build completion
func (s *buildServiceImpl) handleBuildSuccess(
	ctx context.Context,
	buildHistory *build_history.BuildHistory,
	pipelineRun *dto.PipelineRun,
) (*BuildResult, error) {
	// Extract results
	commitHash := pipelineRun.Results["latest_commit_hash"]
	imageTag := pipelineRun.Results["image_tag"]
	shouldBuild := pipelineRun.Results["should_build"]

	s.logger.Info(ctx, "build succeeded",
		zap.Uint("build_history_id", buildHistory.BuildHistoryID),
		zap.String("commit_hash", commitHash),
		zap.String("image_tag", imageTag),
		zap.String("should_build", shouldBuild),
	)

	// Determine if build was actually executed or skipped
	status := "success"
	if shouldBuild == "false" {
		status = "skipped"
	}

	// Ensure started_at is set for fast builds (completed before first poll)
	// This prevents BUILD_HISTORY.started_at from being NULL
	if _, hasStartedAt := buildHistory.StartedAt(); !hasStartedAt && !buildHistory.IsCompleted() {
		startedAt := time.Now()
		if pipelineRun.StartTime != nil {
			startedAt = *pipelineRun.StartTime
		}
		runningSummary := "Build started"
		if err := buildHistory.UpdateRunningStatus(&runningSummary, &startedAt); err != nil {
			s.logger.Warn(ctx, "failed to set started_at for fast build",
				zap.Uint("build_history_id", buildHistory.BuildHistoryID),
				zap.Error(err),
			)
			// Continue with completion - this is not critical
		}
	}

	// Update build history
	// Use Tekton's CompletionTime if available, fallback to time.Now()
	finishedAt := time.Now()
	if pipelineRun.CompletionTime != nil {
		finishedAt = *pipelineRun.CompletionTime
	}
	summary := fmt.Sprintf("Build %s", status)

	var commitHashPtr *string
	if commitHash != "" {
		commitHashPtr = &commitHash
	}

	var updateErr error
	if status == "skipped" {
		updateErr = buildHistory.UpdateCompleteStatus(build_history.BuildHistoryStatusSkipped, &summary, commitHashPtr, finishedAt)
	} else {
		updateErr = buildHistory.UpdateCompleteStatus(build_history.BuildHistoryStatusSuccess, &summary, commitHashPtr, finishedAt)
	}

	if updateErr != nil {
		s.logger.Error(ctx, "failed to update build history to complete status",
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.String("status", status),
			zap.Error(updateErr),
		)
		return nil, fmt.Errorf("failed to update build history to %s: %w", status, updateErr)
	}

	err := s.buildHistoryRepo.Save(ctx, buildHistory)
	if err != nil {
		s.logger.Error(ctx, "failed to save build history",
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to persist build success: %w", err)
	}

	return &BuildResult{
		BuildHistoryID:   buildHistory.BuildHistoryID,
		Status:           status,
		LatestCommitHash: commitHash,
		ImageTag:         imageTag,
		ShouldBuild:      shouldBuild == "true",
	}, nil
}

// handleBuildFailure handles build failure or cancellation
func (s *buildServiceImpl) handleBuildFailure(
	ctx context.Context,
	buildHistory *build_history.BuildHistory,
	pipelineRun *dto.PipelineRun,
) (*BuildResult, error) {
	s.logger.Error(ctx, "build failed",
		zap.Uint("build_history_id", buildHistory.BuildHistoryID),
		zap.String("reason", pipelineRun.Reason),
		zap.String("message", pipelineRun.Message),
	)

	// Determine status based on reason
	var status build_history.BuildHistoryStatus
	var resultStatus string

	if pipelineRun.Reason == "Cancelled" || pipelineRun.Reason == "PipelineRunCancelled" {
		status = build_history.BuildHistoryStatusCancelled
		resultStatus = "cancelled"
	} else {
		status = build_history.BuildHistoryStatusFailed
		resultStatus = "failed"
	}

	// Ensure started_at is set for fast builds (completed before first poll)
	// This prevents BUILD_HISTORY.started_at from being NULL
	if _, hasStartedAt := buildHistory.StartedAt(); !hasStartedAt && !buildHistory.IsCompleted() {
		startedAt := time.Now()
		if pipelineRun.StartTime != nil {
			startedAt = *pipelineRun.StartTime
		}
		runningSummary := "Build started"
		if err := buildHistory.UpdateRunningStatus(&runningSummary, &startedAt); err != nil {
			s.logger.Warn(ctx, "failed to set started_at for fast build",
				zap.Uint("build_history_id", buildHistory.BuildHistoryID),
				zap.Error(err),
			)
			// Continue with completion - this is not critical
		}
	}

	// Update build history
	// Use Tekton's CompletionTime if available, fallback to time.Now()
	finishedAt := time.Now()
	if pipelineRun.CompletionTime != nil {
		finishedAt = *pipelineRun.CompletionTime
	}
	summary := fmt.Sprintf("Build %s: %s", resultStatus, pipelineRun.Message)

	// Try to extract commit hash if available
	commitHash := pipelineRun.Results["latest_commit_hash"]
	var commitHashPtr *string
	if commitHash != "" {
		commitHashPtr = &commitHash
	}

	if err := buildHistory.UpdateCompleteStatus(status, &summary, commitHashPtr, finishedAt); err != nil {
		s.logger.Error(ctx, "failed to update build history to complete status",
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.String("status", resultStatus),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to update build history to %s: %w", resultStatus, err)
	}

	err := s.buildHistoryRepo.Save(ctx, buildHistory)
	if err != nil {
		s.logger.Error(ctx, "failed to save build history",
			zap.Uint("build_history_id", buildHistory.BuildHistoryID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to persist build failure: %w", err)
	}

	return &BuildResult{
		BuildHistoryID:   buildHistory.BuildHistoryID,
		Status:           resultStatus,
		LatestCommitHash: commitHash,
		ErrorMessage:     pipelineRun.Message,
	}, nil
}

// Helper functions

func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int64PtrToString(i *int64) string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%d", *i)
}
