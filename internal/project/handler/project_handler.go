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

// MVP 강제 정책 상수 - Handler 계층에서 정책 적용
const (
	MVPMaxProjectsPerUser = 3       // 사용자당 최대 3개 프로젝트
	MVPForcedCPULimit     = 1000    // 1000m = 1 CPU core
	MVPForcedMemoryLimit  = 2048    // 2Gi = 2048Mi
	MVPForcedDiskLimit    = 2048    // 2Gi = 2048Mi
	MVPForcedTrafficLimit = 1048576 // 1TB = 1048576Mi
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
	}
}

// CreateProjectRequest represents the request body for project creation
// MVP 단계: FQDN, Plan 입력은 무시되고 nil로 강제됨
// 리소스 제한 입력은 무시되고 MVP 고정값으로 강제됨
type CreateProjectRequest struct {
	Name         string  `json:"name" binding:"required,min=1,max=255"`
	FQDN         *string `json:"fqdn,omitempty"`
	Plan         *string `json:"plan,omitempty"`
	CPULimit     uint32  `json:"cpu_limit" binding:"required,min=100,max=4000"`
	MemoryLimit  uint32  `json:"memory_limit" binding:"required,min=128,max=8192"`
	DiskLimit    uint32  `json:"disk_limit" binding:"required,min=128,max=10240"`
	TrafficLimit uint32  `json:"traffic_limit" binding:"required,min=128,max=1048576"`
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

	// MVP 정책 1: 사용자당 프로젝트 개수 제한 확인
	projectCount, err := h.projectService.CountProjectsByUserID(c.Request.Context(), userID.(uint))
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}
	if projectCount >= MVPMaxProjectsPerUser {
		response.Error(c, projecterrors.ErrProjectLimitExceeded, mapProjectError)
		return
	}

	// MVP 정책 2: FQDN, Plan 강제 적용 (null로 강제, Frontend 입력 무시)
	// MVP 정책 3: 리소스 제한 강제 적용 (Frontend 입력 무시)
	input := application.CreateProjectInput{
		Name:         req.Name,
		OwnerID:      userID.(uint),
		FQDN:         nil,                   // MVP: FQDN not allowed
		Plan:         nil,                   // MVP: Plan not allowed
		CPULimit:     MVPForcedCPULimit,     // MVP: 1000m = 1 CPU core
		MemoryLimit:  MVPForcedMemoryLimit,  // MVP: 2048Mi = 2Gi
		DiskLimit:    MVPForcedDiskLimit,    // MVP: 2048Mi = 2Gi
		TrafficLimit: MVPForcedTrafficLimit, // MVP: 1TB = 1048576Mi
	}

	output, err := h.createProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.Created(c, output)
}

// GetProject handles GET /api/v1/projects/:slug
func (h *ProjectHandler) GetProject(c *gin.Context) {
	slug := c.Param("slug")

	// Check user permission for project access
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Execute the use case
	input := application.GetProjectBySlugInput{
		Slug: slug,
	}
	output, err := h.getProjectBySlugUseCase.Execute(c.Request.Context(), input)
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
// MVP 단계: FQDN, Plan 입력은 무시되고 nil로 강제됨
// 리소스 제한 입력은 무시되고 MVP 고정값으로 강제됨
type UpdateProjectRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	FQDN         *string `json:"fqdn,omitempty"`
	Plan         *string `json:"plan,omitempty"`
	CPULimit     *uint32 `json:"cpu_limit,omitempty" binding:"omitempty,min=100,max=4000"`
	MemoryLimit  *uint32 `json:"memory_limit,omitempty" binding:"omitempty,min=128,max=8192"`
	DiskLimit    *uint32 `json:"disk_limit,omitempty" binding:"omitempty,min=128,max=10240"`
	TrafficLimit *uint32 `json:"traffic_limit,omitempty" binding:"omitempty,min=128,max=1048576"`
}

// UpdateProject handles PUT /api/v1/projects/:id
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	slug := c.Param("slug")

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

	// Get project by slug first to check permission
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), project.ProjectID()); err != nil {
		// Return project not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	// MVP 정책: FQDN, Plan, 리소스 제한 강제 적용 (Frontend 입력 무시)
	// Use UpdateProjectUseCase with transaction wrapper
	cpuLimit := uint32(MVPForcedCPULimit)
	memoryLimit := uint32(MVPForcedMemoryLimit)
	diskLimit := uint32(MVPForcedDiskLimit)
	trafficLimit := uint32(MVPForcedTrafficLimit)

	input := application.UpdateProjectInput{
		ProjectID:    project.ProjectID(),
		Name:         req.Name,
		FQDN:         nil,           // MVP: FQDN not allowed
		Plan:         nil,           // MVP: Plan not allowed
		CPULimit:     &cpuLimit,     // MVP: 1000m = 1 CPU core
		MemoryLimit:  &memoryLimit,  // MVP: 2048Mi = 2Gi
		DiskLimit:    &diskLimit,    // MVP: 2048Mi = 2Gi
		TrafficLimit: &trafficLimit, // MVP: 1TB = 1048576Mi
		Status:       nil,           // Status update not allowed
	}

	output, err := h.updateProjectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}

// DeleteProject handles DELETE /api/v1/projects/:slug
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	slug := c.Param("slug")

	// Check user permission for project deletion
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project by slug first to check permission
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), slug)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), project.ProjectID()); err != nil {
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
	userID := currentUserID.(uint)
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if parsedUserID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			id := uint(parsedUserID)
			// Security check: users can only list their own projects
			if id != currentUserID.(uint) {
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
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}
