// Package dto defines Data Transfer Objects for infrastructure layer communications.
package dto

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
