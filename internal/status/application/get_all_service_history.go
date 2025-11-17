package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model/value"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
)

// ServiceHistory represents the uptime history for a single service
type ServiceHistory struct {
	ServiceName  value.ServiceName
	DailyUptime  []*model.DailyUptimeData
}

// GetAllServiceHistoryRequest represents the request parameters
type GetAllServiceHistoryRequest struct {
	Days int // Number of days to retrieve (default: 7)
}

// GetAllServiceHistoryResponse represents the response
type GetAllServiceHistoryResponse struct {
	Services []ServiceHistory
}

// GetAllServiceHistoryUseCase retrieves uptime history for all services
type GetAllServiceHistoryUseCase struct {
	repo repository.StatusRepository
}

// NewGetAllServiceHistoryUseCase creates a new instance
func NewGetAllServiceHistoryUseCase(repo repository.StatusRepository) *GetAllServiceHistoryUseCase {
	return &GetAllServiceHistoryUseCase{
		repo: repo,
	}
}

// Execute retrieves daily uptime data for all services
func (uc *GetAllServiceHistoryUseCase) Execute(ctx context.Context, req GetAllServiceHistoryRequest) (*GetAllServiceHistoryResponse, error) {
	// Default to 7 days if not specified
	days := req.Days
	if days <= 0 {
		days = 7
	}

	// Get all service names
	serviceNames := value.AllServiceNames()

	// Collect history for each service
	var services []ServiceHistory
	for _, serviceName := range serviceNames {
		dailyUptime, err := uc.repo.GetDailyUptimeData(ctx, serviceName, days)
		if err != nil {
			// Log error but continue with other services
			continue
		}

		services = append(services, ServiceHistory{
			ServiceName: serviceName,
			DailyUptime: dailyUptime,
		})
	}

	return &GetAllServiceHistoryResponse{
		Services: services,
	}, nil
}
