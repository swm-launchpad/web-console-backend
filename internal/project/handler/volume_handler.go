package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/project/application"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type VolumeHandler struct {
	addVolumeUseCase       *application.AddVolumeUseCase
	getVolumesUseCase      *application.GetVolumesUseCase
	removeVolumeUseCase    *application.RemoveVolumeUseCase
	checkVolumeNameUseCase *application.CheckVolumeNameUseCase
	permissionService      service.PermissionService
	volumeService          service.VolumeService
	projectService         service.ProjectService
	logger                 logger.Logger
}

func NewVolumeHandler(
	addVolumeUseCase *application.AddVolumeUseCase,
	getVolumesUseCase *application.GetVolumesUseCase,
	removeVolumeUseCase *application.RemoveVolumeUseCase,
	checkVolumeNameUseCase *application.CheckVolumeNameUseCase,
	permissionService service.PermissionService,
	volumeService service.VolumeService,
	projectService service.ProjectService,
	log logger.Logger,
) *VolumeHandler {
	return &VolumeHandler{
		addVolumeUseCase:       addVolumeUseCase,
		getVolumesUseCase:      getVolumesUseCase,
		removeVolumeUseCase:    removeVolumeUseCase,
		checkVolumeNameUseCase: checkVolumeNameUseCase,
		permissionService:      permissionService,
		volumeService:          volumeService,
		projectService:         projectService,
		logger:                 log,
	}
}

// AddVolumeRequest represents the request body for volume creation
type AddVolumeRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Name      string `json:"name" binding:"required,min=1,max=32"`         // Display name, allows up to 32 characters
	Capacity  uint32 `json:"capacity" binding:"required,min=128,max=2048"` // 128-2048 Mi
}

