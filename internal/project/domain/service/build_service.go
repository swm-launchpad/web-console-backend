package service

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// BuildService defines the interface for build operations
// This service is responsible for building individual containers using Tekton pipelines
type BuildService interface {
	// BuildContainer executes a build for a single container
	// This method is designed to be called in a goroutine
	// It triggers the Tekton build pipeline, monitors its status every 30 seconds,
	// and updates the BUILD_HISTORY record accordingly
	//
	// Parameters:
	//   - ctx: Context for cancellation and deadline control
	//   - buildHistoryID: ID of the BUILD_HISTORY record to track this build
	//   - container: Container configuration including git repo, template, and build vars
	//
	// Returns:
	//   - BuildResult: Contains build status, commit hash, and error information
	//   - error: Returns error if build cannot be initiated or tracking fails
	//
	// Return Value Contract:
	//   This method MUST follow one of these patterns:
	//   1. (BuildResult, nil) - Build completed successfully
	//   2. (BuildResult, error) - Build reached terminal failure state with metadata
	//   3. (nil, error) - Context cancelled or tracking lost before completion
	//
	//   NEVER returns (nil, nil) - this is a contract violation
	//   Invariant: If error is nil, result MUST be non-nil
	//              If result is nil, error MUST be non-nil
	//
	// The method will:
	//  1. Trigger Tekton image-build-push pipeline with container config
	//  2. Poll PipelineRun status every 30 seconds
	//  3. Update BUILD_HISTORY status as the build progresses
	//  4. Return BuildResult when the build reaches a terminal state
	//  5. Handle timeouts (5 minutes for tracking start, 30 minutes total)
	BuildContainer(
		ctx context.Context,
		buildHistoryID uint,
		container *dto.BuildContainerInfo,
	) (*BuildResult, error)
}

// BuildResult represents the outcome of a build operation
// This is returned when a build reaches a terminal state
type BuildResult struct {
	// BuildHistoryID is the ID of the BUILD_HISTORY record
	BuildHistoryID uint

	// Status represents the final build status
	// Possible values: "success", "failed", "cancelled", "skipped"
	Status string

	// LatestCommitHash is the commit hash that was built
	// This is obtained from Tekton PipelineRun results
	// May be empty if the build failed before git operations
	LatestCommitHash string

	// ImageTag is the tag of the built container image
	// This is obtained from Tekton PipelineRun results
	// Typically "latest" for successful builds
	ImageTag string

	// ShouldBuild indicates whether the build was actually executed
	// False if the build was skipped due to no changes
	ShouldBuild bool

	// ErrorMessage contains error details if the build failed
	// Empty for successful builds
	ErrorMessage string
}
