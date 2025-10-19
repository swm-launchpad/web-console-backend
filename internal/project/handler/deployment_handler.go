package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/project/application"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
)

type DeploymentHandler struct {
	deployProjectUseCase     *application.DeployProjectUseCase
	refreshDeploymentUseCase *application.RefreshDeploymentUseCase
	permissionService        service.PermissionService
}

func NewDeploymentHandler(
	deployProjectUseCase *application.DeployProjectUseCase,
	refreshDeploymentUseCase *application.RefreshDeploymentUseCase,
	permissionService service.PermissionService,
) *DeploymentHandler {
	return &DeploymentHandler{
		deployProjectUseCase:     deployProjectUseCase,
		refreshDeploymentUseCase: refreshDeploymentUseCase,
		permissionService:        permissionService,
	}
}

// DeployProject handles POST /api/v1/projects/:id/deploy
func (h *DeploymentHandler) DeployProject(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Parse project ID from URL
	projectIDStr := c.Param("id")
	if projectIDStr == "" {
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		response.Error(c, projecterrors.ErrInvalidProjectID, mapProjectError)
		return
	}

	// Check user permission for project modification
	// Return project not found instead of permission denied to prevent information disclosure
	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), uint(projectID)); err != nil {
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.DeployProjectInput{
		ProjectID: uint(projectID),
	}

	output, err := h.deployProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.Accepted(c, output)
}

// RefreshDeployment handles POST /api/v1/projects/:id/deployments/refresh
// This endpoint queries Kubernetes for the latest deployment status and updates the database
func (h *DeploymentHandler) RefreshDeployment(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Parse project ID from URL
	projectIDStr := c.Param("id")
	if projectIDStr == "" {
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		response.Error(c, projecterrors.ErrInvalidProjectID, mapProjectError)
		return
	}

	// Check user permission for project access
	// Return project not found instead of permission denied to prevent information disclosure
	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), uint(projectID)); err != nil {
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Execute use case
	input := application.RefreshDeploymentInput{
		ProjectID: uint(projectID),
	}

	output, err := h.refreshDeploymentUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}
