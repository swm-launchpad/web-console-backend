// Package dto defines Data Transfer Objects for infrastructure layer communications.
// These DTOs are used for exchanging data with external bounded contexts.
package dto

// BuildContainerInfo represents a single container configuration for building.
// It contains all necessary information to build a container image including
// template, git configuration, and build-time environment variables.
type BuildContainerInfo struct {
	// ProjectID is the unique identifier of the project (required)
	// This is used to label PipelineRuns in Tekton for proper tracking
	ProjectID uint `json:"project_id"`

	// ContainerID is the unique identifier of the container (required)
	ContainerID uint `json:"container_id"`

	// Name is the container display name (required)
	Name string `json:"name"`

	// Slug is the container slug identifier (required)
	Slug string `json:"slug"`

	// TemplateBody is the Dockerfile template content (optional)
	// If provided, it will be used to generate the Dockerfile during build
	TemplateBody *string `json:"template_body,omitempty"`

	// TemplateConfig contains configuration for template rendering (optional)
	// Used to substitute variables in the template
	TemplateConfig map[string]interface{} `json:"template_config,omitempty"`

	// GitRepositoryURL is the URL of the Git repository (required for builds)
	// Example: "https://github.com/user/repo"
	GitRepositoryURL string `json:"git_repository_url"`

	// GitBranch is the branch to build from (required)
	// Example: "main", "develop"
	GitBranch string `json:"git_branch"`

	// GitDirectoryPath is the subdirectory path within the repository (optional)
	// Used for monorepo structures
	// Example: "/backend", "/services/api"
	GitDirectoryPath *string `json:"git_directory_path,omitempty"`

	// LastBuiltCommitHash is the commit hash of the last successful build (optional)
	// Used to determine if a rebuild is necessary
	LastBuiltCommitHash *string `json:"last_built_commit_hash,omitempty"`

	// NeedsBuild indicates whether a build is required (required)
	// Set to true when build parameters change
	NeedsBuild bool `json:"needs_build"`

	// BuildVars contains build-time environment variables as key-value pairs (optional)
	// These are passed to the build process
	BuildVars map[string]string `json:"build_vars,omitempty"`

	// InstallationID is the GitHub App installation ID for private repository access (optional)
	// Required for building from private repositories
	InstallationID *int64 `json:"installation_id,omitempty"`
}

// ContainerBuildConfig represents container-specific build configuration.
// This is provided by the Container bounded context and contains only container-level settings.
// Project metadata is managed at the Project level, not by ContainerClient.
type ContainerBuildConfig struct {
	// Containers contains container build configurations (required, at least 1)
	Containers []BuildContainerInfo `json:"containers"`
}
