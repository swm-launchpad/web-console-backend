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
	updateVolumeUseCase *application.UpdateVolumeUseCase
	removeVolumeUseCase *application.RemoveVolumeUseCase
	permissionService   service.PermissionService
}

func NewVolumeHandler(
	addVolumeUseCase *application.AddVolumeUseCase,
	getVolumesUseCase *application.GetVolumesUseCase,
	updateVolumeUseCase *application.UpdateVolumeUseCase,
	removeVolumeUseCase *application.RemoveVolumeUseCase,
	permissionService service.PermissionService,
) *VolumeHandler {
	return &VolumeHandler{
		addVolumeUseCase:    addVolumeUseCase,
		getVolumesUseCase:   getVolumesUseCase,
		updateVolumeUseCase: updateVolumeUseCase,
		removeVolumeUseCase: removeVolumeUseCase,
		permissionService:   permissionService,
	}
}

// AddVolumeRequest represents the request body for volume creation
type AddVolumeRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Name      string `json:"name" binding:"required,min=1,max=63,ascii"`
	Capacity  uint32 `json:"capacity" binding:"required,min=128,max=10240"` // 128-10240 Mi
}

// AddVolume handles POST /api/v1/volumes
func (h *VolumeHandler) AddVolume(c *gin.Context) {
	var req AddVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]interface{}{
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

	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), req.ProjectID); err != nil {
		response.Error(c, err, mapProjectError)
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

	// Get query parameter for project_id
	var projectIDPtr *uint
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		if projectID, err := strconv.ParseUint(projectIDStr, 10, 32); err == nil {
			id := uint(projectID)
			projectIDPtr = &id

			// Check user permission for the specific project
			if err := h.permissionService.CanUserAccessProject(c.Request.Context(), userID.(uint), id); err != nil {
				response.Error(c, err, mapProjectError)
				return
			}
		}
	}

	input := application.GetVolumesInput{
		ProjectID: projectIDPtr,
	}

	output, err := h.getVolumesUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapProjectError)
		return
	}

	// If no specific project was requested, we need to filter results
	// to only show volumes from projects the user has access to
	if projectIDPtr == nil {
		// Filter volumes based on user permissions
		filteredVolumes := make([]application.VolumeListItem, 0)
		for _, volume := range output.Volumes {
			// Check if user can access this volume's project
			if err := h.permissionService.CanUserAccessProject(c.Request.Context(), userID.(uint), volume.ProjectID); err == nil {
				filteredVolumes = append(filteredVolumes, volume)
			}
		}
		output.Volumes = filteredVolumes
	}

	response.OK(c, output)
}

// UpdateVolumeRequest represents the request body for volume update
type UpdateVolumeRequest struct {
	Name     *string `json:"name,omitempty" binding:"omitempty,min=1,max=63,ascii"`
	Capacity *uint32 `json:"capacity,omitempty" binding:"omitempty,min=128,max=10240"` // 128-10240 Mi
}

// UpdateVolume handles PUT /api/v1/volumes/:id
func (h *VolumeHandler) UpdateVolume(c *gin.Context) {
	volumeIDStr := c.Param("id")
	volumeID, err := strconv.ParseUint(volumeIDStr, 10, 32)
	if err != nil {
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]interface{}{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	var req UpdateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]interface{}{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	// Check user permission for the volume's project
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project ID for the volume
	projectID, err := h.permissionService.GetProjectIDForVolume(c.Request.Context(), uint(volumeID))
	if err != nil {
		// Return volume not found to prevent information disclosure
		response.Error(c, projecterrors.ErrVolumeNotFound, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), projectID); err != nil {
		// Return volume not found instead of permission denied to prevent information disclosure
		response.Error(c, projecterrors.ErrVolumeNotFound, mapProjectError)
		return
	}

	input := application.UpdateVolumeInput{
		VolumeID: uint(volumeID),
		Name:     req.Name,
		Capacity: req.Capacity,
	}

	output, err := h.updateVolumeUseCase.Execute(c.Request.Context(), input)
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
		response.Error(c, projecterrors.ErrValidationFailed, mapProjectError, response.WithDetails(map[string]interface{}{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	// Check user permission for the volume's project
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapProjectError)
		return
	}

	// Get project ID for the volume
	projectID, err := h.permissionService.GetProjectIDForVolume(c.Request.Context(), uint(volumeID))
	if err != nil {
		// Return volume not found to prevent information disclosure
		response.Error(c, projecterrors.ErrVolumeNotFound, mapProjectError)
		return
	}

	if err := h.permissionService.CanUserModifyProject(c.Request.Context(), userID.(uint), projectID); err != nil {
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
