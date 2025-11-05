package application

import (
	"context"
	"io"
	"time"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

type StreamProjectLogsInput struct {
	ProjectID uint
}

type StreamProjectLogsUseCase struct {
	projectRepo repository.ProjectRepository
	lokiClient  infrastructure.LokiClient
	kubeClient  infrastructure.KubeClient
	logger      logger.Logger
}

func NewStreamProjectLogsUseCase(
	projectRepo repository.ProjectRepository,
	lokiClient infrastructure.LokiClient,
	kubeClient infrastructure.KubeClient,
	log logger.Logger,
) *StreamProjectLogsUseCase {
	return &StreamProjectLogsUseCase{
		projectRepo: projectRepo,
		lokiClient:  lokiClient,
		kubeClient:  kubeClient,
		logger:      log,
	}
}

func (uc *StreamProjectLogsUseCase) Execute(ctx context.Context, input StreamProjectLogsInput) (io.ReadCloser, error) {
	uc.logger.Info(ctx, "Streaming application logs for project",
		zap.Uint("project_id", input.ProjectID),
	)

	// 1. Get project
	project, err := uc.projectRepo.FindByID(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find project for log streaming",
			zap.Uint("project_id", input.ProjectID),
			zap.Error(err),
		)
		return nil, err
	}

	projectSlug := project.Slug().String()

	uc.logger.Info(ctx, "Project found for log streaming",
		zap.Uint("project_id", input.ProjectID),
		zap.String("project_slug", projectSlug),
	)

	// 2. Check if application pods are running
	podsRunning, err := uc.kubeClient.CheckApplicationPodsRunning(ctx, projectSlug)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check application pods",
			zap.String("project_slug", projectSlug),
			zap.Error(err),
		)
		return nil, err
	}

	if !podsRunning {
		uc.logger.Info(ctx, "No running application pods found",
			zap.String("project_slug", projectSlug),
		)
		return nil, projecterrors.ErrNoRunningPods
	}

	uc.logger.Info(ctx, "Application pods are running, starting log stream",
		zap.String("project_slug", projectSlug),
	)

	// 3. Start real-time log streaming from 5 minutes ago to prevent gap
	// Frontend will deduplicate overlapping logs with HTTP history
	since := time.Now().Add(-5 * time.Minute)
	stream, err := uc.lokiClient.StreamApplicationLogs(ctx, projectSlug, since)
	if err != nil {
		uc.logger.Error(ctx, "Failed to stream application logs from Loki",
			zap.String("project_slug", projectSlug),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "Application log streaming started successfully",
		zap.String("project_slug", projectSlug),
		zap.Time("since", since),
	)

	return stream, nil
}
