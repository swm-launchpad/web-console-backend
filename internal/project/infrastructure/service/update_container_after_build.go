package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	containermodel "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// UpdateContainerAfterBuildInput contains input parameters for container update after build
type UpdateContainerAfterBuildInput struct {
	ContainerID    uint
	BuildStatus    string
	CommitHash     string
	SnapshotBefore *BuildParametersSnapshot
}

// UpdateContainerAfterBuildUseCase handles container state updates after build completion
type UpdateContainerAfterBuildUseCase struct {
	containerRepo repository.ContainerRepository
	txManager     db.TxManager
	logger        logger.Logger
}

// NewUpdateContainerAfterBuildUseCase creates a new use case instance
func NewUpdateContainerAfterBuildUseCase(
	containerRepo repository.ContainerRepository,
	txManager db.TxManager,
	log logger.Logger,
) *UpdateContainerAfterBuildUseCase {
	return &UpdateContainerAfterBuildUseCase{
		containerRepo: containerRepo,
		txManager:     txManager,
		logger:        log,
	}
}

// Execute updates container after build completion
func (uc *UpdateContainerAfterBuildUseCase) Execute(
	ctx context.Context,
	input UpdateContainerAfterBuildInput,
) error {
	uc.logger.Info(ctx, "Starting post-build container update",
		zap.Uint("container_id", input.ContainerID),
		zap.String("build_status", input.BuildStatus),
	)

	// Only update container if build was successful or skipped
	// Skipped builds (should_build=false) also need to clear needs_build flag
	if input.BuildStatus != "success" && input.BuildStatus != "skipped" {
		uc.logger.Info(ctx, "Skipping container update due to non-success build status",
			zap.Uint("container_id", input.ContainerID),
			zap.String("status", input.BuildStatus),
		)
		return nil
	}

	// Execute update in transaction
	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Step 1: Acquire row lock and fetch current container state
		currentContainer, err := uc.containerRepo.FindByIDForUpdate(txCtx, input.ContainerID)
		if err != nil {
			uc.logger.Error(txCtx, "Failed to fetch container with lock",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
			)
			return fmt.Errorf("failed to fetch container with lock: %w", err)
		}

		// Step 2: Compare snapshot with current state to detect changes
		hasChanged := uc.hasBuildParametersChanged(input.SnapshotBefore, currentContainer)

		if hasChanged {
			uc.logger.Warn(txCtx, "Build parameters changed during build, skipping update",
				zap.Uint("container_id", input.ContainerID),
			)
			return nil
		}

		// Step 3: Update container state after successful/skipped build
		uc.logger.Info(txCtx, "Updating container after build",
			zap.Uint("container_id", input.ContainerID),
			zap.String("status", input.BuildStatus),
			zap.String("commit_hash", input.CommitHash),
		)

		// Update last built commit hash only for successful builds (not skipped)
		// Skipped builds (should_build=false) keep the existing commit hash
		if input.BuildStatus == "success" {
			// Update commit hash only if not empty
			// Empty hash can occur with pipeline bugs
			if input.CommitHash != "" {
				currentContainer.SetLastBuiltCommitHash(&input.CommitHash)
			} else {
				uc.logger.Warn(txCtx, "Build returned empty commit hash, keeping previous value",
					zap.Uint("container_id", input.ContainerID),
				)
			}
		} else {
			// Skipped build - keep existing commit hash
			uc.logger.Info(txCtx, "Skipped build - keeping existing commit hash",
				zap.Uint("container_id", input.ContainerID),
			)
		}

		// Clear needs_build flag for both success and skipped
		currentContainer.ClearNeedsBuild()

		// Step 4: Save updated container
		if err := uc.containerRepo.Save(txCtx, currentContainer); err != nil {
			uc.logger.Error(txCtx, "Failed to save container after build",
				zap.Error(err),
				zap.Uint("container_id", input.ContainerID),
			)
			return fmt.Errorf("failed to save container: %w", err)
		}

		uc.logger.Info(txCtx, "Successfully updated container after build",
			zap.Uint("container_id", input.ContainerID),
			zap.Bool("needs_build", currentContainer.NeedsBuild()),
		)

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to update container after build: %w", err)
	}

	return nil
}

