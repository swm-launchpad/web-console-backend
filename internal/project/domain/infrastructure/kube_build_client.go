// Package infrastructure defines interfaces for external infrastructure dependencies.
// These interfaces abstract away external services and Kubernetes resources.
package infrastructure

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// KubeBuildClient defines the interface for interacting with Kubernetes API for build operations.
// This interface abstracts Tekton PipelineRun operations for the image-build-push pipeline,
// allowing the domain layer to query build status and results without depending on
// Kubernetes client libraries.
//
// The implementation uses Kubernetes Dynamic Client to interact with Tekton CRDs
// (Custom Resource Definitions) in the build-pipeline namespace.
//
// Key differences from KubeClient (deploy):
//   - Operates in build-pipeline namespace (vs deploy-pipeline)
//   - Extracts and returns PipelineRun results (latest_commit_hash, image_tag, should_build)
//   - Used for monitoring image build progress, not deployment progress
//
// Authentication is performed using a ServiceAccount token with appropriate RBAC permissions
// to read PipelineRuns in the build-pipeline namespace.
type KubeBuildClient interface {
	// GetPipelineRunStatus retrieves the current status of a specific build PipelineRun.
	// This is used to monitor the progress of an image build and determine if it
	// has succeeded, failed, or is still running.
	//
	// The status is determined by examining the PipelineRun's status.conditions,
	// particularly the "Succeeded" condition type.
	//
	// Build Results:
	// Unlike deploy PipelineRuns, build PipelineRuns include results in status.results:
	//   - latest_commit_hash: The Git commit hash that was built
	//   - image_tag: The image tag used (first 7 chars of commit or "latest")
	//   - should_build: "true" if build was executed, "false" if skipped (no changes)
	//
	// These results are critical for:
	//   - Updating Container.last_built_git_commit_hash
	//   - Determining if build was skipped vs. actually executed
	//   - Tracking which commit was built
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - pipelineRunName: The name of the PipelineRun resource (e.g., "image-build-push-run-abc123")
	//
	// Returns:
	//   - *dto.PipelineRun: The current status of the PipelineRun with Results populated
	//   - error: An error if the operation fails or the PipelineRun is not found
	//
	// Possible status values:
	//   - "True": PipelineRun condition status is True (build succeeded)
	//   - "False": PipelineRun condition status is False (build failed)
	//   - "Unknown": Status cannot be determined (build still running)
	//
	// Example usage:
	//   status, err := client.GetPipelineRunStatus(ctx, "image-build-push-run-abc123")
	//   if err != nil {
	//       return err
	//   }
	//   if status.Status == "True" {
	//       commitHash := status.Results["latest_commit_hash"]
	//       shouldBuild := status.Results["should_build"]
	//       // Handle successful build
	//   }
	GetPipelineRunStatus(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error)

	// FindPipelineRunNameByEventID retrieves the PipelineRun name associated with a Tekton event ID.
	// This is used to look up a PipelineRun when you have the EventID returned from the Tekton
	// EventListener but need to query the PipelineRun's status or results.
	//
	// The EventID is stored in the "triggers.tekton.dev/triggers-eventid" label on PipelineRun resources.
	// This label is automatically set by the Tekton EventListener when a PipelineRun is created.
	//
	// If multiple PipelineRuns are found with the same EventID (which should not normally happen),
	// this method returns the most recently created one based on startTime.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - eventID: The Tekton event ID (e.g., "abc-123-xyz")
	//
	// Returns:
	//   - string: The name of the PipelineRun resource
	//   - error: An error if the operation fails or no PipelineRun is found with the given EventID
	//
	// Possible errors:
	//   - ErrKubePipelineRunNotFound: No PipelineRun found with the given EventID
	//   - ErrKubernetesUnavailable: Kubernetes API is not available or the request failed
	//
	// Example usage:
	//   name, err := client.FindPipelineRunNameByEventID(ctx, "abc-123-xyz")
	//   if err != nil {
	//       return err
	//   }
	//   status, err := client.GetPipelineRunStatus(ctx, name)
	FindPipelineRunNameByEventID(ctx context.Context, eventID string) (string, error)
}
