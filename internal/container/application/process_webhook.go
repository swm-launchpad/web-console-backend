package application

import (
	"context"
	"encoding/json"

	"github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
)

// ProcessWebhookUseCase processes webhook call and triggers project deployment
type ProcessWebhookUseCase struct {
	repo repository.ContainerRepository
}

// NewProcessWebhookUseCase creates a new ProcessWebhookUseCase
func NewProcessWebhookUseCase(repo repository.ContainerRepository) *ProcessWebhookUseCase {
	return &ProcessWebhookUseCase{
		repo: repo,
	}
}

// ProcessWebhookRequest represents the input for processing webhook
type ProcessWebhookRequest struct {
	WebhookToken string
	HTTPMethod   string
	SourceIP     *string
	UserAgent    *string
	Payload      *json.RawMessage
}

// ProcessWebhookResponse represents the output after processing webhook
type ProcessWebhookResponse struct {
	ProjectID     uint32
	ContainerID   uint32
	ContainerSlug string
	ContainerName string
}

// Execute processes the webhook call
func (uc *ProcessWebhookUseCase) Execute(ctx context.Context, req ProcessWebhookRequest) (*ProcessWebhookResponse, error) {
	// Find container by webhook token
	containerID, projectID, err := uc.repo.FindContainerByWebhookToken(ctx, req.WebhookToken)
	if err != nil {
		return nil, err
	}

	// Load full container to get details
	container, err := uc.repo.FindByID(ctx, uint(containerID))
	if err != nil {
		return nil, err
	}

	// Check if webhook is enabled
	if !container.WebhookEnabled() {
		return nil, errors.ErrWebhookNotEnabled
	}

	return &ProcessWebhookResponse{
		ProjectID:     projectID,
		ContainerID:   containerID,
		ContainerSlug: container.Slug().String(),
		ContainerName: container.Name(),
	}, nil
}
