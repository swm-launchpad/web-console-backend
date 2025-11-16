package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/status/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/status/domain/repository"
)

// GetCurrentStatusUseCase retrieves the current status of all services
type GetCurrentStatusUseCase struct {
	statusRepo repository.StatusRepository
}

// NewGetCurrentStatusUseCase creates a new GetCurrentStatusUseCase
func NewGetCurrentStatusUseCase(statusRepo repository.StatusRepository) *GetCurrentStatusUseCase {
	return &GetCurrentStatusUseCase{
		statusRepo: statusRepo,
	}
}

// Execute retrieves the latest status check for all services
func (uc *GetCurrentStatusUseCase) Execute(ctx context.Context) ([]*model.StatusCheck, error) {
	return uc.statusRepo.GetLatestStatusChecks(ctx)
}
