package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"go.uber.org/zap"
)

type CheckFQDNInput struct {
	FQDN      string
	ProjectID *uint32 // Optional: if provided, check with project scope
}

type CheckFQDNOutput struct {
	Exists bool `json:"exists"`
}

type CheckFQDNUseCase struct {
	containerRepo repository.ContainerRepository
	logger        logger.Logger
}

func NewCheckFQDNUseCase(containerRepo repository.ContainerRepository, log logger.Logger) *CheckFQDNUseCase {
	return &CheckFQDNUseCase{
		containerRepo: containerRepo,
		logger:        log,
	}
}

func (uc *CheckFQDNUseCase) Execute(ctx context.Context, input CheckFQDNInput) (*CheckFQDNOutput, error) {
	uc.logger.Debug(ctx, "check FQDN started",
		zap.String("fqdn", input.FQDN),
		zap.Uint32p("project_id", input.ProjectID),
	)

	var exists bool
	var err error

	// If ProjectID is provided, use project-scoped check (stricter validation)
	if input.ProjectID != nil {
		exists, err = uc.containerRepo.CheckFQDNExistsForProject(ctx, input.FQDN, *input.ProjectID)
	} else {
		// Fallback to global check for backward compatibility
		exists, err = uc.containerRepo.CheckFQDNExists(ctx, input.FQDN)
	}

	if err != nil {
		uc.logger.Error(ctx, "failed to check FQDN",
			zap.Error(err),
			zap.String("fqdn", input.FQDN),
			zap.Uint32p("project_id", input.ProjectID),
		)
		return nil, err
	}

	uc.logger.Debug(ctx, "check FQDN completed",
		zap.String("fqdn", input.FQDN),
		zap.Uint32p("project_id", input.ProjectID),
		zap.Bool("exists", exists),
	)

	return &CheckFQDNOutput{
		Exists: exists,
	}, nil
}
