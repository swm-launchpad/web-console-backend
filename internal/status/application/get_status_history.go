package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
)

// GetStatusHistoryUseCase retrieves status check history for a service
type GetStatusHistoryUseCase struct {
	statusRepo repository.StatusRepository
}

// NewGetStatusHistoryUseCase creates a new GetStatusHistoryUseCase
func NewGetStatusHistoryUseCase(statusRepo repository.StatusRepository) *GetStatusHistoryUseCase {
	return &GetStatusHistoryUseCase{
		statusRepo: statusRepo,
	}
}

// Execute retrieves status check history for a specific service within a time period
func (uc *GetStatusHistoryUseCase) Execute(ctx context.Context, serviceName value.ServiceName, hours int) ([]*model.StatusCheck, error) {
	end := time.Now().UTC()
	start := end.Add(-time.Duration(hours) * time.Hour)

	return uc.statusRepo.GetStatusChecksByPeriod(ctx, serviceName, start, end)
}
