package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	model "github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

func TestBuildChangeDetector_ShouldRebuild_TemplateIDChange(t *testing.T) {
	detector := NewBuildChangeDetector()

	// Create base container
	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpuLimit := uint32(1000)
	memLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memLimit)

	templateID1 := uint(1)
	templateID2 := uint(2)

	old := model.ReconstructContainer(
		1, 1, &templateID1, "test", slug, nil, nil, nil,
		gitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	new := model.ReconstructContainer(
		1, 1, &templateID2, "test", slug, nil, nil, nil,
		gitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	result := detector.ShouldRebuild(old, new)
	assert.True(t, result, "Should rebuild when template_id changes")
}

func TestBuildChangeDetector_ShouldRebuild_TemplateIDNilChange(t *testing.T) {
	detector := NewBuildChangeDetector()

	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpuLimit := uint32(1000)
	memLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memLimit)

	templateID := uint(1)

	// nil -> non-nil
	old := model.ReconstructContainer(
		1, 1, nil, "test", slug, nil, nil, nil,
		gitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	new := model.ReconstructContainer(
		1, 1, &templateID, "test", slug, nil, nil, nil,
		gitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	result := detector.ShouldRebuild(old, new)
	assert.True(t, result, "Should rebuild when template_id changes from nil to value")
}

func TestBuildChangeDetector_ShouldRebuild_TemplateConfigChange(t *testing.T) {
	detector := NewBuildChangeDetector()

	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpuLimit := uint32(1000)
	memLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memLimit)

	oldConfig := map[string]interface{}{
		"port":    8080,
		"env":     "production",
		"options": map[string]interface{}{"debug": false},
	}

	newConfig := map[string]interface{}{
		"port":    8080,
		"env":     "production",
		"options": map[string]interface{}{"debug": true}, // Changed
	}

	old := model.ReconstructContainer(
		1, 1, nil, "test", slug, nil, oldConfig, nil,
		gitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	new := model.ReconstructContainer(
		1, 1, nil, "test", slug, nil, newConfig, nil,
		gitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	result := detector.ShouldRebuild(old, new)
	assert.True(t, result, "Should rebuild when template_config changes")
}

func TestBuildChangeDetector_ShouldRebuild_GitRepositoryURLChange(t *testing.T) {
	detector := NewBuildChangeDetector()

	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	oldGitConfig, _ := value.NewGitConfig("https://github.com/test/repo1", "main", nil)
	newGitConfig, _ := value.NewGitConfig("https://github.com/test/repo2", "main", nil)
	cpuLimit := uint32(1000)
	memLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memLimit)

	old := model.ReconstructContainer(
		1, 1, nil, "test", slug, nil, nil, nil,
		oldGitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	new := model.ReconstructContainer(
		1, 1, nil, "test", slug, nil, nil, nil,
		newGitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	result := detector.ShouldRebuild(old, new)
	assert.True(t, result, "Should rebuild when git_repository_url changes")
}

func TestBuildChangeDetector_ShouldRebuild_GitBranchChange(t *testing.T) {
	detector := NewBuildChangeDetector()

	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	oldGitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	newGitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "develop", nil)
	cpuLimit := uint32(1000)
	memLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memLimit)

	old := model.ReconstructContainer(
		1, 1, nil, "test", slug, nil, nil, nil,
		oldGitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	new := model.ReconstructContainer(
		1, 1, nil, "test", slug, nil, nil, nil,
		newGitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	result := detector.ShouldRebuild(old, new)
	assert.True(t, result, "Should rebuild when git_branch changes")
}

func TestBuildChangeDetector_ShouldRebuild_GitDirectoryPathChange(t *testing.T) {
	detector := NewBuildChangeDetector()

	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	oldPath := "app"
	newPath := "service"
	oldGitConfig, err1 := value.NewGitConfig("https://github.com/test/repo", "main", &oldPath)
	if err1 != nil {
		t.Fatalf("Failed to create oldGitConfig: %v", err1)
	}
	newGitConfig, err2 := value.NewGitConfig("https://github.com/test/repo", "main", &newPath)
	if err2 != nil {
		t.Fatalf("Failed to create newGitConfig: %v", err2)
	}
	cpuLimit := uint32(1000)
	memLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memLimit)

	old := model.ReconstructContainer(
		1, 1, nil, "test", slug, nil, nil, nil,
		oldGitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	new := model.ReconstructContainer(
		1, 1, nil, "test", slug, nil, nil, nil,
		newGitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	result := detector.ShouldRebuild(old, new)
	assert.True(t, result, "Should rebuild when git_directory_path changes")
}

func TestBuildChangeDetector_ShouldNotRebuild_NoChanges(t *testing.T) {
	detector := NewBuildChangeDetector()

	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	cpuLimit := uint32(1000)
	memLimit := uint32(2048)
	resourceLimits, _ := value.NewResourceLimits(&cpuLimit, &memLimit)

	templateID := uint(1)
	config := map[string]interface{}{
		"port": 8080,
		"env":  "production",
	}

	old := model.ReconstructContainer(
		1, 1, &templateID, "test", slug, nil, config, nil,
		gitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	new := model.ReconstructContainer(
		1, 1, &templateID, "test", slug, nil, config, nil,
		gitConfig, nil, nil, false, resourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	result := detector.ShouldRebuild(old, new)
	assert.False(t, result, "Should not rebuild when no build parameters change")
}

func TestBuildChangeDetector_ShouldNotRebuild_OnlyNonBuildParametersChange(t *testing.T) {
	detector := NewBuildChangeDetector()

	slug, _ := value.NewContainerSlug("c2025011812000012345678")
	gitConfig, _ := value.NewGitConfig("https://github.com/test/repo", "main", nil)
	oldCPU := uint32(1000)
	newCPU := uint32(2000)
	memLimit := uint32(2048)
	oldResourceLimits, _ := value.NewResourceLimits(&oldCPU, &memLimit)
	newResourceLimits, _ := value.NewResourceLimits(&newCPU, &memLimit)

	old := model.ReconstructContainer(
		1, 1, nil, "old-name", slug, nil, nil, nil,
		gitConfig, nil, nil, false, oldResourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	new := model.ReconstructContainer(
		1, 1, nil, "new-name", slug, nil, nil, nil,
		gitConfig, nil, nil, false, newResourceLimits, nil, nil, nil,
		nil, false, false, nil, time.Now(), time.Now(),
	)

	result := detector.ShouldRebuild(old, new)
	assert.False(t, result, "Should not rebuild when only non-build parameters (name, cpu_limit) change")
}
