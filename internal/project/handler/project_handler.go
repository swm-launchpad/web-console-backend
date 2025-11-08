package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/common/settings"
	"github.com/swm-launchpad/web-console-backend/internal/project/application"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

// Project policy constants
const (
	DefaultCPULimit     uint32 = 500           // 0.5 core (500 millicores) - matches API_SPECIFICATION.md:116
	DefaultMemoryLimit  uint32 = 1024          // 1GB (1024 Mi)
	DefaultDiskLimit    uint32 = 2048          // 2GB (2048 Mi) - matches API_SPECIFICATION.md:118
	DefaultTrafficLimit uint32 = 1048576       // 1TB (1048576 Mi) - maintained for compatibility
	DefaultPlan                = value.PlanEco // Default plan for new projects
)

type ProjectHandler struct {
	createProjectUseCase    *application.CreateProjectUseCase
	getProjectUseCase       *application.GetProjectUseCase
	getProjectBySlugUseCase *application.GetProjectBySlugUseCase
	updateProjectUseCase    *application.UpdateProjectUseCase
	deleteProjectUseCase    *application.DeleteProjectUseCase
	listProjectsUseCase     *application.ListProjectsUseCase
	permissionService       service.PermissionService
	projectService          service.ProjectService
	settingsService         settings.SettingsService
	logger                  logger.Logger
}

func NewProjectHandler(
	createProjectUseCase *application.CreateProjectUseCase,
	getProjectUseCase *application.GetProjectUseCase,
	getProjectBySlugUseCase *application.GetProjectBySlugUseCase,
	updateProjectUseCase *application.UpdateProjectUseCase,
	deleteProjectUseCase *application.DeleteProjectUseCase,
	listProjectsUseCase *application.ListProjectsUseCase,
	permissionService service.PermissionService,
	projectService service.ProjectService,
	settingsService settings.SettingsService,
	log logger.Logger,
) *ProjectHandler {
	return &ProjectHandler{
		createProjectUseCase:    createProjectUseCase,
		getProjectUseCase:       getProjectUseCase,
		getProjectBySlugUseCase: getProjectBySlugUseCase,
		updateProjectUseCase:    updateProjectUseCase,
		deleteProjectUseCase:    deleteProjectUseCase,
		listProjectsUseCase:     listProjectsUseCase,
		permissionService:       permissionService,
		projectService:          projectService,
		settingsService:         settingsService,
		logger:                  log,
	}
}

// CreateProjectRequest represents the request body for project creation
type CreateProjectRequest struct {
	Name         string  `json:"name" binding:"required,min=1,max=32"`
	Plan         *string `json:"plan,omitempty" binding:"omitempty,oneof=free eco pro"`
	CPULimit     *uint32 `json:"cpu_limit,omitempty" binding:"omitempty,min=500,max=8000"`        // 0.5~8 cores, step 500m
	MemoryLimit  *uint32 `json:"memory_limit,omitempty" binding:"omitempty,min=512,max=16384"`    // 0.5~16GB, step 512Mi
	DiskLimit    *uint32 `json:"disk_limit,omitempty" binding:"omitempty,min=1024,max=3072"`      // 1~3GB, step 512Mi
	TrafficLimit *uint32 `json:"traffic_limit,omitempty" binding:"omitempty,min=128,max=1048576"` // Maintained for compatibility
}

