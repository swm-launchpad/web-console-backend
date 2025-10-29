package dto

// UnifiedContainerConfig represents a single-source-of-truth snapshot of container configuration.
// This DTO is returned by GetContainerConfig and can be converted to either build or deployment formats.
// By using a single unified source, we eliminate the possibility of snapshot divergence between
// build and deployment configurations (P1 Badge fix).
type UnifiedContainerConfig struct {
	ProjectID  uint
	Containers []UnifiedContainerInfo
}

// UnifiedContainerInfo contains all information needed for both build and deployment operations.
// This unified structure ensures that build and deployment use identical container snapshots.
type UnifiedContainerInfo struct {
	// === Identity fields (common) ===
	ContainerID uint
	Name        string // Container name (used as slug in some contexts)
	Slug        string // Unique container identifier

	// === Build-specific fields ===
	TemplateID           *uint
	TemplateBody         *string
	TemplateConfig       map[string]interface{}
	GitRepositoryURL     string
	GitBranch            string
	GitDirectoryPath     *string
	LastBuiltCommitHash  *string
	NeedsBuild           bool
	BuildVars            map[string]string
	GitHubInstallationID *int64

	// === Deployment-specific fields ===
	ImageName   string
	ImageTag    string
	CPULimit    *uint32
	MemoryLimit *uint32
	EnvVars     map[string]string
	Secrets     map[string]string
	Networks    []NetworkInfo
	Mounts      []MountInfo

	// === Domain/Health check fields (deployment) ===
	Domain          *string
	HealthCheckType string
	HealthEndpoint  *string
	Port            int
	HealthPort      *int
}

// NetworkInfo represents network configuration for a container
type NetworkInfo struct {
	NetworkID    uint
	InternalPort uint16
	ExternalPort uint16
	ExternalIP   string
	FQDN         string
	NetworkType  string
}

// MountInfo represents volume mount configuration for a container
type MountInfo struct {
	VolumeID  uint
	MountPath string
}
