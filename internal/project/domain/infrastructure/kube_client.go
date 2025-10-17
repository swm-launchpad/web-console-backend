// Package infrastructure defines interfaces for external infrastructure dependencies.
// These interfaces abstract away external services and Kubernetes resources.
package infrastructure

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// KubeClient defines the interface for interacting with Kubernetes API.
// This interface abstracts Tekton PipelineRun operations, allowing the domain layer
// to query deployment status and logs without depending on Kubernetes client libraries.
//
// The implementation uses Kubernetes Dynamic Client to interact with Tekton CRDs
// (Custom Resource Definitions) in the deploy-pipeline namespace.
//
// Authentication is performed using a ServiceAccount token with appropriate RBAC permissions
// to read PipelineRuns and Pod logs in the deployment namespace.
type KubeClient interface {
	// GetPipelineRunStatus retrieves the current status of a specific PipelineRun.
	// This is used to monitor the progress of a deployment and determine if it
	// has succeeded, failed, or is still running.
	//
	// The status is determined by examining the PipelineRun's status.conditions,
	// particularly the "Succeeded" condition type.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - pipelineRunName: The name of the PipelineRun resource (e.g., "deploy-run-abc123")
	//
	// Returns:
	//   - *dto.PipelineRun: The current status of the PipelineRun (ProjectID and EventID will be 0/"")
	//   - error: An error if the operation fails or the PipelineRun is not found
	//
	// Possible status values:
	//   - "True": PipelineRun condition status is True
	//   - "False": PipelineRun condition status is False
	//   - "Unknown": Status cannot be determined
	//
	// Example usage:
	//   status, err := client.GetPipelineRunStatus(ctx, "deploy-run-abc123")
	//   if err != nil {
	//       return err
	//   }
	//   if status.Status == "True" {
	//       // Handle successful deployment
	//   }
	GetPipelineRunStatus(ctx context.Context, pipelineRunName string) (*dto.PipelineRun, error)

	// GetPipelineRunLogs retrieves the aggregated logs from all tasks in a PipelineRun.
	// This is useful for debugging failed deployments or understanding what happened
	// during the deployment process.
	//
	// The implementation traverses the following hierarchy:
	//   PipelineRun → TaskRuns → Pods → Containers → Logs
	//
	// All logs from all containers in all tasks are concatenated into a single string,
	// with appropriate separators to distinguish between different tasks and containers.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - pipelineRunName: The name of the PipelineRun resource
	//
	// Returns:
	//   - string: The aggregated logs from all tasks and containers
	//   - error: An error if the operation fails or the PipelineRun/Pods are not found
	//
	// Note: This operation may take some time for PipelineRuns with many tasks or
	// large log outputs. Consider using appropriate context timeouts.
	//
	// Example usage:
	//   logs, err := client.GetPipelineRunLogs(ctx, "deploy-run-abc123")
	//   if err != nil {
	//       return err
	//   }
	//   fmt.Println(logs)
	GetPipelineRunLogs(ctx context.Context, pipelineRunName string) (string, error)

	// ListPipelineRuns retrieves a list of all PipelineRuns associated with a project.
	// This is used to show deployment history and allow users to track all deployment
	// attempts for their project.
	//
	// PipelineRuns are filtered by the "project-id" label, which is set when the
	// PipelineRun is created by the Tekton EventListener.
	//
	// The results are sorted by creation time in descending order (newest first).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - projectID: The unique identifier of the project
	//
	// Returns:
	//   - []*dto.PipelineRun: A list of PipelineRun summaries for the project
	//   - error: An error if the operation fails
	//
	// The returned list may be empty if no deployments have been triggered for the project.
	//
	// Example usage:
	//   runs, err := client.ListPipelineRuns(ctx, 123)
	//   if err != nil {
	//       return err
	//   }
	//   for _, run := range runs {
	//       fmt.Printf("Run: %s, Status: %s\n", run.Name, run.Status)
	//   }
	ListPipelineRuns(ctx context.Context, projectID uint) ([]*dto.PipelineRun, error)

	// FindPipelineRunNameByEventID retrieves the PipelineRun name associated with a Tekton event ID.
	// This is used to look up a PipelineRun when you have the EventID returned from the Tekton
	// EventListener but need to query the PipelineRun's status or logs.
	//
	// The EventID is stored in the "tekton.dev/triggers-eventid" label on PipelineRun resources.
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
