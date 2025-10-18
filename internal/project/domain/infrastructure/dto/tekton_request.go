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

	// ConfigMaps contains ConfigMap configurations (managed at project level)
	ConfigMaps []ConfigMapInfo `json:"config_maps"`

	// Volumes contains volume configurations (managed at project level)
	Volumes []VolumeInfo `json:"volumes"`

	// Containers contains container configurations (from Container context)
	Containers []ContainerInfo `json:"containers"`
}
