package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/container/application"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
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
	addMountUC        *application.AddMountUseCase
	deleteMountUC     *application.DeleteMountUseCase
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
	addMountUC *application.AddMountUseCase,
	deleteMountUC *application.DeleteMountUseCase,
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
		addMountUC:        addMountUC,
		deleteMountUC:     deleteMountUC,
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
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get project ID from URL parameter
	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidProjectID, mapContainerError)
		return
	}

	var req CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
		ProjectID:            uint(projectID),
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
		response.Error(c, err, mapContainerError)
		return
	}

	resp := ContainerResponse{
		ContainerID: output.ContainerID,
		ProjectID:   output.ProjectID,
		Name:        output.Name,
		Slug:        output.Slug,
		CreatedAt:   output.CreatedAt,
	}

	response.Created(c, resp)
}

// GetContainer handles fetching a container by ID
func (h *ContainerHandler) GetContainer(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	input := application.GetContainerInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
	}

	output, err := h.getContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

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
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	var req UpdateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
		ContainerID:                uint(containerID),
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
		response.Error(c, err, mapContainerError)
		return
	}

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
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	input := application.DeleteContainerInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
	}

	output, err := h.deleteContainerUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, output)
}

// ListContainers handles fetching all containers for a project
func (h *ContainerHandler) ListContainers(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse project ID from URL parameter
	projectIDStr := c.Param("id")
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	input := application.ListContainersInput{
		ProjectID: uint(projectID),
		UserID:    userID.(uint),
	}

	output, err := h.listContainersUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, output)
}

// AddEnvVarRequest represents the request body for adding an environment variable
type AddEnvVarRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// AddEnvVar handles adding an environment variable to a container
func (h *ContainerHandler) AddEnvVar(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	var req AddEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.AddEnvVarInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
		Key:         req.Key,
		Value:       req.Value,
	}

	output, err := h.addEnvVarUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.Created(c, output)
}

// UpdateEnvVarRequest represents the request body for updating an environment variable
type UpdateEnvVarRequest struct {
	Value string `json:"value" binding:"required"`
}

// UpdateEnvVar handles updating an environment variable in a container
func (h *ContainerHandler) UpdateEnvVar(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Parse env var key from URL
	envVarKey := c.Param("key")
	if envVarKey == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	var req UpdateEnvVarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.UpdateEnvVarInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
		EnvVarKey:   envVarKey,
		Value:       req.Value,
	}

	output, err := h.updateEnvVarUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, output)
}

// DeleteEnvVar handles deleting an environment variable from a container
func (h *ContainerHandler) DeleteEnvVar(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Parse env var key from URL
	key := c.Param("key")
	if key == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	input := application.DeleteEnvVarInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
		Key:         key,
	}

	output, err := h.deleteEnvVarUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

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
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	var req AddNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.AddNetworkInput{
		ContainerID:  uint(containerID),
		UserID:       userID.(uint),
		InternalPort: req.InternalPort,
		ExternalPort: req.ExternalPort,
		NetworkType:  req.NetworkType,
		ExternalIP:   req.ExternalIP,
		FQDN:         req.FQDN,
	}

	output, err := h.addNetworkUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.Created(c, output)
}

// DeleteNetwork handles deleting a network port mapping from a container
func (h *ContainerHandler) DeleteNetwork(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Parse internal port from URL
	portStr := c.Param("port")
	if portStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	input := application.DeleteNetworkInput{
		ContainerID:  uint(containerID),
		UserID:       userID.(uint),
		InternalPort: uint16(port),
	}

	output, err := h.deleteNetworkUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

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

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Get container to verify ownership and get networks
	input := application.GetContainerInput{
		ContainerID: uint(containerID),
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

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Get container to verify ownership and get env vars
	input := application.GetContainerInput{
		ContainerID: uint(containerID),
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
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	var req AddSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.AddSecretInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
		Key:         req.Key,
		Value:       req.Value,
	}

	output, err := h.addSecretUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.Created(c, output)
}

// UpdateSecretRequest represents the request body for updating a secret
type UpdateSecretRequest struct {
	Value string `json:"value" binding:"required"`
}

// UpdateSecret handles updating an existing secret
func (h *ContainerHandler) UpdateSecret(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Parse secret key from URL
	key := c.Param("key")
	if key == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	var req UpdateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, containererrors.ErrValidationFailed, mapContainerError)
		return
	}

	input := application.UpdateSecretInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
		Key:         key,
		Value:       req.Value,
	}

	output, err := h.updateSecretUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, output)
}

// DeleteSecret handles deleting a secret from a container
func (h *ContainerHandler) DeleteSecret(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Parse secret key from URL
	key := c.Param("key")
	if key == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	input := application.DeleteSecretInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
		Key:         key,
	}

	output, err := h.deleteSecretUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

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

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Get container to verify ownership and get secrets
	input := application.GetContainerInput{
		ContainerID: uint(containerID),
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
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Bind request body
	var req AddMountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Prepare input
	input := application.AddMountInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
		VolumeID:    req.VolumeID,
		MountPath:   req.MountPath,
	}

	output, err := h.addMountUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.Created(c, output)
}

// DeleteMount handles deleting a volume mount from a container
func (h *ContainerHandler) DeleteMount(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Parse container ID from URL
	containerIDStr := c.Param("id")
	if containerIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	containerID, err := strconv.ParseUint(containerIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Parse volume ID from URL
	volumeIDStr := c.Param("volume_id")
	if volumeIDStr == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	volumeID, err := strconv.ParseUint(volumeIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidFormat, mapContainerError)
		return
	}

	// Prepare input
	input := application.DeleteMountInput{
		ContainerID: uint(containerID),
		UserID:      userID.(uint),
		VolumeID:    uint(volumeID),
	}

	output, err := h.deleteMountUC.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, output)
}
