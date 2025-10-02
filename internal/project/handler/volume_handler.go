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

type VolumeHandler struct {
	addVolumeUseCase    *application.AddVolumeUseCase
	getVolumesUseCase   *application.GetVolumesUseCase
	removeVolumeUseCase *application.RemoveVolumeUseCase
	permissionService   service.PermissionService
}

func NewVolumeHandler(
	addVolumeUseCase *application.AddVolumeUseCase,
	getVolumesUseCase *application.GetVolumesUseCase,
	removeVolumeUseCase *application.RemoveVolumeUseCase,
	permissionService service.PermissionService,
) *VolumeHandler {
	return &VolumeHandler{
		addVolumeUseCase:    addVolumeUseCase,
		getVolumesUseCase:   getVolumesUseCase,
		removeVolumeUseCase: removeVolumeUseCase,
		permissionService:   permissionService,
	}
}

// AddVolumeRequest represents the request body for volume creation
type AddVolumeRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Name      string `json:"name" binding:"required,min=1,max=63"`         // 영소문자로 시작, 영소문자/숫자/하이픈 사용, 영소문자/숫자로 종료
	Capacity  uint32 `json:"capacity" binding:"required,min=128,max=2048"` // 128-2048 Mi
}

// AddVolume handles POST /api/v1/volumes
func (h *VolumeHandler) AddVolume(c *gin.Context) {
	var req AddVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]any{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	// Check user permission for the project
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserAddVolume(c.Request.Context(), userID.(uint), req.ProjectID); err != nil {
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
		response.Error(c, err, mapProjectError)
		return
	}

	response.Created(c, output)
}

// GetVolumes handles GET /api/v1/volumes
func (h *VolumeHandler) GetVolumes(c *gin.Context) {
	// Check user authentication
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get required query parameter for project_id
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		response.Error(c, projecterrors.ErrInvalidProjectID, mapProjectError)
		return
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		response.Error(c, projecterrors.ErrInvalidProjectID, mapProjectError)
		return
	}

	// Check user permission for the specific project
	if err := h.permissionService.CanUserAccessProject(c.Request.Context(), userID.(uint), uint(projectID)); err != nil {
		// Return project not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrProjectNotFound, mapProjectError)
		return
	}

	input := application.GetVolumesInput{
		ProjectID: uint(projectID),
	}

	output, err := h.getVolumesUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}

// RemoveVolume handles DELETE /api/v1/volumes/:id
func (h *VolumeHandler) RemoveVolume(c *gin.Context) {
	volumeIDStr := c.Param("id")
	volumeID, err := strconv.ParseUint(volumeIDStr, 10, 32)
	if err != nil {
		response.Error(c, projecterrors.ErrInvalidVolumeID, mapProjectError)
		return
	}

	// Check user permission for the volume's project
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserRemoveVolume(c.Request.Context(), userID.(uint), uint(volumeID)); err != nil {
		// Return volume not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrVolumeNotFound, mapProjectError)
		return
	}

	input := application.RemoveVolumeInput{
		VolumeID: uint(volumeID),
	}

	output, err := h.removeVolumeUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	response.OK(c, output)
}