// hasBuildParametersChanged compares build parameters between snapshot and current state
// Returns true if any build-affecting parameter has changed
func (uc *UpdateContainerAfterBuildUseCase) hasBuildParametersChanged(
	snapshot *BuildParametersSnapshot,
	current *containermodel.Container,
) bool {
	// Check git repository URL
	if snapshot.GitRepositoryURL != current.GitConfig().RepositoryURL() {
		return true
	}

	// Check git branch
	if snapshot.GitBranch != current.GitConfig().Branch() {
		return true
	}

	// Check git directory path
	currentGitDirPath := current.GitConfig().DirectoryPath()
	if !areOptionalStringsEqual(snapshot.GitDirectoryPath, currentGitDirPath) {
		return true
	}

	// Check template ID (actual value comparison)
	currentTemplateID := current.TemplateID()
	if !areTemplateIDsEqual(snapshot.TemplateID, currentTemplateID) {
		return true
	}

	// Check template config (deep comparison using JSON)
	if !areTemplateConfigsEqual(snapshot.TemplateConfig, current.TemplateConfig()) {
		return true
	}

	// Check build-time environment variables
	if !areBuildVarsEqual(snapshot.BuildVars, current.BuildVars()) {
		return true
	}

	// Check GitHub installation ID (for private repository access)
	currentInstallationID := current.GitHubInstallationID()
	//nolint:staticcheck // S1008: Keeping consistent with other parameter checks above
	if !areInstallationIDsEqual(snapshot.InstallationID, currentInstallationID) {
		return true
	}

	return false
}

// areOptionalStringsEqual compares two optional string pointers
func areOptionalStringsEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// areTemplateIDsEqual compares two optional template ID pointers
func areTemplateIDsEqual(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// areInstallationIDsEqual compares two optional GitHub installation ID pointers
func areInstallationIDsEqual(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// areBuildVarsEqual compares snapshot BuildVars (map) with current BuildVars (slice)
func areBuildVarsEqual(snapshotVars map[string]string, currentVars []containermodel.BuildVar) bool {
	// Convert current BuildVars slice to map for comparison
	currentVarsMap := make(map[string]string, len(currentVars))
	for _, v := range currentVars {
		currentVarsMap[v.Key()] = v.Value()
	}

	// Handle nil or empty maps
	snapshotLen := len(snapshotVars)
	currentLen := len(currentVarsMap)

	// Both empty - equal
	if snapshotLen == 0 && currentLen == 0 {
		return true
	}

	// Different sizes - not equal
	if snapshotLen != currentLen {
		return false
	}

	// Compare each key-value pair
	for key, snapshotValue := range snapshotVars {
		currentValue, exists := currentVarsMap[key]
		if !exists || currentValue != snapshotValue {
			return false
		}
	}

	return true
}

// areTemplateConfigsEqual compares two template configs using JSON serialization
func areTemplateConfigsEqual(a, b map[string]interface{}) bool {
	// Both nil or empty - consider equal
	if len(a) == 0 && len(b) == 0 {
		return true
	}

	// One is empty, other is not
	if (len(a) == 0) != (len(b) == 0) {
		return false
	}

	// Deep comparison using JSON
	aJSON, err := json.Marshal(a)
	if err != nil {
		// If marshaling fails, assume changed to be safe
		return false
	}

	bJSON, err := json.Marshal(b)
	if err != nil {
		// If marshaling fails, assume changed to be safe
		return false
	}

	return string(aJSON) == string(bJSON)
}

// UpdateAfterBuild implements the ContainerUpdater interface
// This adapter method converts from the domain interface to the use case input
func (uc *UpdateContainerAfterBuildUseCase) UpdateAfterBuild(
	ctx context.Context,
	containerID uint,
	buildStatus string,
	commitHash string,
	snapshotBeforeBuild *dto.BuildContainerInfo,
) error {
	// Convert dto.BuildContainerInfo to BuildParametersSnapshot
	// Deep copy TemplateConfig to prevent snapshot aliasing
	snapshot := &BuildParametersSnapshot{
		GitRepositoryURL: snapshotBeforeBuild.GitRepositoryURL,
		GitBranch:        snapshotBeforeBuild.GitBranch,
		GitDirectoryPath: snapshotBeforeBuild.GitDirectoryPath,
		TemplateID:       snapshotBeforeBuild.TemplateID,
		TemplateConfig:   deepCopyTemplateConfig(snapshotBeforeBuild.TemplateConfig),
		BuildVars:        snapshotBeforeBuild.BuildVars,
		InstallationID:   snapshotBeforeBuild.InstallationID,
	}

	// Build input and execute
	input := UpdateContainerAfterBuildInput{
		ContainerID:    containerID,
		BuildStatus:    buildStatus,
		CommitHash:     commitHash,
		SnapshotBefore: snapshot,
	}

	return uc.Execute(ctx, input)
}

// deepCopyTemplateConfig performs a deep copy of template config using JSON serialization
// This ensures nested maps/slices are fully cloned, preventing snapshot aliasing
func deepCopyTemplateConfig(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}

	// Use JSON round-trip for deep copy
	// This handles arbitrary nesting of maps, slices, and primitives
	data, err := json.Marshal(src)
	if err != nil {
		// If marshaling fails, return nil to preserve nil/empty distinction
		// This prevents Tekton from receiving {} when no config was supplied
		return nil
	}

	var dst map[string]interface{}
	if err := json.Unmarshal(data, &dst); err != nil {
		// If unmarshaling fails, return nil to preserve nil/empty distinction
		// This prevents Tekton from receiving {} when no config was supplied
		return nil
	}

	return dst
}
