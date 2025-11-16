package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
)

// GetDailyUptimeUseCase retrieves daily uptime data for a service
type GetDailyUptimeUseCase struct {
	statusRepo repository.StatusRepository
}

// NewGetDailyUptimeUseCase creates a new GetDailyUptimeUseCase
func NewGetDailyUptimeUseCase(statusRepo repository.StatusRepository) *GetDailyUptimeUseCase {
	return &GetDailyUptimeUseCase{
		statusRepo: statusRepo,
	}
}

// Execute retrieves daily uptime data for a service over the specified number of days
func (uc *GetDailyUptimeUseCase) Execute(ctx context.Context, serviceName value.ServiceName, days int) ([]*model.DailyUptimeData, error) {
	return uc.statusRepo.GetDailyUptimeData(ctx, serviceName, days)
}
