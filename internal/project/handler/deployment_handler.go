package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/project/application"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type DeploymentHandler struct {
	deployProjectUseCase *application.DeployProjectUseCase
	permissionService    service.PermissionService
	projectService       service.ProjectService
	logger               logger.Logger
}

func NewDeploymentHandler(
	deployProjectUseCase *application.DeployProjectUseCase,
	permissionService service.PermissionService,
	projectService service.ProjectService,
	log logger.Logger,
) *DeploymentHandler {
	return &DeploymentHandler{
		deployProjectUseCase: deployProjectUseCase,
		permissionService:    permissionService,
		projectService:       projectService,
		logger:               log,
	}
}

// DeployProject handles POST /api/v1/projects/:slug/deploy
func (h *DeploymentHandler) DeployProject(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "deploy project handler started",
		zap.String("handler", "DeployProject"),
		zap.String("slug", slug),
	)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "DeployProject"),
			zap.String("slug", slug),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project slug from URL
	if slug == "" {
		h.logger.Warn(ctx, "missing slug parameter",
			zap.String("handler", "DeployProject"),
		)
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	// Get project by slug first to get project ID
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "DeployProject"),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	// Check user permission for project modification
	// Return project not found instead of permission denied to prevent information disclosure
	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), project.ProjectID()); err != nil {
		h.logger.Warn(ctx, "user permission check failed",
			zap.Error(err),
			zap.String("handler", "DeployProject"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.DeployProjectInput{
		ProjectID: project.ProjectID(),
	}

	output, err := h.deployProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "deploy project use case failed",
			zap.Error(err),
			zap.String("handler", "DeployProject"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "deploy project handler completed",
		zap.String("handler", "DeployProject"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", project.ProjectID()),
		zap.String("slug", slug),
		zap.String("message", output.Message),
	)

	response.Accepted(c, output)
}
