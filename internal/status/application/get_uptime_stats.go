package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
)

// GetUptimeStatsUseCase retrieves uptime statistics for a service
type GetUptimeStatsUseCase struct {
	statusRepo repository.StatusRepository
}

// NewGetUptimeStatsUseCase creates a new GetUptimeStatsUseCase
func NewGetUptimeStatsUseCase(statusRepo repository.StatusRepository) *GetUptimeStatsUseCase {
	return &GetUptimeStatsUseCase{
		statusRepo: statusRepo,
	}
}

// Execute retrieves uptime statistics for a service over a specified period
func (uc *GetUptimeStatsUseCase) Execute(ctx context.Context, serviceName value.ServiceName, hours int) (*model.UptimeStats, error) {
	return uc.statusRepo.GetUptimeStats(ctx, serviceName, hours)
}
