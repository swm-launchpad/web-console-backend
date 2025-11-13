package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"go.uber.org/zap"
)

type CheckProjectNameInput struct {
	Name   string
	UserID uint
}

type CheckProjectNameOutput struct {
	Exists bool `json:"exists"`
}

type CheckProjectNameUseCase struct {
	projectRepo repository.ProjectRepository
	logger      logger.Logger
}

func NewCheckProjectNameUseCase(projectRepo repository.ProjectRepository, log logger.Logger) *CheckProjectNameUseCase {
	return &CheckProjectNameUseCase{
		projectRepo: projectRepo,
		logger:      log,
	}
}

func (uc *CheckProjectNameUseCase) Execute(ctx context.Context, input CheckProjectNameInput) (*CheckProjectNameOutput, error) {
	uc.logger.Debug(ctx, "check project name started",
		zap.String("name", input.Name),
		zap.Uint("user_id", input.UserID),
	)

	exists, err := uc.projectRepo.ExistsByNameAndUserID(ctx, input.Name, input.UserID)
	if err != nil {
		uc.logger.Error(ctx, "failed to check project name",
			zap.Error(err),
			zap.String("name", input.Name),
			zap.Uint("user_id", input.UserID),
		)
		return nil, err
	}

	uc.logger.Debug(ctx, "check project name completed",
		zap.String("name", input.Name),
		zap.Uint("user_id", input.UserID),
		zap.Bool("exists", exists),
	)

	return &CheckProjectNameOutput{
		Exists: exists,
	}, nil
}
