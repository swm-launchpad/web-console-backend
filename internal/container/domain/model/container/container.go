package model

import (
	"time"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

// Container represents a container aggregate root
// It manages EnvVar and Network entities and enforces domain invariants
type Container struct {
	// Private fields (캡슐화)
	containerID            uint
	projectID              uint
	templateID             *uint
	name                   string
	slug                   value.ContainerSlug
	stableWindow           *uint32
	templateConfig         map[string]interface{}
	githubInstallationID   *int64 // GitHub App installation ID for private repos
	gitConfig              value.GitConfig
	gitCommitHash          *string // Current/target git commit hash
	lastBuiltGitCommitHash *string // Last successfully built git commit hash
	resourceLimits         value.ResourceLimits
	monthlyBuildTime       *uint32
	monthlyBuildCount      *uint32
	monthlyUptime          *string   // Uptime percentage as string (e.g., "99.9%")
	envVars                []EnvVar  // Aggregate's internal entities
	networks               []Network // Aggregate's internal entities
	secrets                []Secret  // Aggregate's internal entities for sensitive environment variables
	mounts                 []Mount   // Aggregate's internal entities for volume mounts
	isDeleted              bool
	deletedAt              *time.Time
	createdAt              time.Time
	updatedAt              time.Time
}

var (
	// Error variables for aggregate invariants
	ErrDuplicateEnvVarKey    = containererrors.ErrDuplicateEnvVarKey
	ErrDuplicateInternalPort = containererrors.ErrDuplicateInternalPort
	ErrDuplicateSecretKey    = containererrors.ErrDuplicateSecretKey
)

const (
	MaxEnvVarsPerContainer  = 100
	MaxNetworksPerContainer = 20
	MaxSecretsPerContainer  = 100
	MaxMountsPerContainer   = 10
)

// NewContainer creates a new container with initial configuration
// templateID, templateConfig, and githubInstallationID are optional (pass nil if not needed)
func NewContainer(
	projectID uint,
	name string,
	slug value.ContainerSlug,
	gitConfig value.GitConfig,
	resourceLimits value.ResourceLimits,
	templateID *uint,
	templateConfig map[string]interface{},
	githubInstallationID *int64,
) (*Container, error) {
	if projectID == 0 {
		return nil, containererrors.ErrInvalidProjectID
	}
	if name == "" {
		return nil, containererrors.ErrNameRequired
	}
	// Name is for display purposes, allow up to 255 characters
	if len(name) > 255 {
		return nil, containererrors.ErrNameTooLong
	}

	return &Container{
		projectID:            projectID,
		templateID:           templateID,
		name:                 name,
		slug:                 slug,
		templateConfig:       templateConfig,
		githubInstallationID: githubInstallationID,
		gitConfig:            gitConfig,
		resourceLimits:       resourceLimits,
		envVars:              make([]EnvVar, 0),
		networks:             make([]Network, 0),
		secrets:              make([]Secret, 0),
		mounts:               make([]Mount, 0),
		isDeleted:            false,
		createdAt:            time.Now(),
		updatedAt:            time.Time{}, // Zero time for new containers (NULL in database)
	}, nil
}

// Getter methods
func (c *Container) ContainerID() uint                      { return c.containerID }
func (c *Container) ProjectID() uint                        { return c.projectID }
func (c *Container) TemplateID() *uint                      { return c.templateID }
func (c *Container) Name() string                           { return c.name }
func (c *Container) Slug() value.ContainerSlug              { return c.slug }
func (c *Container) StableWindow() *uint32                  { return c.stableWindow }
func (c *Container) TemplateConfig() map[string]interface{} { return c.templateConfig }
func (c *Container) GitHubInstallationID() *int64           { return c.githubInstallationID }
func (c *Container) GitConfig() value.GitConfig             { return c.gitConfig }
func (c *Container) LastBuiltGitCommitHash() *string        { return c.lastBuiltGitCommitHash }
func (c *Container) ResourceLimits() value.ResourceLimits   { return c.resourceLimits }
func (c *Container) MonthlyBuildTime() *uint32              { return c.monthlyBuildTime }
func (c *Container) MonthlyBuildCount() *uint32             { return c.monthlyBuildCount }
func (c *Container) MonthlyUptime() *string                 { return c.monthlyUptime }
func (c *Container) GitCommitHash() *string                 { return c.gitCommitHash }
func (c *Container) EnvVars() []EnvVar                      { return c.envVars }
func (c *Container) Networks() []Network                    { return c.networks }
func (c *Container) Secrets() []Secret                      { return c.secrets }
func (c *Container) Mounts() []Mount                        { return c.mounts }
func (c *Container) IsDeleted() bool                        { return c.isDeleted }
func (c *Container) DeletedAt() *time.Time                  { return c.deletedAt }
func (c *Container) CreatedAt() time.Time                   { return c.createdAt }
func (c *Container) UpdatedAt() time.Time                   { return c.updatedAt }

// SetContainerID sets the container ID (used by repository after persistence)
func (c *Container) SetContainerID(id uint) {
	c.containerID = id
}

// ChangeName changes the container name
func (c *Container) ChangeName(name string) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	if len(name) == 0 {
		return containererrors.ErrNameRequired
	}

	// Name is for display purposes, allow up to 255 characters
	if len(name) > 255 {
		return containererrors.ErrNameTooLong
	}

	c.name = name
	c.updatedAt = time.Now()
	return nil
}

