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
	createContainerUC    *application.CreateContainerUseCase
	getContainerUC       *application.GetContainerUseCase
	updateContainerUC    *application.UpdateContainerUseCase
	deleteContainerUC    *application.DeleteContainerUseCase
	listContainersUC     *application.ListContainersUseCase
	addEnvVarUC          *application.AddEnvVarUseCase
	updateEnvVarUC       *application.UpdateEnvVarUseCase
	deleteEnvVarUC       *application.DeleteEnvVarUseCase
	addNetworkUC         *application.AddNetworkUseCase
	updateNetworkUC      *application.UpdateNetworkUseCase
	deleteNetworkUC      *application.DeleteNetworkUseCase
	addSecretUC          *application.AddSecretUseCase
	updateSecretUC       *application.UpdateSecretUseCase
	deleteSecretUC       *application.DeleteSecretUseCase
	addBuildVarUC        *application.AddBuildVarUseCase
	updateBuildVarUC     *application.UpdateBuildVarUseCase
	deleteBuildVarUC     *application.DeleteBuildVarUseCase
	addMountUC           *application.AddMountUseCase
	deleteMountUC        *application.DeleteMountUseCase
	checkFQDNUC          *application.CheckFQDNUseCase
	checkContainerNameUC *application.CheckContainerNameUseCase
	createNodePortUC     *application.CreateNodePortUseCase
	getNodePortUC        *application.GetNodePortUseCase
	projectService       projectservice.ProjectService
	volumeService        projectservice.VolumeService
	containerService     containerservice.ContainerService
	permissionSvc        containerservice.PermissionService
	logger               logger.Logger
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
	updateNetworkUC *application.UpdateNetworkUseCase,
	deleteNetworkUC *application.DeleteNetworkUseCase,
	addSecretUC *application.AddSecretUseCase,
	updateSecretUC *application.UpdateSecretUseCase,
	deleteSecretUC *application.DeleteSecretUseCase,
	addBuildVarUC *application.AddBuildVarUseCase,
	updateBuildVarUC *application.UpdateBuildVarUseCase,
	deleteBuildVarUC *application.DeleteBuildVarUseCase,
	addMountUC *application.AddMountUseCase,
	deleteMountUC *application.DeleteMountUseCase,
	checkFQDNUC *application.CheckFQDNUseCase,
	checkContainerNameUC *application.CheckContainerNameUseCase,
	createNodePortUC *application.CreateNodePortUseCase,
	getNodePortUC *application.GetNodePortUseCase,
	projectService projectservice.ProjectService,
	volumeService projectservice.VolumeService,
	containerService containerservice.ContainerService,
	permissionSvc containerservice.PermissionService,
	log logger.Logger,
) *ContainerHandler {
	return &ContainerHandler{
		createContainerUC:    createContainerUC,
		getContainerUC:       getContainerUC,
		updateContainerUC:    updateContainerUC,
		deleteContainerUC:    deleteContainerUC,
		listContainersUC:     listContainersUC,
		addEnvVarUC:          addEnvVarUC,
		updateEnvVarUC:       updateEnvVarUC,
		deleteEnvVarUC:       deleteEnvVarUC,
		addNetworkUC:         addNetworkUC,
		updateNetworkUC:      updateNetworkUC,
		deleteNetworkUC:      deleteNetworkUC,
		addSecretUC:          addSecretUC,
		updateSecretUC:       updateSecretUC,
		deleteSecretUC:       deleteSecretUC,
		addBuildVarUC:        addBuildVarUC,
		updateBuildVarUC:     updateBuildVarUC,
		deleteBuildVarUC:     deleteBuildVarUC,
		addMountUC:           addMountUC,
		deleteMountUC:        deleteMountUC,
		checkFQDNUC:          checkFQDNUC,
		checkContainerNameUC: checkContainerNameUC,
		createNodePortUC:     createNodePortUC,
		getNodePortUC:        getNodePortUC,
		projectService:       projectService,
		volumeService:        volumeService,
		containerService:     containerService,
		permissionSvc:        permissionSvc,
		logger:               log,
	}
}

// VolumeToCreate represents a volume to be created with the container
type VolumeToCreate struct {
	Name      string `json:"name" binding:"required"`
	Capacity  uint32 `json:"capacity" binding:"required"`
	MountPath string `json:"mount_path" binding:"required"`
}

