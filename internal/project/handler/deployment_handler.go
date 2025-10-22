package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/project/application"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type DeploymentHandler struct {
	deployProjectUseCase     *application.DeployProjectUseCase
	getDeploymentUseCase     *application.GetDeploymentUseCase
	refreshDeploymentUseCase *application.RefreshDeploymentUseCase
	permissionService        service.PermissionService
	projectService           service.ProjectService
}

func NewDeploymentHandler(
	deployProjectUseCase *application.DeployProjectUseCase,
	getDeploymentUseCase *application.GetDeploymentUseCase,
	refreshDeploymentUseCase *application.RefreshDeploymentUseCase,
	permissionService service.PermissionService,
	projectService service.ProjectService,
) *DeploymentHandler {
	return &DeploymentHandler{
		deployProjectUseCase:     deployProjectUseCase,
		getDeploymentUseCase:     getDeploymentUseCase,
		refreshDeploymentUseCase: refreshDeploymentUseCase,
		permissionService:        permissionService,
		projectService:           projectService,
	}
}

// DeployProject handles POST /api/v1/projects/:slug/deploy
func (h *DeploymentHandler) DeployProject(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project slug from URL
	slug := c.Param("slug")
	if slug == "" {
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	// Get project by slug first to get project ID
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	// Check user permission for project modification
	// Return project not found instead of permission denied to prevent information disclosure
	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), project.ProjectID()); err != nil {
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.DeployProjectInput{
		ProjectID: project.ProjectID(),
	}

	output, err := h.deployProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.Accepted(c, output)
}

// GetDeployment handles GET /api/v1/projects/:slug/deployments/latest
// This endpoint retrieves the latest deployment status from the database (lightweight query)
func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project slug from URL
	slug := c.Param("slug")
	if slug == "" {
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	// Get project by slug first to get project ID
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	// Check user permission for project access
	// Return project not found instead of permission denied to prevent information disclosure
	if err := h.permissionService.CanUserAccessProject(c.Request.Context(), userID.(uint), project.ProjectID()); err != nil {
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.GetDeploymentInput{
		ProjectID: project.ProjectID(),
	}

	output, err := h.getDeploymentUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}

// RefreshDeployment handles POST /api/v1/projects/:slug/deployments/refresh
// This endpoint queries Kubernetes for the latest deployment status and updates the database
func (h *DeploymentHandler) RefreshDeployment(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project slug from URL
	slug := c.Param("slug")
	if slug == "" {
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	// Get project by slug first to get project ID
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	// Check user permission for project modification
	// Return project not found instead of permission denied to prevent information disclosure
	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), project.ProjectID()); err != nil {
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.RefreshDeploymentInput{
		ProjectID: project.ProjectID(),
	}

	output, err := h.refreshDeploymentUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}
