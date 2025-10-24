package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/container/application"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	containerservice "github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	projectservice "github.com/swm-launchpad/web-console-backend/internal/project/domain/service"
	"go.uber.org/zap"
)

type ContainerHandler struct {
	createContainerUC *application.CreateContainerUseCase
	getContainerUC    *application.GetContainerUseCase
	updateContainerUC *application.UpdateContainerUseCase
	deleteContainerUC *application.DeleteContainerUseCase
	listContainersUC  *application.ListContainersUseCase
	addEnvVarUC       *application.AddEnvVarUseCase
	updateEnvVarUC    *application.UpdateEnvVarUseCase
	deleteEnvVarUC    *application.DeleteEnvVarUseCase
	addNetworkUC      *application.AddNetworkUseCase
	deleteNetworkUC   *application.DeleteNetworkUseCase
	addSecretUC       *application.AddSecretUseCase
	updateSecretUC    *application.UpdateSecretUseCase
	deleteSecretUC    *application.DeleteSecretUseCase
	addBuildVarUC     *application.AddBuildVarUseCase
	updateBuildVarUC  *application.UpdateBuildVarUseCase
	deleteBuildVarUC  *application.DeleteBuildVarUseCase
	addMountUC        *application.AddMountUseCase
	deleteMountUC     *application.DeleteMountUseCase
	projectService    projectservice.ProjectService
	containerService  containerservice.ContainerService
	logger            logger.Logger
}

// Helper method to get container by slug
func (h *ContainerHandler) getContainerBySlug(c *gin.Context) (*model.Container, error) {
	slug := c.Param("slug")
	if slug == "" {
		return nil, containererrors.ErrMissingField
	}
	return h.containerService.GetContainerBySlug(c.Request.Context(), slug)
}

func NewContainerHandler(
	createContainerUC *application.CreateContainerUseCase,
	getContainerUC *application.GetContainerUseCase,
	updateContainerUC *application.UpdateContainerUseCase,
	deleteContainerUC *application.DeleteContainerUseCase,
	listContainersUC *application.ListContainersUseCase,
	addEnvVarUC *application.AddEnvVarUseCase,
	updateEnvVarUC *application.UpdateEnvVarUseCase,
	deleteEnvVarUC *application.DeleteEnvVarUseCase,
	addNetworkUC *application.AddNetworkUseCase,
	deleteNetworkUC *application.DeleteNetworkUseCase,
	addSecretUC *application.AddSecretUseCase,
	updateSecretUC *application.UpdateSecretUseCase,
	deleteSecretUC *application.DeleteSecretUseCase,
	addBuildVarUC *application.AddBuildVarUseCase,
	updateBuildVarUC *application.UpdateBuildVarUseCase,
	deleteBuildVarUC *application.DeleteBuildVarUseCase,
	addMountUC *application.AddMountUseCase,
	deleteMountUC *application.DeleteMountUseCase,
	projectService projectservice.ProjectService,
	containerService containerservice.ContainerService,
	log logger.Logger,
) *ContainerHandler {
	return &ContainerHandler{
		createContainerUC: createContainerUC,
		getContainerUC:    getContainerUC,
		updateContainerUC: updateContainerUC,
		deleteContainerUC: deleteContainerUC,
		listContainersUC:  listContainersUC,
		addEnvVarUC:       addEnvVarUC,
		updateEnvVarUC:    updateEnvVarUC,
		deleteEnvVarUC:    deleteEnvVarUC,
		addNetworkUC:      addNetworkUC,
		deleteNetworkUC:   deleteNetworkUC,
		addSecretUC:       addSecretUC,
		updateSecretUC:    updateSecretUC,
		deleteSecretUC:    deleteSecretUC,
		addBuildVarUC:     addBuildVarUC,
		updateBuildVarUC:  updateBuildVarUC,
		deleteBuildVarUC:  deleteBuildVarUC,
		addMountUC:        addMountUC,
		deleteMountUC:     deleteMountUC,
		projectService:    projectService,
		containerService:  containerService,
		logger:            log,
	}
}

