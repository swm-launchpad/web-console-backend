// Package infrastructure defines interfaces for external infrastructure dependencies.
// These interfaces abstract away external services and deployment platforms.
package infrastructure

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// TektonBuildClient defines the interface for interacting with Tekton build API.
// This interface abstracts the Tekton EventListener endpoint that triggers PipelineRuns
// for building and pushing container images based on the provided configuration.
//
// The implementation uses HTTP POST requests to communicate with the Tekton EventListener,
// which validates the build configuration and creates a PipelineRun resource
// in the Kubernetes cluster.
//
// Authentication is performed using Basic Authentication with credentials configured
// in environment variables (shared with TektonClient).
type TektonBuildClient interface {
	// TriggerBuild initiates a container image build by sending a build request to Tekton API.
	// This triggers the creation of a PipelineRun that will execute the image-build-push pipeline.
	//
	// The build pipeline performs the following steps:
	//   1. Clone the GitHub repository (if github_url is provided)
	//   2. Fetch the latest commit hash
	//   3. Determine if build is needed (force_build or commit hash changed)
	//   4. Generate Dockerfile from template and config
	//   5. Build the container image with build-time environment variables
	//   6. Push the image to the container registry
	//   7. Return build results in PipelineRun status.results
	//
	// The build is asynchronous - this method returns immediately after the Tekton
	// EventListener accepts the request. The actual build status must be monitored
	// separately using KubeBuildClient.GetPipelineRunStatus().
	//
	// Build Necessity Logic:
	//   - If force_build is "true": Always build
	//   - If force_build is "false":
	//     - No github_url: Skip build (return should_build=false in results)
	//     - With github_url: Compare latest_commit_hash with last_build_commit_hash
	//       - If same: Skip build (should_build=false)
	//       - If different: Execute build (should_build=true)
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - request: Complete build configuration including image name, git info, template, etc.
	//
	// Returns:
	//   - *dto.TektonBuildResponse: Response from Tekton EventListener containing event metadata
	//   - error: An error if the request fails, authentication fails, or Tekton rejects the request
	//
	// The Tekton API returns HTTP 202 Accepted if the build request is valid and accepted.
	// The response includes metadata about the triggered event, but does NOT include the
	// PipelineRun name (due to Kubernetes generateName). The PipelineRun must be located
	// using KubeBuildClient.FindPipelineRunNameByEventID() with the EventID.
	//
	// Error cases:
	//   - ErrTektonUnavailable: Tekton API is unreachable or misconfigured
	//   - ErrTektonBuildFailed: Tekton rejected the build request (e.g., validation failed)
	//   - ErrInvalidTektonResponse: Response from Tekton could not be parsed
	//
	// Example usage:
	//   request := &dto.TektonBuildRequest{
	//       ImageName:           "my-app",
	//       GitHubURL:           "https://github.com/org/repo",
	//       GitHubBranch:        "main",
	//       ForceBuild:          "false",
	//       LastBuildCommitHash: "abc123...",
	//       Template:            "FROM node:18\nCOPY . /app",
	//       RegistryURL:         "registry.launchpad.kr/",
	//   }
	//   response, err := client.TriggerBuild(ctx, request)
	//   if err != nil {
	//       return err
	//   }
	//   log.Printf("Build triggered: eventID=%s", response.EventID)
	TriggerBuild(ctx context.Context, request *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error)
}
