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
	deployProjectUseCase     *application.DeployProjectUseCase
	getDeploymentUseCase     *application.GetDeploymentUseCase
	refreshDeploymentUseCase *application.RefreshDeploymentUseCase
	permissionService        service.PermissionService
	projectService           service.ProjectService
	logger                   logger.Logger
}

func NewDeploymentHandler(
	deployProjectUseCase *application.DeployProjectUseCase,
	getDeploymentUseCase *application.GetDeploymentUseCase,
	refreshDeploymentUseCase *application.RefreshDeploymentUseCase,
	permissionService service.PermissionService,
	projectService service.ProjectService,
	log logger.Logger,
) *DeploymentHandler {
	return &DeploymentHandler{
		deployProjectUseCase:     deployProjectUseCase,
		getDeploymentUseCase:     getDeploymentUseCase,
		refreshDeploymentUseCase: refreshDeploymentUseCase,
		permissionService:        permissionService,
		projectService:           projectService,
		logger:                   log,
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

// GetDeployment handles GET /api/v1/projects/:slug/deployments/latest
// This endpoint retrieves the latest deployment status from the database (lightweight query)
func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "get deployment handler started",
		zap.String("handler", "GetDeployment"),
		zap.String("slug", slug),
	)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "GetDeployment"),
			zap.String("slug", slug),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project slug from URL
	if slug == "" {
		h.logger.Warn(ctx, "missing slug parameter",
			zap.String("handler", "GetDeployment"),
		)
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	// Get project by slug first to get project ID
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "GetDeployment"),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	// Check user permission for project access
	// Return project not found instead of permission denied to prevent information disclosure
	if err := h.permissionService.CanUserAccessProject(c.Request.Context(), userID.(uint), project.ProjectID()); err != nil {
		h.logger.Warn(ctx, "user permission check failed",
			zap.Error(err),
			zap.String("handler", "GetDeployment"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.GetDeploymentInput{
		ProjectID: project.ProjectID(),
	}

	output, err := h.getDeploymentUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "get deployment use case failed",
			zap.Error(err),
			zap.String("handler", "GetDeployment"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "get deployment handler completed",
		zap.String("handler", "GetDeployment"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", project.ProjectID()),
		zap.String("slug", slug),
		zap.Uint64("deployment_id", output.DeploymentID),
		zap.String("status", output.Status),
	)

	response.OK(c, output)
}

// RefreshDeployment handles POST /api/v1/projects/:slug/deployments/refresh
// This endpoint queries Kubernetes for the latest deployment status and updates the database
func (h *DeploymentHandler) RefreshDeployment(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "refresh deployment handler started",
		zap.String("handler", "RefreshDeployment"),
		zap.String("slug", slug),
	)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "RefreshDeployment"),
			zap.String("slug", slug),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project slug from URL
	if slug == "" {
		h.logger.Warn(ctx, "missing slug parameter",
			zap.String("handler", "RefreshDeployment"),
		)
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	// Get project by slug first to get project ID
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "RefreshDeployment"),
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
			zap.String("handler", "RefreshDeployment"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.RefreshDeploymentInput{
		ProjectID: project.ProjectID(),
	}

	output, err := h.refreshDeploymentUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "refresh deployment use case failed",
			zap.Error(err),
			zap.String("handler", "RefreshDeployment"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "refresh deployment handler completed",
		zap.String("handler", "RefreshDeployment"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", project.ProjectID()),
		zap.String("slug", slug),
		zap.Uint64("deployment_id", output.DeploymentID),
		zap.String("status", output.Status),
	)

	response.OK(c, output)
}
