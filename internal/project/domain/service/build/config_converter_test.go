package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

func TestConvertToBuildConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := ConvertToBuildConfig(nil)
		assert.Nil(t, result)
	})

	t.Run("empty containers", func(t *testing.T) {
		unified := &dto.UnifiedContainerConfig{
			ProjectID:  1,
			Containers: []dto.UnifiedContainerInfo{},
		}
		result := ConvertToBuildConfig(unified)
		assert.NotNil(t, result)
		assert.Empty(t, result.Containers)
	})

	t.Run("converts unified config to build config", func(t *testing.T) {
		templateID := uint(10)
		templateBody := "FROM node:18"
		gitDirectoryPath := "/app"
		lastBuiltCommitHash := "abc123"
		installationID := int64(12345)

		unified := &dto.UnifiedContainerConfig{
			ProjectID: 1,
			Containers: []dto.UnifiedContainerInfo{
				{
					ContainerID:          100,
					Name:                 "test-container",
					Slug:                 "test-slug",
					TemplateID:           &templateID,
					TemplateBody:         &templateBody,
					TemplateConfig:       map[string]interface{}{"key": "value"},
					GitRepositoryURL:     "https://github.com/test/repo",
					GitBranch:            "main",
					GitDirectoryPath:     &gitDirectoryPath,
					LastBuiltCommitHash:  &lastBuiltCommitHash,
					NeedsBuild:           true,
					BuildVars:            map[string]string{"BUILD_ENV": "production"},
					GitHubInstallationID: &installationID,
				},
			},
		}

		result := ConvertToBuildConfig(unified)

		assert.NotNil(t, result)
		assert.Len(t, result.Containers, 1)

		container := result.Containers[0]
		assert.Equal(t, uint(100), container.ContainerID)
		assert.Equal(t, "test-container", container.Name)
		assert.Equal(t, "test-slug", container.Slug)
		assert.Equal(t, &templateID, container.TemplateID)
		assert.Equal(t, &templateBody, container.TemplateBody)
		assert.Equal(t, map[string]interface{}{"key": "value"}, container.TemplateConfig)
		assert.Equal(t, "https://github.com/test/repo", container.GitRepositoryURL)
		assert.Equal(t, "main", container.GitBranch)
		assert.Equal(t, &gitDirectoryPath, container.GitDirectoryPath)
		assert.Equal(t, &lastBuiltCommitHash, container.LastBuiltCommitHash)
		assert.True(t, container.NeedsBuild)
		assert.Equal(t, map[string]string{"BUILD_ENV": "production"}, container.BuildVars)
		assert.Equal(t, &installationID, container.InstallationID)
	})
}

func TestConvertToDeployConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := ConvertToDeployConfig(nil, nil)
		assert.Nil(t, result)
	})

	t.Run("converts unified config to deploy config without build results", func(t *testing.T) {
		cpuLimit := uint32(1000)
		memoryLimit := uint32(512)

		unified := &dto.UnifiedContainerConfig{
			ProjectID: 1,
			Containers: []dto.UnifiedContainerInfo{
				{
					ContainerID: 100,
					Slug:        "test-slug",
					ImageName:   "test-image",
					ImageTag:    "v1.0.0",
					CPULimit:    &cpuLimit,
					MemoryLimit: &memoryLimit,
					EnvVars:     map[string]string{"ENV": "prod"},
					Secrets:     map[string]string{"SECRET": "value"},
					Port:        8080,
				},
			},
		}

		result := ConvertToDeployConfig(unified, nil)

		assert.NotNil(t, result)
		assert.Len(t, result.Containers, 1)

		container := result.Containers[0]
		assert.Equal(t, "test-slug", container.Name)
		assert.Equal(t, "test-image", container.ImageName)
		assert.Equal(t, "v1.0.0", container.ImageTag)
		assert.Equal(t, "1000m", container.CPULimit)
		assert.Equal(t, "512Mi", container.MemoryRequest) // Key test: 512 MiB should be "512Mi", not "0Mi"
		assert.Equal(t, "512Mi", container.MemoryLimit)   // Key test: 512 MiB should be "512Mi", not "0Mi"
		assert.Equal(t, map[string]string{"ENV": "prod"}, container.EnvVars)
		assert.Equal(t, map[string]string{"SECRET": "value"}, container.Secrets)
	})

	t.Run("updates image tag from build results", func(t *testing.T) {
		unified := &dto.UnifiedContainerConfig{
			ProjectID: 1,
			Containers: []dto.UnifiedContainerInfo{
				{
					ContainerID: 100,
					Slug:        "test-slug",
					ImageName:   "test-image",
					ImageTag:    "old-tag",
					Port:        8080,
				},
			},
		}

		buildResults := []*BuildResult{
			{
				ContainerID:      100,
				Status:           "success",
				LatestCommitHash: "abc1234567",
			},
		}

		result := ConvertToDeployConfig(unified, buildResults)

		assert.NotNil(t, result)
		assert.Len(t, result.Containers, 1)
		assert.Equal(t, "abc1234", result.Containers[0].ImageTag) // First 7 chars of commit hash
	})

	t.Run("handles nil memory and CPU limits with defaults", func(t *testing.T) {
		unified := &dto.UnifiedContainerConfig{
			ProjectID: 1,
			Containers: []dto.UnifiedContainerInfo{
				{
					ContainerID: 100,
					Slug:        "test-slug",
					ImageName:   "test-image",
					ImageTag:    "v1.0.0",
					CPULimit:    nil,
					MemoryLimit: nil,
					Port:        8080,
				},
			},
		}

		result := ConvertToDeployConfig(unified, nil)

		assert.NotNil(t, result)
		assert.Len(t, result.Containers, 1)

		container := result.Containers[0]
		assert.Equal(t, "1000m", container.CPULimit)      // Default 1 core
		assert.Equal(t, "512Mi", container.MemoryRequest) // Default 512Mi
		assert.Equal(t, "1Gi", container.MemoryLimit)     // Default 1Gi
	})

	t.Run("handles large memory limits correctly", func(t *testing.T) {
		memoryLimit := uint32(1024) // 1024 MiB

		unified := &dto.UnifiedContainerConfig{
			ProjectID: 1,
			Containers: []dto.UnifiedContainerInfo{
				{
					ContainerID: 100,
					Slug:        "test-slug",
					ImageName:   "test-image",
					ImageTag:    "v1.0.0",
					MemoryLimit: &memoryLimit,
					Port:        8080,
				},
			},
		}

		result := ConvertToDeployConfig(unified, nil)

		assert.NotNil(t, result)
		assert.Len(t, result.Containers, 1)

		container := result.Containers[0]
		// Key test: 1024 MiB should be "1024Mi", not "0Mi" or "1Mi"
		assert.Equal(t, "1024Mi", container.MemoryRequest)
		assert.Equal(t, "1024Mi", container.MemoryLimit)
	})
}

func TestFormatMemoryLimit(t *testing.T) {
	tests := []struct {
		name     string
		input    *uint32
		expected string
	}{
		{
			name:     "nil returns default",
			input:    nil,
			expected: "1Gi",
		},
		{
			name:     "512 MiB",
			input:    ptrUint32(512),
			expected: "512Mi",
		},
		{
			name:     "1024 MiB",
			input:    ptrUint32(1024),
			expected: "1024Mi",
		},
		{
			name:     "256 MiB",
			input:    ptrUint32(256),
			expected: "256Mi",
		},
		{
			name:     "2048 MiB",
			input:    ptrUint32(2048),
			expected: "2048Mi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMemoryLimit(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatMemoryRequest(t *testing.T) {
	tests := []struct {
		name     string
		input    *uint32
		expected string
	}{
		{
			name:     "nil returns default",
			input:    nil,
			expected: "512Mi",
		},
		{
			name:     "512 MiB",
			input:    ptrUint32(512),
			expected: "512Mi",
		},
		{
			name:     "1024 MiB",
			input:    ptrUint32(1024),
			expected: "1024Mi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMemoryRequest(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCPULimit(t *testing.T) {
	tests := []struct {
		name     string
		input    *uint32
		expected string
	}{
		{
			name:     "nil returns default",
			input:    nil,
			expected: "1000m",
		},
		{
			name:     "1000 millicores",
			input:    ptrUint32(1000),
			expected: "1000m",
		},
		{
			name:     "2000 millicores",
			input:    ptrUint32(2000),
			expected: "2000m",
		},
		{
			name:     "500 millicores",
			input:    ptrUint32(500),
			expected: "500m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCPULimit(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create pointer to uint32
func ptrUint32(v uint32) *uint32 {
	return &v
}
