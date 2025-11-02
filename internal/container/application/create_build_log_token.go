package application

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

// CreateBuildLogTokenInput represents the input for creating a build log token
type CreateBuildLogTokenInput struct {
	UserID      uint
	ContainerID uint
}

// CreateBuildLogTokenOutput represents the output from creating a build log token
type CreateBuildLogTokenOutput struct {
	Token     string
	ExpiresAt time.Time
}

// CreateBuildLogTokenUseCase handles the creation of short-lived tokens for build log access
type CreateBuildLogTokenUseCase struct {
	permissionService service.PermissionService
	jwtUtil           *jwt.JWTUtil
	logger            logger.Logger
}

// NewCreateBuildLogTokenUseCase creates a new instance of CreateBuildLogTokenUseCase
func NewCreateBuildLogTokenUseCase(
	permissionService service.PermissionService,
	jwtUtil *jwt.JWTUtil,
	log logger.Logger,
) *CreateBuildLogTokenUseCase {
	return &CreateBuildLogTokenUseCase{
		permissionService: permissionService,
		jwtUtil:           jwtUtil,
		logger:            log,
	}
}

// Execute creates a build log token after verifying user permissions
func (uc *CreateBuildLogTokenUseCase) Execute(ctx context.Context, input CreateBuildLogTokenInput) (*CreateBuildLogTokenOutput, error) {
	// Check if user has access to the container (read permission is enough for logs)
	if err := uc.permissionService.CanUserAccessContainer(ctx, input.UserID, input.ContainerID); err != nil {
		uc.logger.Warn(ctx, "User permission denied for build log token creation",
			zap.Uint("user_id", input.UserID),
			zap.Uint("container_id", input.ContainerID),
			zap.Error(err),
		)
		return nil, err
	}

	// Generate build log token (30 minutes expiration)
	token, err := uc.jwtUtil.GenerateBuildLogToken(ctx, input.UserID, input.ContainerID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to generate build log token",
			zap.Uint("user_id", input.UserID),
			zap.Uint("container_id", input.ContainerID),
			zap.Error(err),
		)
		return nil, err
	}

	// Calculate expiration time (30 minutes from now)
	expiresAt := time.Now().Add(30 * time.Minute)

	uc.logger.Info(ctx, "Build log token created successfully",
		zap.Uint("user_id", input.UserID),
		zap.Uint("container_id", input.ContainerID),
		zap.Time("expires_at", expiresAt),
	)

	return &CreateBuildLogTokenOutput{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}
