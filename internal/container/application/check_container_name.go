package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"go.uber.org/zap"
)

type CheckContainerNameInput struct {
	ProjectID uint
	Name      string
}

type CheckContainerNameOutput struct {
	Exists bool `json:"exists"`
}

type CheckContainerNameUseCase struct {
	containerRepo repository.ContainerRepository
	logger        logger.Logger
}

func NewCheckContainerNameUseCase(containerRepo repository.ContainerRepository, log logger.Logger) *CheckContainerNameUseCase {
	return &CheckContainerNameUseCase{
		containerRepo: containerRepo,
		logger:        log,
	}
}

func (uc *CheckContainerNameUseCase) Execute(ctx context.Context, input CheckContainerNameInput) (*CheckContainerNameOutput, error) {
	uc.logger.Debug(ctx, "check container name started",
		zap.Uint("project_id", input.ProjectID),
		zap.String("name", input.Name),
	)

	exists, err := uc.containerRepo.ExistsByNameAndProjectID(ctx, input.ProjectID, input.Name)
	if err != nil {
		uc.logger.Error(ctx, "failed to check container name",
			zap.Error(err),
			zap.Uint("project_id", input.ProjectID),
			zap.String("name", input.Name),
		)
		return nil, err
	}

	uc.logger.Debug(ctx, "check container name completed",
		zap.Uint("project_id", input.ProjectID),
		zap.String("name", input.Name),
		zap.Bool("exists", exists),
	)

	return &CheckContainerNameOutput{
		Exists: exists,
	}, nil
}
