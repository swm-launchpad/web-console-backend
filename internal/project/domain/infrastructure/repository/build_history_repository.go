package repository

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/build_history"
)

// BuildHistoryRepository defines the interface for build history persistence
type BuildHistoryRepository interface {
	// Create creates a new build history record
	// The build history ID will be set after successful creation
	Create(ctx context.Context, b *build_history.BuildHistory) error

	// Save updates an existing build history record
	// Used to update status, summary, timestamps, etc.
	Save(ctx context.Context, b *build_history.BuildHistory) error

	// FindByID finds a build history by its ID
	// Returns ErrBuildHistoryNotFound if the build history does not exist
	FindByID(ctx context.Context, buildHistoryID uint) (*build_history.BuildHistory, error)

	// FindLatestByContainerID finds the most recent build history for a container
	// Returns ErrBuildHistoryNotFound if no build history exists for the container
	FindLatestByContainerID(ctx context.Context, containerID uint) (*build_history.BuildHistory, error)

	// FindByContainerID finds all build histories for a container with pagination
	// Returns an empty slice if no build histories exist
	// Build histories are ordered by created_at DESC (newest first)
	FindByContainerID(ctx context.Context, containerID uint, limit, offset int) ([]*build_history.BuildHistory, error)

	// FindByTektonPipelineRunName finds a build history by its Tekton PipelineRun name
	// Used for tracking build status from Tekton events
	// Returns ErrBuildHistoryNotFound if the build history does not exist
	FindByTektonPipelineRunName(ctx context.Context, pipelineRunName string) (*build_history.BuildHistory, error)

	// FindActiveByContainerID finds all active (non-completed) build histories for a container
	// Active means status is untracked, running, or backend_tracking_lost (recoverable)
	// Returns an empty slice if no active build histories exist
	FindActiveByContainerID(ctx context.Context, containerID uint) ([]*build_history.BuildHistory, error)
}
