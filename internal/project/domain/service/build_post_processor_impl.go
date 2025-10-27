package service

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containerrepo "github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	containermodel "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// buildPostProcessorImpl implements BuildPostProcessor
type buildPostProcessorImpl struct {
	containerRepo containerrepo.ContainerRepository
	txManager     db.TxManager
	logger        logger.Logger
}

// NewBuildPostProcessor creates a new BuildPostProcessor instance
func NewBuildPostProcessor(
	containerRepo containerrepo.ContainerRepository,
	txManager db.TxManager,
	log logger.Logger,
) BuildPostProcessor {
	return &buildPostProcessorImpl{
		containerRepo: containerRepo,
		txManager:     txManager,
		logger:        log,
	}
}

// UpdateContainerAfterBuild updates container information after a successful build
func (p *buildPostProcessorImpl) UpdateContainerAfterBuild(
	ctx context.Context,
	containerID uint,
	buildResult *BuildResult,
	snapshotBeforeBuild *dto.BuildContainerInfo,
) error {
	p.logger.Info(ctx, "Starting post-build container update",
		zap.Uint("container_id", containerID),
		zap.Uint("build_history_id", buildResult.BuildHistoryID),
		zap.String("build_status", buildResult.Status),
	)

	// Only update container if build was successful
	if buildResult.Status != "success" {
		p.logger.Info(ctx, "Skipping container update due to non-success build status",
			zap.Uint("container_id", containerID),
			zap.String("status", buildResult.Status),
		)
		return nil
	}

	// Execute update in transaction
	err := p.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Step 1: Acquire row lock and fetch current container state
		currentContainer, err := p.containerRepo.FindByIDForUpdate(txCtx, containerID)
		if err != nil {
			p.logger.Error(txCtx, "Failed to fetch container with lock",
				zap.Error(err),
				zap.Uint("container_id", containerID),
			)
			return fmt.Errorf("failed to fetch container with lock: %w", err)
		}

		// Step 2: Compare snapshot with current state to detect changes
		hasChanged := p.hasBuildParametersChanged(snapshotBeforeBuild, currentContainer)

		if hasChanged {
			p.logger.Warn(txCtx, "Build parameters changed during build, skipping update",
				zap.Uint("container_id", containerID),
			)
			return nil
		}

		// Step 3: Update container state after successful build
		p.logger.Info(txCtx, "Updating container after successful build",
			zap.Uint("container_id", containerID),
			zap.String("commit_hash", buildResult.LatestCommitHash),
		)

		// Update last built commit hash
		currentContainer.SetLastBuiltCommitHash(&buildResult.LatestCommitHash)

		// Clear needs_build flag
		currentContainer.ClearNeedsBuild()

		// Step 4: Save updated container
		if err := p.containerRepo.Save(txCtx, currentContainer); err != nil {
			p.logger.Error(txCtx, "Failed to save container after build",
				zap.Error(err),
				zap.Uint("container_id", containerID),
			)
			return fmt.Errorf("failed to save container: %w", err)
		}

		p.logger.Info(txCtx, "Successfully updated container after build",
			zap.Uint("container_id", containerID),
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
func (p *buildPostProcessorImpl) hasBuildParametersChanged(
	snapshot *dto.BuildContainerInfo,
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

	// Check template ID
	currentTemplateID := current.TemplateID()
	if !areOptionalUintsEqual(snapshot.TemplateBody, currentTemplateID) {
		// Note: TemplateBody in snapshot indicates template usage
		// If snapshot has TemplateBody (not nil), it means template was used
		// We compare with current TemplateID existence
		snapshotHasTemplate := (snapshot.TemplateBody != nil)
		currentHasTemplate := (currentTemplateID != nil)
		if snapshotHasTemplate != currentHasTemplate {
			return true
		}
	}

	// Check template config (deep comparison using JSON)
	if !areTemplateConfigsEqual(snapshot.TemplateConfig, current.TemplateConfig()) {
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

// areOptionalUintsEqual compares template existence
// Since we can't directly compare template body with ID, we check existence
func areOptionalUintsEqual(templateBody *string, templateID *uint) bool {
	// If both are nil or both are non-nil, consider them equal
	// This is a simplified check - actual template content comparison would require
	// fetching the template body from the database
	hasTemplateBody := (templateBody != nil)
	hasTemplateID := (templateID != nil)
	return hasTemplateBody == hasTemplateID
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
