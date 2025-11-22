package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
)

// DisableWebhookUseCase disables webhook for a container
type DisableWebhookUseCase struct {
	repo repository.ContainerRepository
}

// NewDisableWebhookUseCase creates a new DisableWebhookUseCase
func NewDisableWebhookUseCase(repo repository.ContainerRepository) *DisableWebhookUseCase {
	return &DisableWebhookUseCase{
		repo: repo,
	}
}

// DisableWebhookRequest represents the input for disabling webhook
type DisableWebhookRequest struct {
	ContainerID uint32
	UserID      uint32
}

// Execute disables webhook for the container
func (uc *DisableWebhookUseCase) Execute(ctx context.Context, req DisableWebhookRequest) error {
	// Load container
	container, err := uc.repo.FindByID(ctx, uint(req.ContainerID))
	if err != nil {
		return err
	}

	// Disable webhook
	if err := container.DisableWebhook(); err != nil {
		return err
	}

	// Save container
	return uc.repo.Save(ctx, container)
}
