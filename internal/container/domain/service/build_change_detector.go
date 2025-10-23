package service

import (
	"encoding/json"
	"reflect"

	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
)

// BuildChangeDetector is a domain service that detects if build parameters have changed
type BuildChangeDetector interface {
	// ShouldRebuild determines if a container needs to be rebuilt based on parameter changes
	ShouldRebuild(old, new *model.Container) bool
}

type buildChangeDetectorImpl struct{}

// NewBuildChangeDetector creates a new BuildChangeDetector instance
func NewBuildChangeDetector() BuildChangeDetector {
	return &buildChangeDetectorImpl{}
}

// ShouldRebuild checks if any build-affecting parameters have changed
// Build parameters include: template_id, template_config, git_repository_url, git_branch, git_directory_path
func (d *buildChangeDetectorImpl) ShouldRebuild(old, new *model.Container) bool {
	// Check template_id change
	if hasTemplateIDChanged(old, new) {
		return true
	}

	// Check template_config change (deep comparison)
	if hasTemplateConfigChanged(old, new) {
		return true
	}

	// Check git_repository_url change
	if old.GitConfig().RepositoryURL() != new.GitConfig().RepositoryURL() {
		return true
	}

	// Check git_branch change
	if old.GitConfig().Branch() != new.GitConfig().Branch() {
		return true
	}

	// Check git_directory_path change
	if hasGitDirectoryPathChanged(old, new) {
		return true
	}

	return false
}

// hasTemplateIDChanged checks if template_id has changed
func hasTemplateIDChanged(old, new *model.Container) bool {
	oldID := old.TemplateID()
	newID := new.TemplateID()

	// Both nil - no change
	if oldID == nil && newID == nil {
		return false
	}

	// One is nil, other is not - changed
	if (oldID == nil) != (newID == nil) {
		return true
	}

	// Both non-nil - compare values
	return *oldID != *newID
}

// hasTemplateConfigChanged checks if template_config has changed (deep JSON comparison)
func hasTemplateConfigChanged(old, new *model.Container) bool {
	oldConfig := old.TemplateConfig()
	newConfig := new.TemplateConfig()

	// Both nil - no change
	if oldConfig == nil && newConfig == nil {
		return false
	}

	// One is nil, other is not - changed
	if (oldConfig == nil) != (newConfig == nil) {
		return true
	}

	// Deep comparison using JSON serialization
	// This handles nested maps, arrays, and ensures field order doesn't matter
	oldJSON, err := json.Marshal(oldConfig)
	if err != nil {
		// If marshaling fails, assume changed to be safe
		return true
	}

	newJSON, err := json.Marshal(newConfig)
	if err != nil {
		// If marshaling fails, assume changed to be safe
		return true
	}

	// Compare JSON strings
	return !reflect.DeepEqual(oldJSON, newJSON)
}

// hasGitDirectoryPathChanged checks if git_directory_path has changed
func hasGitDirectoryPathChanged(old, new *model.Container) bool {
	oldPath := old.GitConfig().DirectoryPath()
	newPath := new.GitConfig().DirectoryPath()

	// Both nil - no change
	if oldPath == nil && newPath == nil {
		return false
	}

	// One is nil, other is not - changed
	if (oldPath == nil) != (newPath == nil) {
		return true
	}

	// Both non-nil - compare values
	return *oldPath != *newPath
}
