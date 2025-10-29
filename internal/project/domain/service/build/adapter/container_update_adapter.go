package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containerbuild "github.com/swm-launchpad/web-console-backend/internal/container/application/build"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// ContainerUpdateAdapter adapts the project-side interface to container-application use case
// This adapter translates project DTOs into container-application inputs
type ContainerUpdateAdapter struct {
	updateUseCase *containerbuild.UpdateContainerAfterBuildUseCase
	logger        logger.Logger
}

// NewContainerUpdateAdapter creates a new adapter instance
func NewContainerUpdateAdapter(
	updateUseCase *containerbuild.UpdateContainerAfterBuildUseCase,
	log logger.Logger,
) *ContainerUpdateAdapter {
	return &ContainerUpdateAdapter{
		updateUseCase: updateUseCase,
		logger:        log,
	}
}

// UpdateAfterBuild implements the ContainerUpdater interface
// This adapter method converts from the project DTO to container-application input
func (a *ContainerUpdateAdapter) UpdateAfterBuild(
	ctx context.Context,
	containerID uint,
	buildStatus string,
	commitHash string,
	snapshotBeforeBuild *dto.BuildContainerInfo,
) (wasUpdated bool, err error) {
	// Deep copy TemplateConfig to prevent snapshot aliasing
	templateConfig, copyErr := deepCopyTemplateConfig(snapshotBeforeBuild.TemplateConfig)
	if copyErr != nil {
		// Log serialization error for diagnostic purposes
		a.logger.Warn(ctx, "Failed to deep copy template config, using nil",
			zap.Uint("container_id", containerID),
			zap.Error(copyErr),
		)
		templateConfig = nil
	}

	// Deep copy BuildVars to prevent snapshot aliasing
	// Maps are reference types - must copy to prevent mutations from affecting snapshot
	buildVars := copyBuildVars(snapshotBeforeBuild.BuildVars)

	// Convert dto.BuildContainerInfo to BuildParametersSnapshot
	snapshot := &containerbuild.BuildParametersSnapshot{
		GitRepositoryURL: snapshotBeforeBuild.GitRepositoryURL,
		GitBranch:        snapshotBeforeBuild.GitBranch,
		GitDirectoryPath: snapshotBeforeBuild.GitDirectoryPath,
		TemplateID:       snapshotBeforeBuild.TemplateID,
		TemplateConfig:   templateConfig,
		BuildVars:        buildVars,
		InstallationID:   snapshotBeforeBuild.InstallationID,
	}

	// Build input and delegate to container-application use case
	input := containerbuild.UpdateContainerAfterBuildInput{
		ContainerID:    containerID,
		BuildStatus:    buildStatus,
		CommitHash:     commitHash,
		SnapshotBefore: snapshot,
	}

	return a.updateUseCase.Execute(ctx, input)
}

// deepCopyTemplateConfig performs a deep copy of template config using JSON serialization
// This ensures nested maps/slices are fully cloned, preventing snapshot aliasing
// Returns error if serialization fails to help diagnose unexpected template payloads
func deepCopyTemplateConfig(src map[string]interface{}) (map[string]interface{}, error) {
	if src == nil {
		return nil, nil
	}

	// Use JSON round-trip for deep copy
	// This handles arbitrary nesting of maps, slices, and primitives
	data, err := json.Marshal(src)
	if err != nil {
		// Return error to help diagnose unexpected template payloads
		return nil, fmt.Errorf("failed to marshal template config: %w", err)
	}

	var dst map[string]interface{}
	if err := json.Unmarshal(data, &dst); err != nil {
		// Return error to help diagnose unexpected template payloads
		return nil, fmt.Errorf("failed to unmarshal template config: %w", err)
	}

	return dst, nil
}

// copyBuildVars performs a defensive copy of build vars map
// This prevents snapshot aliasing where mutations to the source map affect the snapshot
func copyBuildVars(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}

	// Create new map and copy all key-value pairs
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}

	return dst
}
