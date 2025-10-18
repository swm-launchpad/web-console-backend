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
	// Can reference either a PVC volume (by slug) or ConfigMap (by name)
	VolumeName string `json:"volume_name"`

	// MountPaths is the list of paths where the volume will be mounted (required, at least 1)
	// Example: ["/var/lib/mysql"], ["/docker-entrypoint-initdb.d"]
	MountPaths []string `json:"mount_paths"`
}

// ContainerDeploymentConfig represents container-specific deployment configuration.
// This is provided by the Container bounded context and contains only container-level settings.
// Project metadata (project_id, service_name, namespace, stable_window), ConfigMaps, and Volumes
// are managed at the Project level, not by ContainerClient.
type ContainerDeploymentConfig struct {
	// Containers contains container configurations (required, at least 1)
	Containers []ContainerInfo `json:"containers"`
}
