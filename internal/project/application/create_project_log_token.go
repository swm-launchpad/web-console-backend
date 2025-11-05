package application

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

type CreateProjectLogTokenInput struct {
	ProjectID uint
	UserID    uint
}

type CreateProjectLogTokenOutput struct {
	Token string `json:"token"`
}

type CreateProjectLogTokenUseCase struct {
	projectRepo repository.ProjectRepository
	jwtSecret   string
	logger      logger.Logger
}

func NewCreateProjectLogTokenUseCase(
	projectRepo repository.ProjectRepository,
	jwtSecret string,
	log logger.Logger,
) *CreateProjectLogTokenUseCase {
	return &CreateProjectLogTokenUseCase{
		projectRepo: projectRepo,
		jwtSecret:   jwtSecret,
		logger:      log,
	}
}

func (uc *CreateProjectLogTokenUseCase) Execute(ctx context.Context, input CreateProjectLogTokenInput) (*CreateProjectLogTokenOutput, error) {
	uc.logger.Info(ctx, "Creating project log token",
		zap.Uint("project_id", input.ProjectID),
		zap.Uint("user_id", input.UserID),
	)

	// Verify project exists (permission check)
	project, err := uc.projectRepo.FindByID(ctx, input.ProjectID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to find project for log token",
			zap.Uint("project_id", input.ProjectID),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "Project found for log token creation",
		zap.Uint("project_id", project.ProjectID()),
		zap.String("project_slug", project.Slug().String()),
	)

	// Create JWT token with minimal permissions (user_id + project_id only)
	// Token valid for 15 minutes
	claims := jwt.MapClaims{
		"user_id":    input.UserID,
		"project_id": project.ProjectID(),
		"exp":        time.Now().Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(uc.jwtSecret))
	if err != nil {
		uc.logger.Error(ctx, "Failed to sign JWT token",
			zap.Uint("project_id", input.ProjectID),
			zap.Error(err),
		)
		return nil, err
	}

	uc.logger.Info(ctx, "Project log token created successfully",
		zap.Uint("project_id", project.ProjectID()),
		zap.Uint("user_id", input.UserID),
	)

	return &CreateProjectLogTokenOutput{Token: tokenString}, nil
}