// CreateProject handles POST /api/v1/projects
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "create project handler started",
		zap.String("handler", "CreateProject"),
	)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "CreateProject"),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "CreateProject"),
		)
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]any{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	// Prepare plan
	var plan *value.Plan
	if req.Plan != nil {
		p, err := value.NewPlan(*req.Plan)
		if err != nil {
			h.logger.Warn(ctx, "invalid plan provided",
				zap.Error(err),
				zap.String("handler", "CreateProject"),
				zap.String("plan", *req.Plan),
			)
			response.Error(c, projecterrors.ErrInvalidPlan, mapProjectError, response.WithDetails(map[string]any{
				"message": "Invalid plan: must be one of free, eco, or pro",
			}))
			return
		}
		plan = &p
	} else {
		// Default plan if not provided
		defaultPlan := DefaultPlan
		plan = &defaultPlan
	}

	// Prepare resource limits with defaults
	var cpuLimit, memoryLimit, diskLimit, trafficLimit uint32

	// Free plan has fixed resources - fetch from DB to stay in sync with ValidationService
	if plan != nil && *plan == value.PlanFree {
		// Get Free plan limits from settings (fail if not found)
		freeCPU, err := h.settingsService.GetFreePlanCPULimit()
		if err != nil {
			h.logger.Error(ctx, "failed to get free plan CPU limit",
				zap.Error(err),
				zap.String("handler", "CreateProject"),
			)
			response.Error(c, err, mapProjectError)
			return
		}

		freeMem, err := h.settingsService.GetFreePlanMemoryLimit()
		if err != nil {
			h.logger.Error(ctx, "failed to get free plan memory limit",
				zap.Error(err),
				zap.String("handler", "CreateProject"),
			)
			response.Error(c, err, mapProjectError)
			return
		}

		freeDisk, err := h.settingsService.GetFreePlanDiskLimit()
		if err != nil {
			h.logger.Error(ctx, "failed to get free plan disk limit",
				zap.Error(err),
				zap.String("handler", "CreateProject"),
			)
			response.Error(c, err, mapProjectError)
			return
		}

		cpuLimit = uint32(freeCPU)
		memoryLimit = uint32(freeMem)
		diskLimit = uint32(freeDisk)
		trafficLimit = DefaultTrafficLimit
	} else {
		// For non-Free plans, use defaults or client input
		cpuLimit = DefaultCPULimit
		if req.CPULimit != nil {
			cpuLimit = *req.CPULimit
		}

		memoryLimit = DefaultMemoryLimit
		if req.MemoryLimit != nil {
			memoryLimit = *req.MemoryLimit
		}

		diskLimit = DefaultDiskLimit
		if req.DiskLimit != nil {
			diskLimit = *req.DiskLimit
		}

		trafficLimit = DefaultTrafficLimit
		if req.TrafficLimit != nil {
			trafficLimit = *req.TrafficLimit
		}
	}

	input := application.CreateProjectInput{
		Name:         req.Name,
		OwnerID:      userID.(uint),
		Plan:         plan,
		CPULimit:     cpuLimit,
		MemoryLimit:  memoryLimit,
		DiskLimit:    diskLimit,
		TrafficLimit: trafficLimit,
	}

	output, err := h.createProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "create project use case failed",
			zap.Error(err),
			zap.String("handler", "CreateProject"),
			zap.Uint("user_id", userID.(uint)),
			zap.String("project_name", req.Name),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "create project handler completed",
		zap.String("handler", "CreateProject"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", output.ProjectID),
		zap.String("project_slug", output.Slug),
	)

	response.Created(c, output)
}

// GetProject handles GET /api/v1/projects/:slug
func (h *ProjectHandler) GetProject(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "get project handler started",
		zap.String("handler", "GetProject"),
		zap.String("slug", slug),
	)

	// Check user permission for project access
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "GetProject"),
			zap.String("slug", slug),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Execute the use case
	input := application.GetProjectBySlugInput{
		Slug: slug,
	}
	output, err := h.getProjectBySlugUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "get project by slug use case failed",
			zap.Error(err),
			zap.String("handler", "GetProject"),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	// Check user permission for project access
	// If user doesn't have permission, return the same error as "not found" to prevent information disclosure
	if err := h.permissionService.CanUserAccessProject(c.Request.Context(), userID.(uint), output.ProjectID); err != nil {
		h.logger.Warn(ctx, "user permission check failed",
			zap.Error(err),
			zap.String("handler", "GetProject"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", output.ProjectID),
			zap.String("slug", slug),
		)
		// Return project not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	h.logger.Info(ctx, "get project handler completed",
		zap.String("handler", "GetProject"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", output.ProjectID),
		zap.String("slug", slug),
	)

	response.OK(c, output)
}

// UpdateProjectRequest represents the request body for project update
type UpdateProjectRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=1,max=32"`
	Plan         *string `json:"plan,omitempty" binding:"omitempty,oneof=free eco pro"`
	CPULimit     *uint32 `json:"cpu_limit,omitempty" binding:"omitempty,min=500,max=8000"`        // 0.5~8 cores, step 500m
	MemoryLimit  *uint32 `json:"memory_limit,omitempty" binding:"omitempty,min=512,max=16384"`    // 0.5~16GB, step 512Mi
	DiskLimit    *uint32 `json:"disk_limit,omitempty" binding:"omitempty,min=1024,max=3072"`      // 1~3GB, step 512Mi
	TrafficLimit *uint32 `json:"traffic_limit,omitempty" binding:"omitempty,min=128,max=1048576"` // Maintained for compatibility
}

