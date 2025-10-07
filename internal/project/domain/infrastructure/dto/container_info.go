// Package dto defines Data Transfer Objects for infrastructure layer communications.
// These DTOs are used for exchanging data with external bounded contexts.
package dto

// ContainerInfo represents a single container configuration for deployment.
// It contains all necessary information to deploy a container including
// image details, resource limits, health checks, and volume mounts.
type ContainerInfo struct {
	// Name is the container name (required)
	Name string `json:"name"`

	// Domain is the external domain for the container (optional)
	// If provided, the container will be accessible from outside
	// If not provided, the container is internal-only
	Domain *string `json:"domain,omitempty"`

	// HealthCheckType specifies the type of health check (required)
	// Valid values: "http", "tcp", "none"
	HealthCheckType string `json:"health_check_type"`

	// HealthEndpoint is the HTTP endpoint path for health checks (required if HealthCheckType is "http")
	// Example: "/health", "/actuator/health"
	HealthEndpoint *string `json:"health_endpoint,omitempty"`

	// Port is the container port number (required)
	Port int `json:"port"`

	// HealthPort is the port for health checks (optional)
	// If not specified, Port will be used for health checks
	// Useful for Spring Boot's management.server.port configuration
	HealthPort *int `json:"health_port,omitempty"`

	// ImageName is the full container image name (required)
	// Example: "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/spring-helloworld"
	ImageName string `json:"image_name"`

	// ImageTag is the container image tag (required)
	// Example: "e5c373e", "latest"
	ImageTag string `json:"image_tag"`

	// EnvVars contains environment variables as key-value pairs (optional)
	EnvVars map[string]string `json:"env_vars,omitempty"`

	// Secrets contains secret values as key-value pairs (optional)
	Secrets map[string]string `json:"secrets,omitempty"`

	// CPULimit is the CPU limit (required)
	// Format: "1000m" (millicores)
	CPULimit string `json:"cpu_limit"`

	// MemoryRequest is the memory request amount (required)
	// Format: "512Mi"
	MemoryRequest string `json:"memory_request"`

	// MemoryLimit is the memory limit (required)
	// Format: "1Gi"
	MemoryLimit string `json:"memory_limit"`

	// VolumeMounts contains volume mount configurations (required, can be empty array)
	VolumeMounts []VolumeMount `json:"volume_mounts"`
}

// VolumeMount represents a volume mount configuration.
// It specifies which volume to mount and where to mount it in the container.
type VolumeMount struct {
	// VolumeName is the name of the volume to mount (required)
	// Must match a volume name in VolumeInfo
	VolumeName string `json:"volume_name"`

	// MountPaths is the list of paths where the volume will be mounted (required, at least 1)
	// Example: ["/var/lib/mysql"], ["/docker-entrypoint-initdb.d"]
	MountPaths []string `json:"mount_paths"`
}

// VolumeInfo represents a volume configuration for the deployment.
// Volumes can be either persistent volume claims (PVC) or ConfigMap references.
type VolumeInfo struct {
	// Name is the volume name used in the pod (required)
	Name string `json:"name"`

	// Type specifies the volume type (optional, default: "pvc")
	// Valid values: "pvc" (persistent volume claim), "config_map"
	Type *string `json:"type,omitempty"`

	// Capacity is the volume capacity (required if Type is "pvc" or not specified)
	// Format: "1Gi", "10Gi"
	Capacity *string `json:"capacity,omitempty"`

	// ConfigMapName is the ConfigMap name to use (required if Type is "config_map")
	// Must match a ConfigMap name in ConfigMapInfo
	ConfigMapName *string `json:"config_map_name,omitempty"`
}

// ConfigMapInfo represents a Kubernetes ConfigMap configuration.
// ConfigMaps are used to store non-confidential data in key-value pairs.
type ConfigMapInfo struct {
	// Name is the ConfigMap name (required)
	// The actual Kubernetes resource will be named as "<service_name>-<name>"
	Name string `json:"name"`

	// Data contains the ConfigMap data as key-value pairs (required)
	// Supports multi-line strings for file contents
	Data map[string]string `json:"data"`
}

// ContainerDeploymentConfig represents container-specific deployment configuration.
// This is provided by the Container bounded context and contains only container-level settings.
// Project metadata (project_id, service_name, namespace, stable_window) should come from
// the Project bounded context.
type ContainerDeploymentConfig struct {
	// ConfigMaps contains ConfigMap configurations (required, can be empty array)
	ConfigMaps []ConfigMapInfo `json:"config_maps"`

	// Volumes contains volume configurations (required, can be empty array)
	Volumes []VolumeInfo `json:"volumes"`

	// Containers contains container configurations (required, at least 1)
	Containers []ContainerInfo `json:"containers"`
}

// DeploymentRequest represents the final deployment request for Tekton API.
// This combines Project metadata with Container configuration.
type DeploymentRequest struct {
	// ProjectID is the unique project identifier (from Project context)
	ProjectID string `json:"project_id"`

	// ServiceName is the Knative service name (from Project.slug)
	// Will be used as the base name for all resources
	ServiceName string `json:"service_name"`

	// Namespace is the Kubernetes namespace for deployment (constant: "application")
	Namespace string `json:"namespace"`

	// StableWindow is the observation period for scale-to-zero decisions in seconds (constant: 180)
	StableWindow int `json:"stable_window"`

	// ConfigMaps contains ConfigMap configurations (from Container context)
	ConfigMaps []ConfigMapInfo `json:"config_maps"`

	// Volumes contains volume configurations (from Container context)
	Volumes []VolumeInfo `json:"volumes"`

	// Containers contains container configurations (from Container context)
	Containers []ContainerInfo `json:"containers"`
}