// AddVolume handles POST /api/v1/volumes
func (h *VolumeHandler) AddVolume(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "add volume handler started",
		zap.String("handler", "AddVolume"),
	)

	var req AddVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "AddVolume"),
		)
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]any{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	// Check user permission for the project
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "AddVolume"),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserAddVolume(c.Request.Context(), userID.(uint), req.ProjectID); err != nil {
		h.logger.Warn(ctx, "user permission check failed",
			zap.Error(err),
			zap.String("handler", "AddVolume"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", req.ProjectID),
		)
		// Return project not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	input := application.AddVolumeInput{
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Capacity:  req.Capacity,
	}

	output, err := h.addVolumeUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "add volume use case failed",
			zap.Error(err),
			zap.String("handler", "AddVolume"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", req.ProjectID),
			zap.String("volume_name", req.Name),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "add volume handler completed",
		zap.String("handler", "AddVolume"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", req.ProjectID),
		zap.Uint("volume_id", output.VolumeID),
		zap.String("volume_slug", output.Slug),
	)

	response.Created(c, output)
}

// GetVolumes handles GET /api/v1/volumes
func (h *VolumeHandler) GetVolumes(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "get volumes handler started",
		zap.String("handler", "GetVolumes"),
	)

	// Check user authentication
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "GetVolumes"),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get required query parameter for project_id
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		h.logger.Warn(ctx, "missing project_id parameter",
			zap.String("handler", "GetVolumes"),
		)
		response.Error(c, projecterrors.ErrInvalidProjectID, mapProjectError)
		return
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		h.logger.Warn(ctx, "invalid project_id parameter",
			zap.Error(err),
			zap.String("handler", "GetVolumes"),
			zap.String("project_id_str", projectIDStr),
		)
		response.Error(c, projecterrors.ErrInvalidProjectID, mapProjectError)
		return
	}

	// Check user permission for the specific project
	if err := h.permissionService.CanUserAccessProject(c.Request.Context(), userID.(uint), uint(projectID)); err != nil {
		h.logger.Warn(ctx, "user permission check failed",
			zap.Error(err),
			zap.String("handler", "GetVolumes"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint64("project_id", projectID),
		)
		// Return project not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	input := application.GetVolumesInput{
		ProjectID: uint(projectID),
	}

	output, err := h.getVolumesUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "get volumes use case failed",
			zap.Error(err),
			zap.String("handler", "GetVolumes"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint64("project_id", projectID),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "get volumes handler completed",
		zap.String("handler", "GetVolumes"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint64("project_id", projectID),
		zap.Int("volume_count", len(output.Volumes)),
	)

	response.OK(c, output)
}

// RemoveVolume handles DELETE /api/v1/volumes/:slug
func (h *VolumeHandler) RemoveVolume(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "remove volume handler started",
		zap.String("handler", "RemoveVolume"),
		zap.String("slug", slug),
	)

	// Check user permission for the volume's project
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "RemoveVolume"),
			zap.String("slug", slug),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get volume by slug first to get volume ID for permission check
	volume, err := h.volumeService.GetVolumeBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(ctx, "failed to get volume by slug",
			zap.Error(err),
			zap.String("handler", "RemoveVolume"),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserRemoveVolume(c.Request.Context(), userID.(uint), volume.VolumeID()); err != nil {
		h.logger.Warn(ctx, "user permission check failed",
			zap.Error(err),
			zap.String("handler", "RemoveVolume"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("volume_id", volume.VolumeID()),
			zap.String("slug", slug),
		)
		// Return volume not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrVolumeNotFound, mapProjectError)
		return
	}

	input := application.RemoveVolumeInput{
		VolumeID: volume.VolumeID(),
	}

	output, err := h.removeVolumeUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "remove volume use case failed",
			zap.Error(err),
			zap.String("handler", "RemoveVolume"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("volume_id", volume.VolumeID()),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "remove volume handler completed",
		zap.String("handler", "RemoveVolume"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("volume_id", volume.VolumeID()),
		zap.String("slug", slug),
	)

	response.OK(c, output)
}

// CheckVolumeName handles checking if a volume name already exists in a project
// GET /api/v1/projects/:slug/volumes/check-name?name=xxx
func (h *VolumeHandler) CheckVolumeName(c *gin.Context) {
	ctx := c.Request.Context()
	projectSlug := c.Param("slug")
	name := c.Query("name")

	h.logger.Debug(ctx, "check volume name handler started",
		zap.String("handler", "CheckVolumeName"),
		zap.String("project_slug", projectSlug),
		zap.String("name", name),
	)

	if name == "" {
		h.logger.Warn(ctx, "missing name parameter",
			zap.String("handler", "CheckVolumeName"),
			zap.String("project_slug", projectSlug),
		)
		response.Error(c, projecterrors.ErrMissingField, mapProjectError)
		return
	}

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "CheckVolumeName"),
		)
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project by slug to get project_id
	project, err := h.projectService.GetProjectBySlug(ctx, projectSlug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "CheckVolumeName"),
			zap.String("project_slug", projectSlug),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	// Check if user has permission to access this project
	if err := h.permissionService.CanUserAccessProject(ctx, userID.(uint), project.ProjectID()); err != nil {
		h.logger.Warn(ctx, "user does not have permission to access project",
			zap.Error(err),
			zap.String("handler", "CheckVolumeName"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
		)
		response.Error(c, projecterrors.ErrPermissionDenied, mapProjectError)
		return
	}

	input := application.CheckVolumeNameInput{
		ProjectID: project.ProjectID(),
		Name:      name,
	}

	output, err := h.checkVolumeNameUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "failed to check volume name",
			zap.Error(err),
			zap.String("handler", "CheckVolumeName"),
			zap.String("project_slug", projectSlug),
			zap.String("name", name),
			zap.Uint("project_id", project.ProjectID()),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Debug(ctx, "check volume name handler completed",
		zap.String("handler", "CheckVolumeName"),
		zap.String("project_slug", projectSlug),
		zap.String("name", name),
		zap.Uint("project_id", project.ProjectID()),
		zap.Bool("exists", output.Exists),
	)

	response.OK(c, output)
}