// VolumeToCreate represents a volume to be created with the container
type VolumeToCreate struct {
	Name      string `json:"name" binding:"required"`
	Capacity  uint32 `json:"capacity" binding:"required"`
	MountPath string `json:"mount_path" binding:"required"`
}

// CreateContainerRequest represents the request body for creating a container
type CreateContainerRequest struct {
	Name                 string                 `json:"name" binding:"required"`
	GitURL               string                 `json:"git_url,omitempty"`
	GitBranch            string                 `json:"git_branch,omitempty"`
	GitDirectory         *string                `json:"git_directory,omitempty"`
	GitSubpath           *string                `json:"git_subpath,omitempty"`
	GitHubInstallationID *int64                 `json:"github_installation_id,omitempty"`
	CPULimit             *uint32                `json:"cpu_limit,omitempty"`
	MemoryLimit          *uint32                `json:"memory_limit,omitempty"`
	DiskLimit            *uint32                `json:"disk_limit,omitempty"`
	TemplateID           *uint                  `json:"template_id,omitempty"`
	TemplateConfig       map[string]interface{} `json:"template_config,omitempty"`
	Volumes              []VolumeToCreate       `json:"volumes,omitempty"`
}

// ContainerResponse represents the response for container operations
type ContainerResponse struct {
	ContainerID uint   `json:"container_id"`
	ProjectID   uint   `json:"project_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	FQDN        string `json:"fqdn,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// CreateContainer handles creating a new container
func (h *ContainerHandler) CreateContainer(c *gin.Context) {
	ctx := c.Request.Context()
	projectSlug := c.Param("slug")

	h.logger.Info(ctx, "create container handler started",
		zap.String("handler", "CreateContainer"),
		zap.String("project_slug", projectSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "CreateContainer"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get project slug from URL parameter
	if projectSlug == "" {
		h.logger.Warn(ctx, "missing project slug parameter",
			zap.String("handler", "CreateContainer"),
		)
		response.Error(c, containererrors.ErrInvalidProjectID, mapContainerError)
		return
	}

	// Get container slug from URL
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), projectSlug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "CreateContainer"),
			zap.String("project_slug", projectSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}
	projectID := project.ProjectID()

	var req CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "CreateContainer"),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	// Set default values if not provided
	cpuLimit := uint32(1000)
	if req.CPULimit != nil {
		cpuLimit = *req.CPULimit
	}
	memoryLimit := uint32(2048)
	if req.MemoryLimit != nil {
		memoryLimit = *req.MemoryLimit
	}

	// GitDirectory is the same as GitSubpath (backward compatibility)
	gitDirectory := req.GitDirectory
	if req.GitSubpath != nil {
		gitDirectory = req.GitSubpath
	}

	// Convert volumes to application layer format
	var volumes []application.VolumeToCreate
	for _, v := range req.Volumes {
		volumes = append(volumes, application.VolumeToCreate{
			Name:      v.Name,
			Capacity:  v.Capacity,
			MountPath: v.MountPath,
		})
	}

	input := application.CreateContainerInput{
		ProjectID:            projectID,
		UserID:               userID.(uint),
		Name:                 req.Name,
		GitURL:               req.GitURL,
		GitBranch:            req.GitBranch,
		GitDirectory:         gitDirectory,
		GitHubInstallationID: req.GitHubInstallationID,
		CPULimit:             cpuLimit,
		MemoryLimit:          memoryLimit,
		TemplateID:           req.TemplateID,
		TemplateConfig:       req.TemplateConfig,
		Volumes:              volumes,
	}

	output, err := h.createContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "create container use case failed",
			zap.Error(err),
			zap.String("handler", "CreateContainer"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", projectID),
			zap.String("container_name", req.Name),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "create container handler completed",
		zap.String("handler", "CreateContainer"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", projectID),
		zap.Uint("container_id", output.ContainerID),
		zap.String("container_slug", output.Slug),
	)

	resp := ContainerResponse{
		ContainerID: output.ContainerID,
		ProjectID:   output.ProjectID,
		Name:        output.Name,
		Slug:        output.Slug,
		CreatedAt:   output.CreatedAt,
	}

	response.Created(c, resp)
}

// GetContainer handles fetching a container by slug
func (h *ContainerHandler) GetContainer(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "get container handler started",
		zap.String("handler", "GetContainer"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "GetContainer"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container slug from URL
	if containerSlug == "" {
		h.logger.Warn(ctx, "missing container slug parameter",
			zap.String("handler", "GetContainer"),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	// Get container by slug to get container ID
	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "GetContainer"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	input := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}

	output, err := h.getContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "get container use case failed",
			zap.Error(err),
			zap.String("handler", "GetContainer"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "get container handler completed",
		zap.String("handler", "GetContainer"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("container_slug", containerSlug),
	)

	response.OK(c, output)
}

// UpdateContainerRequest represents the request body for updating a container
type UpdateContainerRequest struct {
	Name                      *string                `json:"name,omitempty"`
	StableWindow              *uint32                `json:"stable_window,omitempty"`
	GitHubInstallationID      *int64                 `json:"github_installation_id,omitempty"`
	UnsetGitHubInstallationID bool                   `json:"unset_github_installation_id,omitempty"`
	GitURL                    *string                `json:"git_url,omitempty"`
	GitBranch                 *string                `json:"git_branch,omitempty"`
	GitDirectory              *string                `json:"git_directory,omitempty"`
	CPULimit                  *uint32                `json:"cpu_limit,omitempty"`
	MemoryLimit               *uint32                `json:"memory_limit,omitempty"`
	TemplateID                *uint                  `json:"template_id,omitempty"`
	TemplateConfig            map[string]interface{} `json:"template_config,omitempty"`
}

// UpdateContainer handles updating a container
func (h *ContainerHandler) UpdateContainer(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "update container handler started",
		zap.String("handler", "UpdateContainer"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "UpdateContainer"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "UpdateContainer"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}
	containerID := container.ContainerID()

	var req UpdateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "UpdateContainer"),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	// Determine GitHubInstallationID to pass to use case
	var githubInstallationID *int64
	updateGitHubInstallation := false
	if req.UnsetGitHubInstallationID {
		// Explicitly unset - set to nil
		githubInstallationID = nil
		updateGitHubInstallation = true
	} else if req.GitHubInstallationID != nil {
		// Set to specific value
		githubInstallationID = req.GitHubInstallationID
		updateGitHubInstallation = true
	}
	// If neither flag is set, don't update (keep existing value)

	input := application.UpdateContainerInput{
		ContainerID:                containerID,
		UserID:                     userID.(uint),
		Name:                       req.Name,
		StableWindow:               req.StableWindow,
		GitHubInstallationID:       githubInstallationID,
		UpdateGitHubInstallationID: updateGitHubInstallation,
		GitURL:                     req.GitURL,
		GitBranch:                  req.GitBranch,
		GitDirectory:               req.GitDirectory,
		CPULimit:                   req.CPULimit,
		MemoryLimit:                req.MemoryLimit,
		TemplateID:                 req.TemplateID,
		TemplateConfig:             req.TemplateConfig,
	}

	output, err := h.updateContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "update container use case failed",
			zap.Error(err),
			zap.String("handler", "UpdateContainer"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", containerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "update container handler completed",
		zap.String("handler", "UpdateContainer"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", output.ContainerID),
		zap.String("container_slug", output.Slug),
	)

	resp := ContainerResponse{
		ContainerID: output.ContainerID,
		Name:        output.Name,
		Slug:        output.Slug,
		UpdatedAt:   output.UpdatedAt,
	}

	response.OK(c, resp)
}

// DeleteContainer handles deleting a container
func (h *ContainerHandler) DeleteContainer(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "delete container handler started",
		zap.String("handler", "DeleteContainer"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "DeleteContainer"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "DeleteContainer"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	input := application.DeleteContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}

	output, err := h.deleteContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "delete container use case failed",
			zap.Error(err),
			zap.String("handler", "DeleteContainer"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "delete container handler completed",
		zap.String("handler", "DeleteContainer"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("container_slug", containerSlug),
	)

	response.OK(c, output)
}

// ListContainers handles fetching all containers for a project
func (h *ContainerHandler) ListContainers(c *gin.Context) {
	ctx := c.Request.Context()
	projectSlug := c.Param("slug")

	h.logger.Info(ctx, "list containers handler started",
		zap.String("handler", "ListContainers"),
		zap.String("project_slug", projectSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "ListContainers"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get project slug from URL parameter
	if projectSlug == "" {
		h.logger.Warn(ctx, "missing project slug parameter",
			zap.String("handler", "ListContainers"),
		)
		response.Error(c, containererrors.ErrInvalidProjectID, mapContainerError)
		return
	}

	// Get container slug from URL
	project, err := h.projectService.GetProjectBySlug(c.Request.Context(), projectSlug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "ListContainers"),
			zap.String("project_slug", projectSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	input := application.ListContainersInput{
		ProjectID: project.ProjectID(),
		UserID:    userID.(uint),
	}

	output, err := h.listContainersUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "list containers use case failed",
			zap.Error(err),
			zap.String("handler", "ListContainers"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "list containers handler completed",
		zap.String("handler", "ListContainers"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("project_id", project.ProjectID()),
		zap.Int("container_count", len(output.Containers)),
	)

	response.OK(c, output)
}

// AddEnvVarRequest represents the request body for adding an environment variable
type AddEnvVarRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// AddEnvVar handles adding an environment variable to a container
func (h *ContainerHandler) AddEnvVar(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "add env var handler started",
		zap.String("handler", "AddEnvVar"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "AddEnvVar"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "AddEnvVar"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	var req AddEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "AddEnvVar"),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.AddEnvVarInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		Key:         req.Key,
		Value:       req.Value,
	}

	output, err := h.addEnvVarUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "add env var use case failed",
			zap.Error(err),
			zap.String("handler", "AddEnvVar"),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", req.Key),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "add env var handler completed",
		zap.String("handler", "AddEnvVar"),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("key", req.Key),
	)

	response.Created(c, output)
}

// UpdateEnvVarRequest represents the request body for updating an environment variable
type UpdateEnvVarRequest struct {
	Value string `json:"value" binding:"required"`
}

// UpdateEnvVar handles updating an environment variable in a container
func (h *ContainerHandler) UpdateEnvVar(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")
	envVarKey := c.Param("key")

	h.logger.Info(ctx, "update env var handler started",
		zap.String("handler", "UpdateEnvVar"),
		zap.String("container_slug", containerSlug),
		zap.String("key", envVarKey),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "UpdateEnvVar"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "UpdateEnvVar"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Parse env var key from URL
	if envVarKey == "" {
		h.logger.Warn(ctx, "missing env var key parameter",
			zap.String("handler", "UpdateEnvVar"),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	var req UpdateEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "UpdateEnvVar"),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.UpdateEnvVarInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		EnvVarKey:   envVarKey,
		Value:       req.Value,
	}

	output, err := h.updateEnvVarUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "update env var use case failed",
			zap.Error(err),
			zap.String("handler", "UpdateEnvVar"),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", envVarKey),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "update env var handler completed",
		zap.String("handler", "UpdateEnvVar"),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("key", envVarKey),
	)

	response.OK(c, output)
}

// DeleteEnvVar handles deleting an environment variable from a container
func (h *ContainerHandler) DeleteEnvVar(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")
	key := c.Param("key")

	h.logger.Info(ctx, "delete env var handler started",
		zap.String("handler", "DeleteEnvVar"),
		zap.String("container_slug", containerSlug),
		zap.String("key", key),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "DeleteEnvVar"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "DeleteEnvVar"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Parse env var key from URL
	if key == "" {
		h.logger.Warn(ctx, "missing env var key parameter",
			zap.String("handler", "DeleteEnvVar"),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	input := application.DeleteEnvVarInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		Key:         key,
	}

	output, err := h.deleteEnvVarUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "delete env var use case failed",
			zap.Error(err),
			zap.String("handler", "DeleteEnvVar"),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", key),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "delete env var handler completed",
		zap.String("handler", "DeleteEnvVar"),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("key", key),
	)

	response.OK(c, output)
}

// AddNetworkRequest represents the request body for adding a network port mapping
type AddNetworkRequest struct {
	InternalPort *uint16 `json:"internal_port,omitempty"`
	ExternalPort *uint16 `json:"external_port,omitempty"`
	NetworkType  string  `json:"network_type" binding:"required"`
	ExternalIP   *string `json:"external_ip,omitempty"`
	FQDN         *string `json:"fqdn,omitempty"`
}

// AddNetwork handles adding a network port mapping to a container
func (h *ContainerHandler) AddNetwork(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")

	h.logger.Info(ctx, "add network handler started",
		zap.String("handler", "AddNetwork"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "AddNetwork"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "AddNetwork"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	var req AddNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "AddNetwork"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.AddNetworkInput{
		ContainerID:  container.ContainerID(),
		UserID:       userID.(uint),
		InternalPort: req.InternalPort,
		ExternalPort: req.ExternalPort,
		NetworkType:  req.NetworkType,
		ExternalIP:   req.ExternalIP,
		FQDN:         req.FQDN,
	}

	output, err := h.addNetworkUC.Execute(c.Request.Context(), input)
	if err != nil {
		errorFields := []zap.Field{
			zap.Error(err),
			zap.String("handler", "AddNetwork"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
		}
		if req.InternalPort != nil {
			errorFields = append(errorFields, zap.Uint16("internal_port", *req.InternalPort))
		}
		h.logger.Error(ctx, "add network use case failed", errorFields...)
		response.Error(c, err, mapContainerError)
		return
	}

	infoFields := []zap.Field{
		zap.String("handler", "AddNetwork"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Uint("network_id", output.NetworkID),
	}
	if req.InternalPort != nil {
		infoFields = append(infoFields, zap.Uint16("internal_port", *req.InternalPort))
	}
	h.logger.Info(ctx, "add network handler completed", infoFields...)

	response.Created(c, output)
}

// DeleteNetwork handles deleting a network port mapping from a container
func (h *ContainerHandler) DeleteNetwork(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")
	portStr := c.Param("port")

	h.logger.Info(ctx, "delete network handler started",
		zap.String("handler", "DeleteNetwork"),
		zap.String("container_slug", containerSlug),
		zap.String("port", portStr),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "DeleteNetwork"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "DeleteNetwork"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Parse internal port from URL
	if portStr == "" {
		h.logger.Warn(ctx, "missing port parameter",
			zap.String("handler", "DeleteNetwork"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		h.logger.Warn(ctx, "invalid port format",
			zap.Error(err),
			zap.String("handler", "DeleteNetwork"),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("port", portStr),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	input := application.DeleteNetworkInput{
		ContainerID:  container.ContainerID(),
		UserID:       userID.(uint),
		InternalPort: uint16(port),
	}

	output, err := h.deleteNetworkUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "delete network use case failed",
			zap.Error(err),
			zap.String("handler", "DeleteNetwork"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.Uint16("internal_port", uint16(port)),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "delete network handler completed",
		zap.String("handler", "DeleteNetwork"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Uint16("internal_port", uint16(port)),
	)

	response.OK(c, output)
}

// ListNetworks handles fetching all network port mappings for a container
func (h *ContainerHandler) ListNetworks(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	// Get container to verify ownership and get networks
	input := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}

	output, err := h.getContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	// Return just the networks
	response.OK(c, gin.H{"networks": output.Networks})
}

// ListEnvVars handles fetching all environment variables for a container
func (h *ContainerHandler) ListEnvVars(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	// Get container to verify ownership and get env vars
	input := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}

	output, err := h.getContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	// Return just the environment variables
	response.OK(c, gin.H{"env_vars": output.EnvVars})
}

// AddSecretRequest represents the request body for adding a secret
type AddSecretRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// AddSecret handles adding a secret to a container
func (h *ContainerHandler) AddSecret(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")

	h.logger.Info(ctx, "add secret handler started",
		zap.String("handler", "AddSecret"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "AddSecret"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "AddSecret"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	var req AddSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "AddSecret"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.AddSecretInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		Key:         req.Key,
		Value:       req.Value,
	}

	output, err := h.addSecretUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "add secret use case failed",
			zap.Error(err),
			zap.String("handler", "AddSecret"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", req.Key),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "add secret handler completed",
		zap.String("handler", "AddSecret"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Uint("secret_id", output.SecretID),
		zap.String("key", req.Key),
	)

	response.Created(c, output)
}

// UpdateSecretRequest represents the request body for updating a secret
type UpdateSecretRequest struct {
	Value string `json:"value" binding:"required"`
}

// UpdateSecret handles updating an existing secret
func (h *ContainerHandler) UpdateSecret(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")
	key := c.Param("key")

	h.logger.Info(ctx, "update secret handler started",
		zap.String("handler", "UpdateSecret"),
		zap.String("container_slug", containerSlug),
		zap.String("key", key),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "UpdateSecret"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "UpdateSecret"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Parse secret key from URL
	if key == "" {
		h.logger.Warn(ctx, "missing key parameter",
			zap.String("handler", "UpdateSecret"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	var req UpdateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "UpdateSecret"),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", key),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.UpdateSecretInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		Key:         key,
		Value:       req.Value,
	}

	output, err := h.updateSecretUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "update secret use case failed",
			zap.Error(err),
			zap.String("handler", "UpdateSecret"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", key),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "update secret handler completed",
		zap.String("handler", "UpdateSecret"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("key", key),
	)

	response.OK(c, output)
}

// DeleteSecret handles deleting a secret from a container
func (h *ContainerHandler) DeleteSecret(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")
	key := c.Param("key")

	h.logger.Info(ctx, "delete secret handler started",
		zap.String("handler", "DeleteSecret"),
		zap.String("container_slug", containerSlug),
		zap.String("key", key),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "DeleteSecret"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "DeleteSecret"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Parse secret key from URL
	if key == "" {
		h.logger.Warn(ctx, "missing key parameter",
			zap.String("handler", "DeleteSecret"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	input := application.DeleteSecretInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		Key:         key,
	}

	output, err := h.deleteSecretUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "delete secret use case failed",
			zap.Error(err),
			zap.String("handler", "DeleteSecret"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", key),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "delete secret handler completed",
		zap.String("handler", "DeleteSecret"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("key", key),
	)

	response.OK(c, output)
}

// ListSecrets handles fetching all secrets for a container
func (h *ContainerHandler) ListSecrets(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	// Get container to verify ownership and get secrets
	input := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}

	output, err := h.getContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	// Return just the secrets
	response.OK(c, gin.H{"secrets": output.Secrets})
}

// AddMountRequest represents the request body for adding a mount
type AddMountRequest struct {
	VolumeID  uint   `json:"volume_id" binding:"required"`
	MountPath string `json:"mount_path" binding:"required"`
}

// AddMount handles adding a volume mount to a container
func (h *ContainerHandler) AddMount(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")

	h.logger.Info(ctx, "add mount handler started",
		zap.String("handler", "AddMount"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "AddMount"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "AddMount"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Bind request body
	var req AddMountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "AddMount"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Prepare input
	input := application.AddMountInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		VolumeID:    req.VolumeID,
		MountPath:   req.MountPath,
	}

	output, err := h.addMountUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "add mount use case failed",
			zap.Error(err),
			zap.String("handler", "AddMount"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.Uint("volume_id", req.VolumeID),
			zap.String("mount_path", req.MountPath),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "add mount handler completed",
		zap.String("handler", "AddMount"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Uint("volume_id", req.VolumeID),
		zap.String("mount_path", req.MountPath),
	)

	response.Created(c, output)
}

// DeleteMount handles deleting a volume mount from a container
func (h *ContainerHandler) DeleteMount(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")
	volumeIDStr := c.Param("volume_id")

	h.logger.Info(ctx, "delete mount handler started",
		zap.String("handler", "DeleteMount"),
		zap.String("container_slug", containerSlug),
		zap.String("volume_id", volumeIDStr),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "DeleteMount"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "DeleteMount"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Parse volume ID from URL
	if volumeIDStr == "" {
		h.logger.Warn(ctx, "missing volume_id parameter",
			zap.String("handler", "DeleteMount"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	volumeID, err := strconv.ParseUint(volumeIDStr, 10, 32)
	if err != nil {
		h.logger.Warn(ctx, "invalid volume_id format",
			zap.Error(err),
			zap.String("handler", "DeleteMount"),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("volume_id", volumeIDStr),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Prepare input
	input := application.DeleteMountInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		VolumeID:    uint(volumeID),
	}

	output, err := h.deleteMountUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "delete mount use case failed",
			zap.Error(err),
			zap.String("handler", "DeleteMount"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.Uint("volume_id", uint(volumeID)),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "delete mount handler completed",
		zap.String("handler", "DeleteMount"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Uint("volume_id", uint(volumeID)),
	)

	response.OK(c, output)
}

// ============================================================================
// Build Variable Handlers
// ============================================================================

// ListBuildVars handles fetching all build variables for a container
func (h *ContainerHandler) ListBuildVars(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	// Get container to verify ownership and get build variables
	input := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}

	output, err := h.getContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	// Return just the build variables
	response.OK(c, gin.H{"build_vars": output.BuildVars})
}

// AddBuildVarRequest represents the request body for adding a build variable
type AddBuildVarRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// AddBuildVar handles adding a build variable to a container
func (h *ContainerHandler) AddBuildVar(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")

	h.logger.Info(ctx, "add build var handler started",
		zap.String("handler", "AddBuildVar"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "AddBuildVar"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "AddBuildVar"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	var req AddBuildVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "AddBuildVar"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.AddBuildVarInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		Key:         req.Key,
		Value:       req.Value,
	}

	output, err := h.addBuildVarUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "add build var use case failed",
			zap.Error(err),
			zap.String("handler", "AddBuildVar"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", req.Key),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "add build var handler completed",
		zap.String("handler", "AddBuildVar"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Uint("build_var_id", output.BuildVarID),
		zap.String("key", req.Key),
	)

	response.Created(c, output)
}

// UpdateBuildVarRequest represents the request body for updating a build variable
type UpdateBuildVarRequest struct {
	Value string `json:"value" binding:"required"`
}

// UpdateBuildVar handles updating a build variable in a container
func (h *ContainerHandler) UpdateBuildVar(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")
	buildVarKey := c.Param("key")

	h.logger.Info(ctx, "update build var handler started",
		zap.String("handler", "UpdateBuildVar"),
		zap.String("container_slug", containerSlug),
		zap.String("key", buildVarKey),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "UpdateBuildVar"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "UpdateBuildVar"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Parse build var key from URL
	if buildVarKey == "" {
		h.logger.Warn(ctx, "missing key parameter",
			zap.String("handler", "UpdateBuildVar"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	var req UpdateBuildVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "UpdateBuildVar"),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", buildVarKey),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.UpdateBuildVarInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		BuildVarKey: buildVarKey,
		Value:       req.Value,
	}

	output, err := h.updateBuildVarUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "update build var use case failed",
			zap.Error(err),
			zap.String("handler", "UpdateBuildVar"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", buildVarKey),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "update build var handler completed",
		zap.String("handler", "UpdateBuildVar"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("key", buildVarKey),
	)

	response.OK(c, output)
}

// DeleteBuildVar handles deleting a build variable from a container
func (h *ContainerHandler) DeleteBuildVar(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("container_slug")
	key := c.Param("key")

	h.logger.Info(ctx, "delete build var handler started",
		zap.String("handler", "DeleteBuildVar"),
		zap.String("container_slug", containerSlug),
		zap.String("key", key),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "DeleteBuildVar"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "DeleteBuildVar"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Parse build var key from URL
	if key == "" {
		h.logger.Warn(ctx, "missing key parameter",
			zap.String("handler", "DeleteBuildVar"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	input := application.DeleteBuildVarInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		Key:         key,
	}

	output, err := h.deleteBuildVarUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "delete build var use case failed",
			zap.Error(err),
			zap.String("handler", "DeleteBuildVar"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.String("key", key),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "delete build var handler completed",
		zap.String("handler", "DeleteBuildVar"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("key", key),
	)

	response.OK(c, output)
}
