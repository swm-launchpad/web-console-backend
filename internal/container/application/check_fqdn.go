package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"go.uber.org/zap"
)

type CheckFQDNInput struct {
	FQDN string
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
	)

	exists, err := uc.containerRepo.CheckFQDNExists(ctx, input.FQDN)
	if err != nil {
		uc.logger.Error(ctx, "failed to check FQDN",
			zap.Error(err),
			zap.String("fqdn", input.FQDN),
		)
		return nil, err
	}

	uc.logger.Debug(ctx, "check FQDN completed",
		zap.String("fqdn", input.FQDN),
		zap.Bool("exists", exists),
	)

	return &CheckFQDNOutput{
		Exists: exists,
	}, nil
}
