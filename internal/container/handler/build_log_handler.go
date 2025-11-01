package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/container/application"
	containerservice "github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

// BuildLogHandler handles build log related HTTP requests
type BuildLogHandler struct {
	createBuildLogTokenUC *application.CreateBuildLogTokenUseCase
	containerService      containerservice.ContainerService
	logger                logger.Logger
}

// NewBuildLogHandler creates a new BuildLogHandler instance
func NewBuildLogHandler(
	createBuildLogTokenUC *application.CreateBuildLogTokenUseCase,
	containerService containerservice.ContainerService,
	log logger.Logger,
) *BuildLogHandler {
	return &BuildLogHandler{
		createBuildLogTokenUC: createBuildLogTokenUC,
		containerService:      containerService,
		logger:                log,
	}
}

// CreateBuildLogTokenRequest represents the request to create a build log token
// No body is needed - containerID comes from URL slug
type CreateBuildLogTokenRequest struct{}

// CreateBuildLogTokenResponse represents the response containing the build log token
type CreateBuildLogTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"` // ISO 8601 format
}

// CreateBuildLogToken handles POST /api/v1/containers/:slug/build-log-token
// @Summary Create build log access token
// @Description Creates a short-lived token (30 minutes) for accessing build logs via WebSocket
// @Tags containers
// @Accept json
// @Produce json
// @Param slug path string true "Container slug"
// @Success 200 {object} CreateBuildLogTokenResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/containers/{slug}/build-log-token [post]
func (h *BuildLogHandler) CreateBuildLogToken(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated user ID
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container by slug
	slug := c.Param("slug")
	if slug == "" {
		h.logger.Warn(ctx, "Container slug is missing")
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.containerService.GetContainerBySlug(ctx, slug)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get container by slug",
			zap.String("slug", slug),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Execute use case
	input := application.CreateBuildLogTokenInput{
		UserID:      userID.(uint),
		ContainerID: container.ContainerID(),
	}

	output, err := h.createBuildLogTokenUC.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "Failed to create build log token",
			zap.Uint("user_id", input.UserID),
			zap.Uint("container_id", input.ContainerID),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Return response
	resp := CreateBuildLogTokenResponse{
		Token:     output.Token,
		ExpiresAt: output.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"), // ISO 8601
	}

	h.logger.Info(ctx, "Build log token created successfully",
		zap.Uint("user_id", input.UserID),
		zap.Uint("container_id", input.ContainerID),
		zap.String("expires_at", resp.ExpiresAt),
	)

	response.OK(c, resp)
}