// SetStableWindow sets the stability window time
func (c *Container) SetStableWindow(window *uint32) {
	c.stableWindow = window
	c.updatedAt = time.Now()
}

// SetGitHubInstallationID sets the GitHub installation ID for private repository access
func (c *Container) SetGitHubInstallationID(installationID *int64) {
	c.githubInstallationID = installationID
	c.updatedAt = time.Now()
}

// UpdateGitConfig updates the Git configuration
func (c *Container) UpdateGitConfig(gitConfig value.GitConfig) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	c.gitConfig = gitConfig
	c.updatedAt = time.Now()
	return nil
}

// UpdateResourceLimits updates the resource limits
func (c *Container) UpdateResourceLimits(limits value.ResourceLimits) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	c.resourceLimits = limits
	c.updatedAt = time.Now()
	return nil
}

// UpdateTemplateConfig updates the template ID and configuration
func (c *Container) UpdateTemplateConfig(templateID *uint, config map[string]interface{}) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	c.templateID = templateID
	c.templateConfig = config
	c.updatedAt = time.Now()
	return nil
}

// SetLastBuiltCommitHash sets the last successfully built commit hash
func (c *Container) SetLastBuiltCommitHash(commitHash *string) {
	c.lastBuiltGitCommitHash = commitHash
	c.updatedAt = time.Now()
}

// UpdateMonthlyMetrics updates the monthly metrics
func (c *Container) UpdateMonthlyMetrics(buildTime, buildCount *uint32, uptime *string) error {
	// Note: buildTime and buildCount are unsigned, so no need to check < 0
	if uptime != nil && len(*uptime) > 10 {
		return containererrors.ErrInvalidUptime
	}

	c.monthlyBuildTime = buildTime
	c.monthlyBuildCount = buildCount
	c.monthlyUptime = uptime
	c.updatedAt = time.Now()
	return nil
}

// AddEnvVar adds an environment variable to the container
func (c *Container) AddEnvVar(key, value string) (*EnvVar, error) {
	if c.isDeleted {
		return nil, nil // Already deleted
	}

	// Check limit
	if len(c.envVars) >= MaxEnvVarsPerContainer {
		return nil, containererrors.ErrMaxEnvVarsExceeded
	}

	// Check for duplicate key
	for _, ev := range c.envVars {
		if ev.Key() == key {
			return nil, containererrors.ErrDuplicateEnvVarKey
		}
	}

	// Create new EnvVar entity
	envVar, err := NewEnvVar(c.containerID, key, value)
	if err != nil {
		return nil, err
	}

	c.envVars = append(c.envVars, *envVar)
	c.updatedAt = time.Now()
	return envVar, nil
}

// UpdateEnvVar updates an existing environment variable
func (c *Container) UpdateEnvVar(key, newValue string) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	for i := range c.envVars {
		if c.envVars[i].Key() == key {
			if err := c.envVars[i].UpdateValue(newValue); err != nil {
				return err
			}
			c.updatedAt = time.Now()
			return nil
		}
	}

	return containererrors.ErrEnvVarNotFound
}

// DeleteEnvVar removes an environment variable
func (c *Container) DeleteEnvVar(key string) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	for i, ev := range c.envVars {
		if ev.Key() == key {
			c.envVars = append(c.envVars[:i], c.envVars[i+1:]...)
			c.updatedAt = time.Now()
			return nil
		}
	}

	return containererrors.ErrEnvVarNotFound
}

// AddNetwork adds a network port mapping to the container
func (c *Container) AddNetwork(internalPort, externalPort *uint16, networkType value.NetworkType, externalIP *string, fqdn *string) (*Network, error) {
	if c.isDeleted {
		return nil, nil // Already deleted
	}

	// Check limit
	if len(c.networks) >= MaxNetworksPerContainer {
		return nil, containererrors.ErrMaxNetworksExceeded
	}

	// Check for duplicate internal port
	if internalPort != nil {
		for _, nw := range c.networks {
			if nw.InternalPort() != nil && *nw.InternalPort() == *internalPort {
				return nil, containererrors.ErrDuplicateInternalPort
			}
		}
	}

	// Create new Network entity
	network, err := NewNetwork(c.containerID, internalPort, externalPort, networkType, externalIP, fqdn)
	if err != nil {
		return nil, err
	}

	c.networks = append(c.networks, *network)
	c.updatedAt = time.Now()
	return network, nil
}

