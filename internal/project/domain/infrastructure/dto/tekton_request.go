// Package dto defines Data Transfer Objects for infrastructure layer communications.
package dto

// TektonDeployRequest represents the complete request payload for Tekton API.
// This is the top-level structure sent to the Tekton deployment endpoint.
//
// API Endpoint: POST https://tekton-api.launchpad.kr/deploy
type TektonDeployRequest struct {
	// DeploymentConfigJSON contains the deployment configuration
	DeploymentConfigJSON DeploymentConfig `json:"deployment_config_json"`

	// DryRun determines whether to actually deploy or just validate
	// Valid values: "true" or "false" (string, not boolean)
	DryRun string `json:"dry_run"`
}

// DeploymentConfig represents the deployment configuration for Tekton API.
// This combines Project metadata with Container configuration.
//
// This structure maps exactly to Tekton's deployment_config_json specification.
type DeploymentConfig struct {
	// ProjectID is the unique project identifier (from Project context)
	ProjectID string `json:"project_id"`

	// ServiceName is the Knative service name (from Project.slug)
	// Will be used as the base name for all resources
	ServiceName string `json:"service_name"`

	// Namespace is the Kubernetes namespace for deployment (constant: "application")
	Namespace string `json:"namespace"`

	// StableWindow is the observation period for scale-to-zero decisions in seconds (constant: 180)
	StableWindow int `json:"stable_window"`

	// Plan is the project plan type: "free", "eco", or "pro"
	// This determines scale-to-zero behavior and resource allocation policies
	Plan string `json:"plan"`

	// ConfigMaps contains ConfigMap configurations (managed at project level)
	ConfigMaps []ConfigMapInfo `json:"config_maps"`

	// Volumes contains volume configurations (managed at project level)
	Volumes []VolumeInfo `json:"volumes"`

	// Containers contains container configurations ready for Tekton deployment
	// These are converted from ContainerInfo with volume_id mapped to volume_slug
	Containers []TektonContainerInfo `json:"containers"`
}

// TektonContainerInfo represents a container configuration ready for Tekton deployment.
// This is the final form sent to Tekton API, with all volume_ids resolved to volume_slugs.
type TektonContainerInfo struct {
	// Name is the container name (required)
	Name string `json:"name"`

	// Domain is the external domain for the container (optional)
	Domain *string `json:"domain,omitempty"`

	// HealthCheckType specifies the type of health check (required)
	HealthCheckType string `json:"health_check_type"`

	// HealthEndpoint is the HTTP endpoint path for health checks (optional)
	HealthEndpoint *string `json:"health_endpoint,omitempty"`

	// Port is the container port number (required)
	Port int `json:"port"`

	// HealthPort is the port for health checks (optional)
	HealthPort *int `json:"health_port,omitempty"`

	// ImageName is the full container image name (required)
	ImageName string `json:"image_name"`

	// ImageTag is the container image tag (required)
	ImageTag string `json:"image_tag"`

	// EnvVars contains environment variables (optional)
	EnvVars map[string]string `json:"env_vars,omitempty"`

	// Secrets contains secret values (optional)
	Secrets map[string]string `json:"secrets,omitempty"`

	// CPULimit is the CPU limit (required)
	CPULimit string `json:"cpu_limit"`

	// MemoryRequest is the memory request amount (required)
	MemoryRequest string `json:"memory_request"`

	// MemoryLimit is the memory limit (required)
	MemoryLimit string `json:"memory_limit"`

	// VolumeMounts contains volume mount configurations with resolved volume slugs
	VolumeMounts []TektonVolumeMount `json:"volume_mounts"`
}

// TektonVolumeMount represents a volume mount for Tekton deployment.
// Volume IDs have been resolved to volume slugs at this stage.
type TektonVolumeMount struct {
	// VolumeName is the slug of the volume to mount (required)
	VolumeName string `json:"volume_name"`

	// MountPaths is the list of paths where the volume will be mounted (required)
	MountPaths []string `json:"mount_paths"`
}

// VolumeInfo represents a volume configuration for the deployment.
// Volumes are PVC (persistent volume claims) managed at project level.
type VolumeInfo struct {
	// Name is the volume name used in the pod (required)
	// For PVC volumes, this is the volume slug
	Name string `json:"name"`

	// Capacity is the volume capacity (required for PVC)
	// Format: "1Gi", "10Gi"
	Capacity *string `json:"capacity,omitempty"`
}

// ConfigMapInfo represents a Kubernetes ConfigMap configuration.
// ConfigMaps are managed at project level and used to store non-confidential data.
type ConfigMapInfo struct {
	// Name is the ConfigMap name (required)
	// The actual Kubernetes resource will be named as "<service_name>-<name>"
	Name string `json:"name"`

	// Data contains the ConfigMap data as key-value pairs (required)
	// Supports multi-line strings for file contents
	Data map[string]string `json:"data"`
}
