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

type ProjectStatusHandler struct {
	getProjectStatusUseCase     *application.GetProjectStatusUseCase
	refreshProjectStatusUseCase *application.RefreshProjectStatusUseCase
	permissionService           service.PermissionService
	projectService              service.ProjectService
	logger                      logger.Logger
}

func NewProjectStatusHandler(
	getProjectStatusUseCase *application.GetProjectStatusUseCase,
	refreshProjectStatusUseCase *application.RefreshProjectStatusUseCase,
	permissionService service.PermissionService,
	projectService service.ProjectService,
	log logger.Logger,
) *ProjectStatusHandler {
	return &ProjectStatusHandler{
		getProjectStatusUseCase:     getProjectStatusUseCase,
		refreshProjectStatusUseCase: refreshProjectStatusUseCase,
		permissionService:           permissionService,
		projectService:              projectService,
		logger:                      log,
	}
}

// GetProjectStatus handles GET /api/v1/projects/:slug/status
// This endpoint retrieves the integrated project status from the database (lightweight query)
// Used for regular polling to check build and deployment status
func (h *ProjectStatusHandler) GetProjectStatus(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "get project status handler started",
		zap.String("handler", "GetProjectStatus"),
		zap.String("slug", slug),
	)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "GetProjectStatus"),
			zap.String("slug", slug),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project slug from URL
	if slug == "" {
		h.logger.Warn(ctx, "missing slug parameter",
			zap.String("handler", "GetProjectStatus"),
		)
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	// Get project by slug first to get project ID
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "GetProjectStatus"),
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
			zap.String("handler", "GetProjectStatus"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.GetProjectStatusInput{
		ProjectID: project.ProjectID(),
	}

	output, err := h.getProjectStatusUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "get project status use case failed",
			zap.Error(err),
			zap.String("handler", "GetProjectStatus"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "get project status handler completed",
		zap.String("handler", "GetProjectStatus"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", project.ProjectID()),
		zap.String("slug", slug),
		zap.String("operation_status", output.OperationStatus),
	)

	response.OK(c, output)
}

// RefreshProjectStatus handles GET /api/v1/projects/:slug/status/refresh
// This endpoint forces a refresh of project status from Kubernetes API
// Used when user explicitly requests fresh status data
func (h *ProjectStatusHandler) RefreshProjectStatus(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "refresh project status handler started",
		zap.String("handler", "RefreshProjectStatus"),
		zap.String("slug", slug),
	)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "RefreshProjectStatus"),
			zap.String("slug", slug),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project slug from URL
	if slug == "" {
		h.logger.Warn(ctx, "missing slug parameter",
			zap.String("handler", "RefreshProjectStatus"),
		)
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	// Get project by slug first to get project ID
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "RefreshProjectStatus"),
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
			zap.String("handler", "RefreshProjectStatus"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.RefreshProjectStatusInput{
		ProjectID: project.ProjectID(),
	}

	output, err := h.refreshProjectStatusUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "refresh project status use case failed",
			zap.Error(err),
			zap.String("handler", "RefreshProjectStatus"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "refresh project status handler completed",
		zap.String("handler", "RefreshProjectStatus"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", project.ProjectID()),
		zap.String("slug", slug),
		zap.String("operation_status", output.OperationStatus),
	)

	response.OK(c, output)
}
