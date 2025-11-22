package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

// EnableWebhookUseCase enables webhook for a container and generates a token
type EnableWebhookUseCase struct {
	repo           repository.ContainerRepository
	webhookService service.WebhookTokenService
}

// NewEnableWebhookUseCase creates a new EnableWebhookUseCase
func NewEnableWebhookUseCase(
	repo repository.ContainerRepository,
	webhookService service.WebhookTokenService,
) *EnableWebhookUseCase {
	return &EnableWebhookUseCase{
		repo:           repo,
		webhookService: webhookService,
	}
}

// EnableWebhookRequest represents the input for enabling webhook
type EnableWebhookRequest struct {
	ContainerID uint32
	UserID      uint32
}

// EnableWebhookResponse represents the output after enabling webhook
type EnableWebhookResponse struct {
	WebhookToken string
}

// Execute enables webhook for the container
func (uc *EnableWebhookUseCase) Execute(ctx context.Context, req EnableWebhookRequest) (*EnableWebhookResponse, error) {
	// Load container
	container, err := uc.repo.FindByID(ctx, uint(req.ContainerID))
	if err != nil {
		return nil, err
	}

	// Check if webhook is already enabled
	if container.WebhookEnabled() {
		// If already enabled and has token, return existing token
		if container.WebhookToken() != nil {
			return &EnableWebhookResponse{
				WebhookToken: *container.WebhookToken(),
			}, nil
		}
	}

	// Generate unique token
	token, err := uc.webhookService.GenerateUniqueToken(ctx)
	if err != nil {
		return nil, err
	}

	// Enable webhook and set token
	tokenStr := token.Value()
	container.SetWebhookToken(&tokenStr)
	if err := container.EnableWebhook(); err != nil {
		return nil, err
	}

	// Save container
	if err := uc.repo.Save(ctx, container); err != nil {
		return nil, err
	}

	return &EnableWebhookResponse{
		WebhookToken: tokenStr,
	}, nil
}