// DeleteNetwork removes a network port mapping by network ID
func (c *Container) DeleteNetwork(networkID uint) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	for i, nw := range c.networks {
		if nw.NetworkID() == networkID {
			c.networks = append(c.networks[:i], c.networks[i+1:]...)
			c.updatedAt = time.Now()
			return nil
		}
	}

	return containererrors.ErrNetworkNotFound
}

// DeleteNetworkByInternalPort removes a network port mapping by internal port
func (c *Container) DeleteNetworkByInternalPort(internalPort uint16) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	for i, nw := range c.networks {
		if nw.InternalPort() != nil && *nw.InternalPort() == internalPort {
			c.networks = append(c.networks[:i], c.networks[i+1:]...)
			c.updatedAt = time.Now()
			return nil
		}
	}

	return containererrors.ErrNetworkNotInContainer
}

// SoftDelete marks the container as deleted
func (c *Container) SoftDelete() error {
	if c.isDeleted {
		return nil // Already deleted
	}

	now := time.Now()
	c.isDeleted = true
	c.deletedAt = &now
	c.updatedAt = now
	return nil
}

// HasEnvVar checks if an environment variable with the given key exists
func (c *Container) HasEnvVar(key string) bool {
	for _, ev := range c.envVars {
		if ev.Key() == key {
			return true
		}
	}
	return false
}

// GetEnvVar retrieves an environment variable by key
func (c *Container) GetEnvVar(key string) (*EnvVar, error) {
	for i := range c.envVars {
		if c.envVars[i].Key() == key {
			return &c.envVars[i], nil
		}
	}
	return nil, containererrors.ErrEnvVarNotFound
}

// HasNetwork checks if a network with the given ID exists
func (c *Container) HasNetwork(networkID uint) bool {
	for _, nw := range c.networks {
		if nw.NetworkID() == networkID {
			return true
		}
	}
	return false
}

// AddSecret adds a secret to the container
func (c *Container) AddSecret(key, value string) (*Secret, error) {
	if c.isDeleted {
		return nil, nil // Already deleted
	}

	// Check limit
	if len(c.secrets) >= MaxSecretsPerContainer {
		return nil, containererrors.ErrMaxSecretsExceeded
	}

	// Check for duplicate key
	for _, s := range c.secrets {
		if s.Key() == key {
			return nil, containererrors.ErrDuplicateSecretKey
		}
	}

	// Create new Secret entity
	secret, err := NewSecret(c.containerID, key, value)
	if err != nil {
		return nil, err
	}

	c.secrets = append(c.secrets, *secret)
	c.updatedAt = time.Now()
	return secret, nil
}

// UpdateSecret updates an existing secret
func (c *Container) UpdateSecret(key, newValue string) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	for i := range c.secrets {
		if c.secrets[i].Key() == key {
			if err := c.secrets[i].UpdateValue(newValue); err != nil {
				return err
			}
			c.updatedAt = time.Now()
			return nil
		}
	}

	return containererrors.ErrSecretNotFound
}

// DeleteSecret removes a secret
func (c *Container) DeleteSecret(key string) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	for i, s := range c.secrets {
		if s.Key() == key {
			c.secrets = append(c.secrets[:i], c.secrets[i+1:]...)
			c.updatedAt = time.Now()
			return nil
		}
	}

	return containererrors.ErrSecretNotFound
}

// HasSecret checks if a secret with the given key exists
func (c *Container) HasSecret(key string) bool {
	for _, s := range c.secrets {
		if s.Key() == key {
			return true
		}
	}
	return false
}

// GetSecret retrieves a secret by key
func (c *Container) GetSecret(key string) (*Secret, error) {
	for i := range c.secrets {
		if c.secrets[i].Key() == key {
			return &c.secrets[i], nil
		}
	}
	return nil, containererrors.ErrSecretNotFound
}

// AddMount adds a volume mount to the container
func (c *Container) AddMount(volumeID uint, mountPath string) (*Mount, error) {
	if c.isDeleted {
		return nil, nil // Already deleted
	}

	// Check limit
	if len(c.mounts) >= MaxMountsPerContainer {
		return nil, containererrors.ErrMaxMountsExceeded
	}

	// Check for duplicate volume ID
	for _, m := range c.mounts {
		if m.VolumeID() == volumeID {
			return nil, containererrors.ErrVolumeAlreadyMounted
		}
	}

	// Check for duplicate mount path
	for _, m := range c.mounts {
		if m.MountPath() == mountPath {
			return nil, containererrors.ErrDuplicateMountPath
		}
	}

	// Create new Mount entity
	mount, err := NewMount(c.containerID, volumeID, mountPath)
	if err != nil {
		return nil, err
	}

	c.mounts = append(c.mounts, *mount)
	c.updatedAt = time.Now()
	return mount, nil
}

