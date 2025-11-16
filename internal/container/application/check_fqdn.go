package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"go.uber.org/zap"
)

type CheckFQDNInput struct {
	FQDN      string
	ProjectID uint32 // Required: project context for accurate FQDN validation
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
		zap.Uint32("project_id", input.ProjectID),
	)

	// Always use project-scoped check for accurate business logic
	// Business rules:
	// 1. Same project: can reuse soft-deleted FQDN immediately
	// 2. Different project: cannot reuse even soft-deleted FQDN (infrastructure using it)
	exists, err := uc.containerRepo.CheckFQDNExistsForProject(ctx, input.FQDN, uint(input.ProjectID))

	if err != nil {
		uc.logger.Error(ctx, "failed to check FQDN",
			zap.Error(err),
			zap.String("fqdn", input.FQDN),
			zap.Uint32("project_id", input.ProjectID),
		)
		return nil, err
	}

	uc.logger.Debug(ctx, "check FQDN completed",
		zap.String("fqdn", input.FQDN),
		zap.Uint32("project_id", input.ProjectID),
		zap.Bool("exists", exists),
	)

	return &CheckFQDNOutput{
		Exists: exists,
	}, nil
}