// UpdateProject handles PUT /api/v1/projects/:id
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "update project handler started",
		zap.String("handler", "UpdateProject"),
		zap.String("slug", slug),
	)

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "UpdateProject"),
			zap.String("slug", slug),
		)
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]any{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	// Check user permission for project update
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "UpdateProject"),
			zap.String("slug", slug),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project by slug first to check permission
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "UpdateProject"),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), project.ProjectID()); err != nil {
		h.logger.Warn(ctx, "user permission check failed",
			zap.Error(err),
			zap.String("handler", "UpdateProject"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		// Return project not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Prepare plan if provided
	var plan *value.Plan
	if req.Plan != nil {
		p, err := value.NewPlan(*req.Plan)
		if err != nil {
			h.logger.Warn(ctx, "invalid plan provided",
				zap.Error(err),
				zap.String("handler", "UpdateProject"),
				zap.String("plan", *req.Plan),
			)
			response.Error(c, projecterrors.ErrInvalidPlan, mapProjectError, response.WithDetails(map[string]any{
				"message": "Invalid plan: must be one of free, eco, or pro",
			}))
			return
		}
		plan = &p
	}

	input := application.UpdateProjectInput{
		ProjectID:    project.ProjectID(),
		ActingUserID: userID.(uint),
		Name:         req.Name,
		Plan:         plan,
		CPULimit:     req.CPULimit,
		MemoryLimit:  req.MemoryLimit,
		DiskLimit:    req.DiskLimit,
		TrafficLimit: req.TrafficLimit,
		Status:       nil, // Status update not allowed
	}

	output, err := h.updateProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "update project use case failed",
			zap.Error(err),
			zap.String("handler", "UpdateProject"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "update project handler completed",
		zap.String("handler", "UpdateProject"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", output.ProjectID),
		zap.String("slug", slug),
	)

	response.OK(c, output)
}

// DeleteProject handles DELETE /api/v1/projects/:slug
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "delete project handler started",
		zap.String("handler", "DeleteProject"),
		zap.String("slug", slug),
	)

	// Check user permission for project deletion
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "DeleteProject"),
			zap.String("slug", slug),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project by slug first to check permission
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "DeleteProject"),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), project.ProjectID()); err != nil {
		h.logger.Warn(ctx, "user permission check failed",
			zap.Error(err),
			zap.String("handler", "DeleteProject"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		// Return project not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// Use DeleteProjectUseCase which properly handles cascade deletion of volumes
	input := application.DeleteProjectInput{
		ProjectID: project.ProjectID(),
	}

	output, err := h.deleteProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "delete project use case failed",
			zap.Error(err),
			zap.String("handler", "DeleteProject"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "delete project handler completed",
		zap.String("handler", "DeleteProject"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", project.ProjectID()),
		zap.String("slug", slug),
	)

	response.OK(c, output)
}

// ListProjects handles GET /api/v1/projects
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "list projects handler started",
		zap.String("handler", "ListProjects"),
	)

	// Get current user ID for security check
	currentUserID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "ListProjects"),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get query parameters
	userID := currentUserID.(uint)
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if parsedUserID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			id := uint(parsedUserID)
			// Security check: users can only list their own projects
			if id != currentUserID.(uint) {
				h.logger.Warn(ctx, "user trying to list other user's projects",
					zap.String("handler", "ListProjects"),
					zap.Uint("current_user_id", currentUserID.(uint)),
					zap.Uint("requested_user_id", id),
				)
				response.Error(c, projecterrors.ErrPermissionDenied, mapProjectError)
				return
			}
			userID = id
		}
	}

	input := application.ListProjectsInput{
		UserID: userID,
	}

	output, err := h.listProjectsUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "list projects use case failed",
			zap.Error(err),
			zap.String("handler", "ListProjects"),
			zap.Uint("user_id", userID),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "list projects handler completed",
		zap.String("handler", "ListProjects"),
		zap.Uint("user_id", userID),
		zap.Int("project_count", len(output.Projects)),
	)

	response.OK(c, output)
}