// DeleteMount removes a volume mount by volume ID
func (c *Container) DeleteMount(volumeID uint) error {
	if c.isDeleted {
		return nil // Already deleted
	}

	for i, m := range c.mounts {
		if m.VolumeID() == volumeID {
			c.mounts = append(c.mounts[:i], c.mounts[i+1:]...)
			c.updatedAt = time.Now()
			return nil
		}
	}

	return containererrors.ErrMountNotInContainer
}

// HasMount checks if a volume is mounted to this container
func (c *Container) HasMount(volumeID uint) bool {
	for _, m := range c.mounts {
		if m.VolumeID() == volumeID {
			return true
		}
	}
	return false
}

// GetMount retrieves a mount by volume ID
func (c *Container) GetMount(volumeID uint) (*Mount, error) {
	for i := range c.mounts {
		if c.mounts[i].VolumeID() == volumeID {
			return &c.mounts[i], nil
		}
	}
	return nil, containererrors.ErrMountNotFound
}

// AddSecretDirect adds an already-constructed Secret entity to the container
// This is used when loading entities from the database
func (c *Container) AddSecretDirect(secret *Secret) error {
	// Check for duplicate key
	for _, s := range c.secrets {
		if s.Key() == secret.Key() {
			return ErrDuplicateSecretKey
		}
	}
	c.secrets = append(c.secrets, *secret)
	return nil
}

// AddEnvVarDirect adds an already-constructed EnvVar entity to the container
// This is used when loading entities from the database
func (c *Container) AddEnvVarDirect(envVar *EnvVar) error {
	// Check for duplicate key
	for _, ev := range c.envVars {
		if ev.Key() == envVar.Key() {
			return ErrDuplicateEnvVarKey
		}
	}
	c.envVars = append(c.envVars, *envVar)
	return nil
}

// AddNetworkDirect adds an already-constructed Network entity to the container
// This is used when loading entities from the database
func (c *Container) AddNetworkDirect(network *Network) error {
	// Check for duplicate internal port
	if network.InternalPort() != nil {
		for _, nw := range c.networks {
			if nw.InternalPort() != nil && *nw.InternalPort() == *network.InternalPort() {
				return ErrDuplicateInternalPort
			}
		}
	}
	c.networks = append(c.networks, *network)
	return nil
}

// AddMountDirect adds an already-constructed Mount entity to the container
// This is used when loading entities from the database
func (c *Container) AddMountDirect(mount *Mount) error {
	// Check for duplicate volume ID
	for _, m := range c.mounts {
		if m.VolumeID() == mount.VolumeID() {
			return containererrors.ErrVolumeAlreadyMounted
		}
	}
	c.mounts = append(c.mounts, *mount)
	return nil
}

// ReconstructContainer reconstructs a container from persistence
// This is used when loading a container from the database
func ReconstructContainer(
	containerID uint,
	projectID uint,
	templateID *uint,
	name string,
	slug value.ContainerSlug,
	stableWindow *uint32,
	templateConfig map[string]interface{},
	githubInstallationID *int64,
	gitConfig value.GitConfig,
	gitCommitHash *string,
	lastBuiltGitCommitHash *string,
	resourceLimits value.ResourceLimits,
	monthlyBuildTime *uint32,
	monthlyBuildCount *uint32,
	monthlyUptime *string,
	isDeleted bool,
	deletedAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *Container {
	return &Container{
		containerID:            containerID,
		projectID:              projectID,
		templateID:             templateID,
		name:                   name,
		slug:                   slug,
		stableWindow:           stableWindow,
		templateConfig:         templateConfig,
		githubInstallationID:   githubInstallationID,
		gitConfig:              gitConfig,
		gitCommitHash:          gitCommitHash,
		lastBuiltGitCommitHash: lastBuiltGitCommitHash,
		resourceLimits:         resourceLimits,
		monthlyBuildTime:       monthlyBuildTime,
		monthlyBuildCount:      monthlyBuildCount,
		monthlyUptime:          monthlyUptime,
		envVars:                []EnvVar{},
		networks:               []Network{},
		secrets:                []Secret{},
		mounts:                 []Mount{},
		createdAt:              createdAt,
		updatedAt:              updatedAt,
		deletedAt:              deletedAt,
		isDeleted:              isDeleted,
	}
}
