package application

import (
	"context"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

// RegenerateWebhookTokenUseCase regenerates webhook token for a container
type RegenerateWebhookTokenUseCase struct {
	repo           repository.ContainerRepository
	webhookService service.WebhookTokenService
}

// NewRegenerateWebhookTokenUseCase creates a new RegenerateWebhookTokenUseCase
func NewRegenerateWebhookTokenUseCase(
	repo repository.ContainerRepository,
	webhookService service.WebhookTokenService,
) *RegenerateWebhookTokenUseCase {
	return &RegenerateWebhookTokenUseCase{
		repo:           repo,
		webhookService: webhookService,
	}
}

// RegenerateWebhookTokenRequest represents the input for regenerating token
type RegenerateWebhookTokenRequest struct {
	ContainerID uint32
	UserID      uint32
}

// RegenerateWebhookTokenResponse represents the output after regenerating token
type RegenerateWebhookTokenResponse struct {
	WebhookToken string
}

// Execute regenerates webhook token for the container
func (uc *RegenerateWebhookTokenUseCase) Execute(ctx context.Context, req RegenerateWebhookTokenRequest) (*RegenerateWebhookTokenResponse, error) {
	// Load container
	container, err := uc.repo.FindByID(ctx, uint(req.ContainerID))
	if err != nil {
		return nil, err
	}

	// Check if webhook is enabled
	if !container.WebhookEnabled() {
		return nil, containererrors.ErrWebhookNotEnabled
	}

	// Generate new unique token
	token, err := uc.webhookService.GenerateUniqueToken(ctx)
	if err != nil {
		return nil, err
	}

	// Set new token
	tokenStr := token.Value()
	container.SetWebhookToken(&tokenStr)

	// Save container
	if err := uc.repo.Save(ctx, container); err != nil {
		return nil, err
	}

	return &RegenerateWebhookTokenResponse{
		WebhookToken: tokenStr,
	}, nil
}
