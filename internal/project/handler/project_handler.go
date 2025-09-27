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

type ProjectHandler struct {
	createProjectUseCase *application.CreateProjectUseCase
	getProjectUseCase    *application.GetProjectUseCase
	updateProjectUseCase *application.UpdateProjectUseCase
	deleteProjectUseCase *application.DeleteProjectUseCase
	listProjectsUseCase  *application.ListProjectsUseCase
	permissionService    service.PermissionService
}

func NewProjectHandler(
	createProjectUseCase *application.CreateProjectUseCase,
	getProjectUseCase *application.GetProjectUseCase,
	updateProjectUseCase *application.UpdateProjectUseCase,
	deleteProjectUseCase *application.DeleteProjectUseCase,
	listProjectsUseCase *application.ListProjectsUseCase,
	permissionService service.PermissionService,
) *ProjectHandler {
	return &ProjectHandler{
		createProjectUseCase: createProjectUseCase,
		getProjectUseCase:    getProjectUseCase,
		updateProjectUseCase: updateProjectUseCase,
		deleteProjectUseCase: deleteProjectUseCase,
		listProjectsUseCase:  listProjectsUseCase,
		permissionService:    permissionService,
	}
}

// CreateProjectRequest represents the request body for project creation
type CreateProjectRequest struct {
	Name          string  `json:"name" binding:"required,min=1,max=100"`
	Slug          string  `json:"slug" binding:"required,min=3,max=63,ascii"`
	FQDN          *string `json:"fqdn,omitempty" binding:"omitempty,max=253,fqdn"`
	Plan          *string `json:"plan,omitempty"`                                                // Plan validation disabled per user request
	CPULimit      *uint32 `json:"cpu_limit,omitempty" binding:"omitempty,min=0,max=4000"`        // 0-4000 millicores
	MemoryRequest *uint32 `json:"memory_request,omitempty" binding:"omitempty,min=128,max=8192"` // 128-8192 Mi
	MemoryLimit   *uint32 `json:"memory_limit,omitempty" binding:"omitempty,min=128,max=8192"`   // 128-8192 Mi
	DiskLimit     *uint32 `json:"disk_limit,omitempty" binding:"omitempty,min=128,max=10240"`    // 128-10240 Mi
	TrafficLimit  *uint64 `json:"traffic_limit,omitempty" binding:"omitempty,min=128"`           // min=128Mi, no max
}

// CreateProject handles POST /api/v1/projects
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]any{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	input := application.CreateProjectInput{
		Name:          req.Name,
		Slug:          req.Slug,
		OwnerID:       userID.(uint),
		FQDN:          req.FQDN,
		Plan:          req.Plan,
		CPULimit:      req.CPULimit,
		MemoryRequest: req.MemoryRequest,
		MemoryLimit:   req.MemoryLimit,
		DiskLimit:     req.DiskLimit,
		TrafficLimit:  req.TrafficLimit,
	}

	output, err := h.createProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.Created(c, output)
}

// GetProject handles GET /api/v1/projects/:id
func (h *ProjectHandler) GetProject(c *gin.Context) {
	projectIDStr := c.Param("id")

	// Try to parse as ID first
	var input application.GetProjectInput
	var projectID uint
	if id, err := strconv.ParseUint(projectIDStr, 10, 32); err == nil {
		projectID = uint(id)
		input.ProjectID = &projectID
	} else {
		// Treat as slug - we need to resolve it to check permission
		input.Slug = &projectIDStr
	}

	// Check user permission for project access
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Execute the use case first
	output, err := h.getProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	// Check user permission for project access
	// If user doesn't have permission, return the same error as "not found" to prevent information disclosure
	if err := h.permissionService.CanUserAccessProject(c.Request.Context(), userID.(uint), output.ProjectID); err != nil {
		// Return project not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	response.OK(c, output)
}

// UpdateProjectRequest represents the request body for project update
type UpdateProjectRequest struct {
	Name          *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	FQDN          *string `json:"fqdn,omitempty" binding:"omitempty,max=253,fqdn"`
	Plan          *string `json:"plan,omitempty"` // Plan validation disabled per user request
	Status        *string `json:"status,omitempty" binding:"omitempty,oneof=active inactive suspended"`
	CPULimit      *uint32 `json:"cpu_limit,omitempty" binding:"omitempty,min=0,max=4000"`        // 0-4000 millicores
	MemoryRequest *uint32 `json:"memory_request,omitempty" binding:"omitempty,min=128,max=8192"` // 128-8192 Mi
	MemoryLimit   *uint32 `json:"memory_limit,omitempty" binding:"omitempty,min=128,max=8192"`   // 128-8192 Mi
	DiskLimit     *uint32 `json:"disk_limit,omitempty" binding:"omitempty,min=128,max=10240"`    // 128-10240 Mi
	TrafficLimit  *uint64 `json:"traffic_limit,omitempty" binding:"omitempty,min=128"`           // min=128Mi, no max
}

// UpdateProject handles PUT /api/v1/projects/:id
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]any{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]any{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	// Check user permission for project update
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), uint(projectID)); err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	input := application.UpdateProjectInput{
		ProjectID:     uint(projectID),
		Name:          req.Name,
		FQDN:          req.FQDN,
		Plan:          req.Plan,
		Status:        req.Status,
		CPULimit:      req.CPULimit,
		MemoryRequest: req.MemoryRequest,
		MemoryLimit:   req.MemoryLimit,
		DiskLimit:     req.DiskLimit,
		TrafficLimit:  req.TrafficLimit,
	}

	output, err := h.updateProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}

// DeleteProject handles DELETE /api/v1/projects/:id
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]any{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	// Check user permission for project deletion
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), uint(projectID)); err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	input := application.DeleteProjectInput{
		ProjectID: uint(projectID),
	}

	output, err := h.deleteProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}

// ListProjects handles GET /api/v1/projects
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	// Get current user ID for security check
	currentUserID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get query parameters
	var userIDPtr *uint
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			id := uint(userID)
			// Security check: users can only list their own projects
			if id != currentUserID.(uint) {
				response.Error(c, projecterrors.ErrPermissionDenied, mapProjectError)
				return
			}
			userIDPtr = &id
		}
	}

	// If no user_id specified, use current user
	if userIDPtr == nil {
		id := currentUserID.(uint)
		userIDPtr = &id
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	input := application.ListProjectsInput{
		UserID: userIDPtr,
		Offset: offset,
		Limit:  limit,
	}

	output, err := h.listProjectsUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}
