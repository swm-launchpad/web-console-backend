// Package dto defines Data Transfer Objects for infrastructure layer communications.
package dto

import "time"

// PipelineRun represents information about a Tekton PipelineRun.
// This unified structure is used for both single PipelineRun status queries
// and listing multiple PipelineRuns for a project.
type PipelineRun struct {
	// Name is the PipelineRun resource name
	Name string

	// ProjectID is the associated project ID
	// Retrieved from the "project-id" label on the PipelineRun
	// Set to 0 when querying a single PipelineRun status
	ProjectID uint

	// EventID is the Tekton event ID that triggered this PipelineRun
	// Retrieved from the "triggers.tekton.dev/triggers-eventid" label on the PipelineRun
	// Empty when querying a single PipelineRun status
	EventID string

	// Status is the raw condition status from Tekton PipelineRun
	// Valid values: "True", "False", "Unknown"
	// Retrieved from status.conditions[].status (type=Succeeded preferred, fallback to first condition)
	Status string

	// Reason provides a brief reason for the current status
	// Raw value from status.conditions[].reason (type=Succeeded preferred, fallback to first condition)
	// Examples: "Completed", "Failed", "Running", "PipelineRunTimeout"
	Reason string

	// Message provides additional information about the status
	// Raw value from status.conditions[].message (type=Succeeded preferred, fallback to first condition)
	Message string

	// StartTime is when the PipelineRun started executing
	// Nil if the PipelineRun hasn't started yet
	StartTime *time.Time

	// CompletionTime is when the PipelineRun completed
	// Nil if the PipelineRun is still running or hasn't started
	CompletionTime *time.Time

	// Results contains the task results from the PipelineRun
	// For build pipelines, this typically includes:
	//   - latest_commit_hash: The latest Git commit hash
	//   - image_tag: The image tag used for the build
	//   - should_build: Whether the build was executed ("true" or "false")
	// This field is populated only when specifically requested (e.g., for build monitoring)
	// and will be nil for deploy pipeline queries
	Results map[string]string
}
