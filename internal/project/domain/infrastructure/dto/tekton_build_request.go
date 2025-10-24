// Package dto defines Data Transfer Objects for infrastructure layer communications.
package dto

// TektonBuildRequest represents the request payload for triggering a Tekton build pipeline.
// This request is sent to the Tekton EventListener endpoint to initiate an image build process.
//
// The build pipeline performs the following steps:
//  1. Clones the GitHub repository (if github_url is provided)
//  2. Generates a Dockerfile from the template and config
//  3. Builds the container image with build-time environment variables
//  4. Pushes the image to the container registry
//  5. Returns build results (latest_commit_hash, image_tag, should_build)
//
// The pipeline can skip the build if:
//   - force_build is "false" AND
//   - latest_commit_hash matches last_build_commit_hash (no code changes)
//
// All string parameters are sent as-is to the Tekton API, with boolean force_build
// converted to "true" or "false" string.
type TektonBuildRequest struct {
	// ImageName is the name of the container image to build
	// This will be used as the image name in the registry (e.g., "my-app")
	// Required field
	ImageName string `json:"image_name"`

	// GitHubURL is the GitHub repository URL to clone
	// If empty, the build will use only the template without cloning a repository
	// Example: "https://github.com/org/repo"
	// Optional field
	GitHubURL string `json:"github_url,omitempty"`

	// GitHubBranch is the branch to checkout from the repository
	// Required if GitHubURL is provided
	// Example: "main", "develop"
	// Optional field
	GitHubBranch string `json:"github_branch,omitempty"`

	// DirectoryPath is the path within the repository where the build should occur
	// Defaults to "." (repository root) if not specified
	// Example: "./backend", "./services/api"
	// Optional field
	DirectoryPath string `json:"directory_path,omitempty"`

	// ForceBuild controls whether to force a rebuild even if no code changes detected
	// Must be "true" or "false" (string, not boolean, as required by Tekton API)
	// - "true": Always build regardless of commit hash comparison
	// - "false": Build only if latest_commit_hash differs from last_build_commit_hash
	// Required field
	ForceBuild string `json:"force_build"`

	// LastBuildCommitHash is the commit hash from the previous successful build
	// Used for comparison with latest_commit_hash to determine if build is needed
	// Empty string if this is the first build
	// Optional field
	LastBuildCommitHash string `json:"last_build_commit_hash,omitempty"`

	// Template is the full Dockerfile template content
	// This template may contain gomplate syntax for variable substitution
	// Example: "FROM {{ .base_image }}\nCOPY . /app"
	// Required field
	Template string `json:"template"`

	// DockerfileConfigJSON is a JSON string containing template variable values
	// Used by gomplate to substitute variables in the Template
	// Example: `{"base_image":"node:18","port":"3000"}`
	// Optional field
	DockerfileConfigJSON string `json:"dockerfile_config_json,omitempty"`

	// BuildEnvJSON is a JSON string containing build-time environment variables
	// These are converted to ARG statements and available during the build process
	// Example: `{"NODE_ENV":"production","API_URL":"https://api.example.com"}`
	// Optional field
	BuildEnvJSON string `json:"build_env_json,omitempty"`

	// RegistryURL is the container registry URL where the built image will be pushed
	// Defaults to "registry.launchpad.kr/" if not specified
	// Example: "registry.example.com/"
	// Optional field
	RegistryURL string `json:"registry_url,omitempty"`

	// InstallationID is the GitHub App installation ID for accessing private repositories
	// Required for private repositories, optional for public repositories
	// Example: "12345678"
	// Optional field
	InstallationID string `json:"installation_id,omitempty"`
}
