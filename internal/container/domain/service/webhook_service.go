package service

import (
	"context"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
)

// WebhookTokenService provides domain-level webhook token operations
type WebhookTokenService interface {
	GenerateUniqueToken(ctx context.Context) (*model.WebhookToken, error)
}

type webhookTokenService struct {
	repo repository.ContainerRepository
}

// NewWebhookTokenService creates a new WebhookTokenService
func NewWebhookTokenService(repo repository.ContainerRepository) WebhookTokenService {
	return &webhookTokenService{
		repo: repo,
	}
}

// GenerateUniqueToken generates a globally unique webhook token
// Retries up to 10 times if collision occurs
func (s *webhookTokenService) GenerateUniqueToken(ctx context.Context) (*model.WebhookToken, error) {
	const maxRetries = 10

	for i := 0; i < maxRetries; i++ {
		token, err := model.GenerateWebhookToken()
		if err != nil {
			return nil, err
		}

		// Check if token already exists
		exists, err := s.repo.ExistsWebhookToken(ctx, token.Value())
		if err != nil {
			return nil, err
		}

		if !exists {
			return token, nil
		}
		// Token collision - retry
	}

	// Failed to generate unique token after max retries
	return nil, containererrors.ErrDuplicateWebhookToken
}
