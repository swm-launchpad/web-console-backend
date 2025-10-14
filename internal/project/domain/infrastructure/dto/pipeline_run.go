// Package dto defines Data Transfer Objects for infrastructure layer communications.
package dto

import "time"

// PipelineRunStatus represents the status of a Tekton PipelineRun.
// It contains information about the current state of a deployment pipeline execution.
type PipelineRunStatus struct {
	// Name is the PipelineRun resource name
	Name string

	// Status is the overall status of the PipelineRun
	// Valid values: "Succeeded", "Failed", "Running", "Pending", "Unknown"
	Status string

	// StartTime is when the PipelineRun started executing
	// Nil if the PipelineRun hasn't started yet
	StartTime *time.Time

	// CompletionTime is when the PipelineRun completed
	// Nil if the PipelineRun is still running or hasn't started
	CompletionTime *time.Time

	// Message provides additional information about the status
	// Typically contains error messages for failed runs or completion messages
	Message string

	// Reason provides a brief reason for the current status
	// Examples: "Succeeded", "Failed", "Running", "PipelineRunTimeout"
	Reason string
}

// PipelineRunInfo represents summary information about a PipelineRun.
// It is used when listing multiple PipelineRuns for a project.
type PipelineRunInfo struct {
	// Name is the PipelineRun resource name
	Name string

	// ProjectID is the associated project ID
	// Retrieved from the "project-id" label on the PipelineRun
	ProjectID uint

	// EventID is the Tekton event ID that triggered this PipelineRun
	// Retrieved from the "tekton.dev/triggers-eventid" label on the PipelineRun
	EventID string

	// Status is the overall status of the PipelineRun
	// Valid values: "Succeeded", "Failed", "Running", "Pending", "Unknown"
	Status string

	// StartTime is when the PipelineRun started executing
	// Nil if the PipelineRun hasn't started yet
	StartTime *time.Time

	// CompletionTime is when the PipelineRun completed
	// Nil if the PipelineRun is still running or hasn't started
	CompletionTime *time.Time

	// Message provides additional information about the status
	Message string
}
