package application

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

type GetProjectLogHistoryInput struct {
	ProjectID uint
	Before    time.Time // Zero value means current time (backward pagination)
	After     time.Time // For forward pagination
	Limit     int       // Default 100
}

type GetProjectLogHistoryOutput struct {
	Logs []infrastructure.ApplicationLogEntry `json:"logs"`
}

type GetProjectLogHistoryUseCase struct {
	projectRepo repository.ProjectRepository
	lokiClient  infrastructure.LokiClient
	logger      logger.Logger
}

func NewGetProjectLogHistoryUseCase(
	projectRepo repository.ProjectRepository,
	lokiClient infrastructure.LokiClient,
	log logger.Logger,
) *GetProjectLogHistoryUseCase {
	return &GetProjectLogHistoryUseCase{
		projectRepo: projectRepo,
		lokiClient:  lokiClient,
		logger:      log,
	}
}

func (uc *GetProjectLogHistoryUseCase) Execute(ctx context.Context, input GetProjectLogHistoryInput) (*GetProjectLogHistoryOutput, error) {
	uc.logger.Info(ctx, "Querying application log history",
		zap.Uint("project_id", input.ProjectID),
		zap.Time("before", input.Before),
		zap.Time("after", input.After),
		zap.Int("limit", input.Limit),
	)

	// 1. Get project
	project, err := uc.projectRepo.FindByID(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find project for log history",
			zap.Uint("project_id", input.ProjectID),
			zap.Error(err),
		)
		return nil, err
	}

	projectSlug := project.Slug().String()

	// 2. Set default limit
	limit := input.Limit
	if limit == 0 {
		limit = 100
	}

	uc.logger.Info(ctx, "Project found for log history query",
		zap.Uint("project_id", input.ProjectID),
		zap.String("project_slug", projectSlug),
		zap.Int("limit", limit),
	)

	// 3. Query Loki (forward or backward based on input)
	var logs []infrastructure.ApplicationLogEntry

	if !input.After.IsZero() {
		// Forward pagination: get logs after specific timestamp
		uc.logger.Info(ctx, "Using forward pagination (after timestamp)",
			zap.Time("after", input.After),
		)
		logs, err = uc.lokiClient.QueryApplicationLogsAfter(
			ctx,
			projectSlug,
			input.After,
			limit,
		)
	} else {
		// Backward pagination: get logs before specific timestamp (default)
		uc.logger.Info(ctx, "Using backward pagination (before timestamp)",
			zap.Time("before", input.Before),
		)
		logs, err = uc.lokiClient.QueryApplicationLogsHistory(
			ctx,
			projectSlug,
			input.Before,
			limit,
		)
	}

	if err != nil {
		uc.logger.Error(ctx, "Failed to query application log history",
			zap.String("project_slug", projectSlug),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "Application log history query completed",
		zap.String("project_slug", projectSlug),
		zap.Int("log_count", len(logs)),
	)

	return &GetProjectLogHistoryOutput{Logs: logs}, nil
}