// NetworkToCreate represents a network to be created with the container
type NetworkToCreate struct {
	InternalPort uint16  `json:"internal_port" binding:"required"`
	NetworkType  string  `json:"network_type" binding:"required"`
	FQDN         *string `json:"fqdn,omitempty"`
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
	Networks             []NetworkToCreate      `json:"networks,omitempty"`
	EnvVars              []EnvVarToCreate       `json:"env_vars,omitempty"`
	Secrets              []EnvVarToCreate       `json:"secrets,omitempty"`
	BuildVars            []EnvVarToCreate       `json:"build_vars,omitempty"`
}

// EnvVarToCreate represents an environment variable to create
type EnvVarToCreate struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
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

	// Convert networks to application layer format
	var networks []application.NetworkToCreate
	for _, n := range req.Networks {
		networks = append(networks, application.NetworkToCreate{
			InternalPort: n.InternalPort,
			NetworkType:  n.NetworkType,
			FQDN:         n.FQDN,
		})
	}

	// Convert env vars to application layer format
	var envVars []application.EnvVarToCreate
	for _, e := range req.EnvVars {
		envVars = append(envVars, application.EnvVarToCreate{
			Key:   e.Key,
			Value: e.Value,
		})
	}

	// Convert secrets to application layer format
	var secrets []application.EnvVarToCreate
	for _, s := range req.Secrets {
		secrets = append(secrets, application.EnvVarToCreate{
			Key:   s.Key,
			Value: s.Value,
		})
	}

	// Convert build vars to application layer format
	var buildVars []application.EnvVarToCreate
	for _, b := range req.BuildVars {
		buildVars = append(buildVars, application.EnvVarToCreate{
			Key:   b.Key,
			Value: b.Value,
		})
	}

	input := application.CreateContainerInput{
		ProjectID:            projectID,
		UserID:               userID.(uint),
		Name:                 req.Name,
		GitURL:               req.GitURL,
		GitBranch:            req.GitBranch,
		GitSubpath:           gitDirectory,
		GitHubInstallationID: req.GitHubInstallationID,
		CPULimit:             cpuLimit,
		MemoryLimit:          memoryLimit,
		TemplateID:           req.TemplateID,
		TemplateConfig:       req.TemplateConfig,
		Volumes:              volumes,
		Networks:             networks,
		EnvVars:              envVars,
		Secrets:              secrets,
		BuildVars:            buildVars,
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
	GitDirectory              *string                `json:"git_directory,omitempty"` // Deprecated: use GitSubpath instead
	GitSubpath                *string                `json:"git_subpath,omitempty"`
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

	// GitSubpath is the same as GitDirectory (backward compatibility)
	gitSubpath := req.GitSubpath
	if req.GitSubpath == nil && req.GitDirectory != nil {
		gitSubpath = req.GitDirectory
	}

	input := application.UpdateContainerInput{
		ContainerID:                containerID,
		UserID:                     userID.(uint),
		Name:                       req.Name,
		StableWindow:               req.StableWindow,
		GitHubInstallationID:       githubInstallationID,
		UpdateGitHubInstallationID: updateGitHubInstallation,
		GitURL:                     req.GitURL,
		GitBranch:                  req.GitBranch,
		GitSubpath:                 gitSubpath,
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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: output.ContainerID,
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", output.ContainerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: output.ContainerID,
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", output.ContainerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.Created(c, fullContainer)
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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: output.ContainerID,
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", output.ContainerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
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

	_, err = h.deleteEnvVarUC.Execute(c.Request.Context(), input)
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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
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
	containerSlug := c.Param("slug")

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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: output.ContainerID,
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", output.ContainerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.Created(c, fullContainer)
}

// DeleteNetwork handles deleting a network port mapping from a container
func (h *ContainerHandler) DeleteNetwork(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")
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

	_, err = h.deleteNetworkUC.Execute(c.Request.Context(), input)
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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
}

// UpdateNetworkRequest represents the request body for updating a network
type UpdateNetworkRequest struct {
	InternalPort *uint16 `json:"internal_port,omitempty"`
	NetworkType  string  `json:"network_type,omitempty"`
	FQDN         *string `json:"fqdn,omitempty"`
}

// UpdateNetwork handles updating a network port mapping
func (h *ContainerHandler) UpdateNetwork(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")
	networkIDStr := c.Param("network_id")

	h.logger.Info(ctx, "update network handler started",
		zap.String("handler", "UpdateNetwork"),
		zap.String("container_slug", containerSlug),
		zap.String("network_id", networkIDStr),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "UpdateNetwork"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse network ID
	networkID, err := strconv.ParseUint(networkIDStr, 10, 32)
	if err != nil {
		h.logger.Warn(ctx, "invalid network ID parameter",
			zap.Error(err),
			zap.String("handler", "UpdateNetwork"),
			zap.String("network_id_str", networkIDStr),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "UpdateNetwork"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Bind request
	var req UpdateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "UpdateNetwork"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	// Execute use case
	input := application.UpdateNetworkInput{
		ContainerID:  container.ContainerID(),
		UserID:       userID.(uint),
		NetworkID:    uint(networkID),
		InternalPort: req.InternalPort,
		NetworkType:  req.NetworkType,
		FQDN:         req.FQDN,
	}

	output, err := h.updateNetworkUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "update network use case failed",
			zap.Error(err),
			zap.String("handler", "UpdateNetwork"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
			zap.Uint64("network_id", networkID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "update network handler completed",
		zap.String("handler", "UpdateNetwork"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Uint("network_id", output.NetworkID),
	)

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: output.ContainerID,
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", output.ContainerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
}

// ListNetworks handles fetching all network port mappings for a container
func (h *ContainerHandler) ListNetworks(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "list networks handler started",
		zap.String("handler", "ListNetworks"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "ListNetworks"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "ListNetworks"),
		)
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
		h.logger.Error(ctx, "get container use case failed",
			zap.Error(err),
			zap.String("handler", "ListNetworks"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "list networks handler completed",
		zap.String("handler", "ListNetworks"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Int("count", len(output.Networks)),
	)

	// Return just the networks
	response.OK(c, gin.H{"networks": output.Networks})
}

// ListEnvVars handles fetching all environment variables for a container
func (h *ContainerHandler) ListEnvVars(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "list env vars handler started",
		zap.String("handler", "ListEnvVars"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "ListEnvVars"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "ListEnvVars"),
		)
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
		h.logger.Error(ctx, "get container use case failed",
			zap.Error(err),
			zap.String("handler", "ListEnvVars"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "list env vars handler completed",
		zap.String("handler", "ListEnvVars"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Int("count", len(output.EnvVars)),
	)

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
	containerSlug := c.Param("slug")

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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: output.ContainerID,
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", output.ContainerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.Created(c, fullContainer)
}

// UpdateSecretRequest represents the request body for updating a secret
type UpdateSecretRequest struct {
	Value string `json:"value" binding:"required"`
}

// UpdateSecret handles updating an existing secret
func (h *ContainerHandler) UpdateSecret(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")
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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: output.ContainerID,
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", output.ContainerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
}

// DeleteSecret handles deleting a secret from a container
func (h *ContainerHandler) DeleteSecret(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")
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

	_, err = h.deleteSecretUC.Execute(c.Request.Context(), input)
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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
}

// ListSecrets handles fetching all secrets for a container
func (h *ContainerHandler) ListSecrets(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "list secrets handler started",
		zap.String("handler", "ListSecrets"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "ListSecrets"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "ListSecrets"),
		)
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
		h.logger.Error(ctx, "get container use case failed",
			zap.Error(err),
			zap.String("handler", "ListSecrets"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "list secrets handler completed",
		zap.String("handler", "ListSecrets"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Int("count", len(output.Secrets)),
	)

	// Return just the secrets
	response.OK(c, gin.H{"secrets": output.Secrets})
}

// ============================================================================
// Build Variable Handlers
// ============================================================================

// ListBuildVars handles fetching all build variables for a container
func (h *ContainerHandler) ListBuildVars(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "list build vars handler started",
		zap.String("handler", "ListBuildVars"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "ListBuildVars"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "ListBuildVars"),
		)
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
		h.logger.Error(ctx, "get container use case failed",
			zap.Error(err),
			zap.String("handler", "ListBuildVars"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "list build vars handler completed",
		zap.String("handler", "ListBuildVars"),
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
		zap.Int("count", len(output.BuildVars)),
	)

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
	containerSlug := c.Param("slug")

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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: output.ContainerID,
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", output.ContainerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.Created(c, fullContainer)
}

// UpdateBuildVarRequest represents the request body for updating a build variable
type UpdateBuildVarRequest struct {
	Value string `json:"value" binding:"required"`
}

// UpdateBuildVar handles updating a build variable in a container
func (h *ContainerHandler) UpdateBuildVar(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")
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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: output.ContainerID,
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", output.ContainerID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
}

// DeleteBuildVar handles deleting a build variable from a container
func (h *ContainerHandler) DeleteBuildVar(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")
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

	_, err = h.deleteBuildVarUC.Execute(c.Request.Context(), input)
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

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
}

// ====================
// Volume Management (Container sub-resource)
// ====================

// VolumeResponse represents a volume with mount information
type VolumeResponse struct {
	VolumeID  uint   `json:"volume_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Capacity  uint32 `json:"capacity"`
	MountPath string `json:"mount_path"`
	CreatedAt string `json:"created_at"`
}

// ListVolumes handles listing all volumes mounted to a container
// GET /api/v1/containers/:slug/volumes
func (h *ContainerHandler) ListVolumes(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "list volumes handler started",
		zap.String("handler", "ListVolumes"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "ListVolumes"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "ListVolumes"),
			zap.String("container_slug", containerSlug),
			zap.Uint("user_id", userID.(uint)),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Check permission using GetContainerUseCase (security fix - consistent with ListEnvVars, ListSecrets pattern)
	input := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}

	_, err = h.getContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "permission check failed",
			zap.Error(err),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Build response with volume and mount information
	volumes := make([]VolumeResponse, 0, len(container.Mounts()))
	for _, mount := range container.Mounts() {
		// Get volume details from volume service
		volume, err := h.volumeService.GetVolume(c.Request.Context(), mount.VolumeID())
		if err != nil {
			h.logger.Error(ctx, "failed to get volume details",
				zap.Error(err),
				zap.Uint("volume_id", mount.VolumeID()),
			)
			// Skip volumes that were deleted
			continue
		}

		volumes = append(volumes, VolumeResponse{
			VolumeID:  volume.VolumeID(),
			Name:      volume.Name(),
			Slug:      volume.Slug().String(),
			Capacity:  volume.Capacity(),
			MountPath: mount.MountPath(),
			CreatedAt: volume.CreatedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	h.logger.Info(ctx, "list volumes handler completed",
		zap.String("handler", "ListVolumes"),
		zap.Uint("container_id", container.ContainerID()),
		zap.Int("volume_count", len(volumes)),
	)

	response.OK(c, map[string]interface{}{
		"volumes": volumes,
	})
}

// AddVolumeRequest represents the request body for adding a volume to a container
type AddVolumeRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=32"`
	Capacity  uint32 `json:"capacity" binding:"required,min=128,max=2048"`
	MountPath string `json:"mount_path" binding:"required"`
}

// AddVolume handles creating a new volume and mounting it to a container
// POST /api/v1/containers/:slug/volumes
func (h *ContainerHandler) AddVolume(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")

	h.logger.Info(ctx, "add volume handler started",
		zap.String("handler", "AddVolume"),
		zap.String("container_slug", containerSlug),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "AddVolume"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "AddVolume"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Step 0: Check permission BEFORE creating volume (security fix)
	if err := h.permissionSvc.CanUserModifyContainer(c.Request.Context(), userID.(uint), container.ContainerID()); err != nil {
		h.logger.Warn(ctx, "permission check failed",
			zap.Error(err),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	var req AddVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "AddVolume"),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError, response.WithDetails(map[string]any{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	// Step 1: Create volume using volume service
	volume, err := h.volumeService.CreateVolume(c.Request.Context(), container.ProjectID(), req.Name, req.Capacity)
	if err != nil {
		h.logger.Error(ctx, "failed to create volume",
			zap.Error(err),
			zap.String("handler", "AddVolume"),
			zap.Uint("project_id", container.ProjectID()),
			zap.String("volume_name", req.Name),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Step 2: Mount volume to container using AddMountUseCase
	mountInput := application.AddMountInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		VolumeID:    volume.VolumeID(),
		MountPath:   req.MountPath,
	}

	_, err = h.addMountUC.Execute(c.Request.Context(), mountInput)
	if err != nil {
		h.logger.Error(ctx, "failed to mount volume",
			zap.Error(err),
			zap.String("handler", "AddVolume"),
			zap.Uint("container_id", container.ContainerID()),
			zap.Uint("volume_id", volume.VolumeID()),
		)

		// Rollback: Delete orphan volume on mount failure
		if deleteErr := h.volumeService.DeleteVolume(c.Request.Context(), volume.VolumeID()); deleteErr != nil {
			h.logger.Error(ctx, "failed to cleanup orphan volume after mount failure",
				zap.Error(deleteErr),
				zap.Uint("volume_id", volume.VolumeID()),
				zap.NamedError("original_error", err),
			)
		}

		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "add volume handler completed",
		zap.String("handler", "AddVolume"),
		zap.Uint("container_id", container.ContainerID()),
		zap.Uint("volume_id", volume.VolumeID()),
		zap.String("mount_path", req.MountPath),
	)

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.Created(c, fullContainer)
}

// DeleteVolume handles deleting a volume from a container
// DELETE /api/v1/containers/:slug/volumes/:volume_id
func (h *ContainerHandler) DeleteVolume(c *gin.Context) {
	ctx := c.Request.Context()
	containerSlug := c.Param("slug")
	volumeIDStr := c.Param("volume_id")

	h.logger.Info(ctx, "delete volume handler started",
		zap.String("handler", "DeleteVolume"),
		zap.String("container_slug", containerSlug),
		zap.String("volume_id", volumeIDStr),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "DeleteVolume"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse volume ID
	volumeID, err := strconv.ParseUint(volumeIDStr, 10, 32)
	if err != nil {
		h.logger.Warn(ctx, "invalid volume ID parameter",
			zap.Error(err),
			zap.String("handler", "DeleteVolume"),
			zap.String("volume_id_str", volumeIDStr),
		)
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "DeleteVolume"),
			zap.String("container_slug", containerSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Step 1: Unmount volume using DeleteMountUseCase
	unmountInput := application.DeleteMountInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
		VolumeID:    uint(volumeID),
	}

	_, err = h.deleteMountUC.Execute(c.Request.Context(), unmountInput)
	if err != nil {
		h.logger.Error(ctx, "failed to unmount volume",
			zap.Error(err),
			zap.String("handler", "DeleteVolume"),
			zap.Uint("container_id", container.ContainerID()),
			zap.Uint64("volume_id", volumeID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Step 2: Delete volume using volume service
	err = h.volumeService.DeleteVolume(c.Request.Context(), uint(volumeID))
	if err != nil {
		h.logger.Error(ctx, "failed to delete volume",
			zap.Error(err),
			zap.String("handler", "DeleteVolume"),
			zap.Uint64("volume_id", volumeID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "delete volume handler completed",
		zap.String("handler", "DeleteVolume"),
		zap.Uint("container_id", container.ContainerID()),
		zap.Uint64("volume_id", volumeID),
	)

	// Get full container details to return complete information
	getInput := application.GetContainerInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}
	fullContainer, err := h.getContainerUC.Execute(ctx, getInput)
	if err != nil {
		h.logger.Error(ctx, "failed to get updated container details",
			zap.Error(err),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, fullContainer)
}

// CheckFQDN handles checking if an FQDN is already in use
func (h *ContainerHandler) CheckFQDN(c *gin.Context) {
	ctx := c.Request.Context()
	fqdn := c.Query("fqdn")
	projectIDStr := c.Query("project_id")

	h.logger.Debug(ctx, "check FQDN handler started",
		zap.String("handler", "CheckFQDN"),
		zap.String("fqdn", fqdn),
		zap.String("project_id", projectIDStr),
	)

	if fqdn == "" {
		h.logger.Warn(ctx, "missing FQDN parameter",
			zap.String("handler", "CheckFQDN"),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	// project_id is now required for accurate FQDN validation
	if projectIDStr == "" {
		h.logger.Warn(ctx, "missing project_id parameter",
			zap.String("handler", "CheckFQDN"),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		h.logger.Warn(ctx, "invalid project_id parameter",
			zap.String("handler", "CheckFQDN"),
			zap.String("project_id", projectIDStr),
			zap.Error(err),
		)
		response.Error(c, containererrors.ErrInvalidProjectID, mapContainerError)
		return
	}

	input := application.CheckFQDNInput{
		FQDN:      fqdn,
		ProjectID: uint32(projectID),
	}

	output, err := h.checkFQDNUC.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "failed to check FQDN",
			zap.Error(err),
			zap.String("handler", "CheckFQDN"),
			zap.String("fqdn", fqdn),
			zap.Uint32("project_id", input.ProjectID),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Debug(ctx, "check FQDN handler completed",
		zap.String("handler", "CheckFQDN"),
		zap.String("fqdn", fqdn),
		zap.Uint32("project_id", input.ProjectID),
		zap.Bool("exists", output.Exists),
	)

	response.OK(c, output)
}

// CheckContainerName handles checking if a container name is already in use within a project
func (h *ContainerHandler) CheckContainerName(c *gin.Context) {
	ctx := c.Request.Context()
	projectSlug := c.Param("slug")
	name := c.Query("name")

	h.logger.Debug(ctx, "check container name handler started",
		zap.String("handler", "CheckContainerName"),
		zap.String("project_slug", projectSlug),
		zap.String("name", name),
	)

	if projectSlug == "" || name == "" {
		h.logger.Warn(ctx, "missing required parameters",
			zap.String("handler", "CheckContainerName"),
			zap.String("project_slug", projectSlug),
			zap.String("name", name),
		)
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "CheckContainerName"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get project by slug to get project_id
	project, err := h.projectService.GetProjectBySlug(ctx, projectSlug)
	if err != nil {
		h.logger.Error(ctx, "failed to get project by slug",
			zap.Error(err),
			zap.String("handler", "CheckContainerName"),
			zap.String("project_slug", projectSlug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Check if user has permission to create containers in this project
	if err := h.permissionSvc.CanUserCreateContainer(ctx, userID.(uint), project.ProjectID()); err != nil {
		h.logger.Warn(ctx, "user does not have permission to create containers in project",
			zap.Error(err),
			zap.String("handler", "CheckContainerName"),
			zap.Uint("user_id", userID.(uint)),
			zap.Uint("project_id", project.ProjectID()),
		)
		response.Error(c, containererrors.ErrPermissionDenied, mapContainerError)
		return
	}

	input := application.CheckContainerNameInput{
		ProjectID: project.ProjectID(),
		Name:      name,
	}

	output, err := h.checkContainerNameUC.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "failed to check container name",
			zap.Error(err),
			zap.String("handler", "CheckContainerName"),
			zap.Uint("project_id", project.ProjectID()),
			zap.String("name", name),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Debug(ctx, "check container name handler completed",
		zap.String("handler", "CheckContainerName"),
		zap.Uint("project_id", project.ProjectID()),
		zap.String("name", name),
		zap.Bool("exists", output.Exists),
	)

	response.OK(c, output)
}

// CreateNodePort handles creating a temporary NodePort for TCP network
func (h *ContainerHandler) CreateNodePort(c *gin.Context) {
	ctx := c.Request.Context()

	h.logger.Info(ctx, "create nodeport handler started",
		zap.String("handler", "CreateNodePort"),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Error(ctx, "user ID not found in context",
			zap.String("handler", "CreateNodePort"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "CreateNodePort"),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Execute use case
	input := application.CreateNodePortInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}

	output, err := h.createNodePortUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "failed to create nodeport",
			zap.Error(err),
			zap.String("handler", "CreateNodePort"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "create nodeport handler completed",
		zap.String("handler", "CreateNodePort"),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("status", output.Status),
	)

	response.Accepted(c, output)
}

// GetNodePort handles retrieving NodePort status and connection info
func (h *ContainerHandler) GetNodePort(c *gin.Context) {
	ctx := c.Request.Context()

	h.logger.Info(ctx, "get nodeport handler started",
		zap.String("handler", "GetNodePort"),
	)

	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Error(ctx, "user ID not found in context",
			zap.String("handler", "GetNodePort"),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.getContainerBySlug(c)
	if err != nil {
		h.logger.Error(ctx, "failed to get container by slug",
			zap.Error(err),
			zap.String("handler", "GetNodePort"),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Execute use case
	input := application.GetNodePortInput{
		ContainerID: container.ContainerID(),
		UserID:      userID.(uint),
	}

	output, err := h.getNodePortUC.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "failed to get nodeport",
			zap.Error(err),
			zap.String("handler", "GetNodePort"),
			zap.Uint("container_id", container.ContainerID()),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "get nodeport handler completed",
		zap.String("handler", "GetNodePort"),
		zap.Uint("container_id", container.ContainerID()),
		zap.String("host", output.Host),
		zap.Int("port", output.Port),
		zap.String("status", output.Status),
	)

	response.OK(c, output)
}
