package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/container/value"
)

func defaultGitConfig() value.GitConfig {
	gitConfig, _ := value.NewGitConfig("https://github.com/user/repo.git", "main", nil)
	return gitConfig
}

func defaultResourceLimits() value.ResourceLimits {
	cpu := uint32(1000)
	memory := uint32(512)
	limits, _ := value.NewResourceLimits(&cpu, &memory)
	return limits
}

func TestNewContainer_Success(t *testing.T) {
	projectID := uint(1)
	name := "Backend API"
	slug, _ := value.NewContainerSlug("backend-api")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, nil, nil)

	require.NoError(t, err)
	assert.NotNil(t, container)
	assert.Equal(t, projectID, container.ProjectID())
	assert.Equal(t, name, container.Name())
	assert.Equal(t, slug, container.Slug())
	assert.Equal(t, gitConfig, container.GitConfig())
	assert.Equal(t, resourceLimits, container.ResourceLimits())
	assert.False(t, container.IsDeleted())
	assert.NotZero(t, container.CreatedAt())
	assert.True(t, container.UpdatedAt().IsZero())
	assert.Empty(t, container.EnvVars())
	assert.Empty(t, container.Networks())
}

func TestNewContainer_WithTemplate(t *testing.T) {
	projectID := uint(1)
	name := "Backend API"
	slug, _ := value.NewContainerSlug("backend-api")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()
	templateID := uint(10)
	templateConfig := map[string]interface{}{
		"framework": "express",
		"version":   "4.18.0",
	}

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, &templateID, templateConfig)

	require.NoError(t, err)
	assert.NotNil(t, container)
	assert.Equal(t, &templateID, container.TemplateID())
	assert.Equal(t, templateConfig, container.TemplateConfig())
}

func TestNewContainer_InvalidInputs(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	t.Run("Invalid project ID", func(t *testing.T) {
		container, err := NewContainer(0, "Backend API", slug, gitConfig, resourceLimits, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrInvalidProjectID)
		assert.Nil(t, container)
	})

	t.Run("Empty name", func(t *testing.T) {
		container, err := NewContainer(1, "", slug, gitConfig, resourceLimits, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrNameRequired)
		assert.Nil(t, container)
	})

	t.Run("Name too long", func(t *testing.T) {
		longName := string(make([]byte, 101))
		for range longName {
			longName = "a" + longName[1:]
		}
		container, err := NewContainer(1, longName, slug, gitConfig, resourceLimits, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrNameTooLong)
		assert.Nil(t, container)
	})
}

func TestContainer_SetContainerID(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)

	assert.Equal(t, uint(0), container.ContainerID())

	container.SetContainerID(999)
	assert.Equal(t, uint(999), container.ContainerID())

	// EnvVars and Networks should also have updated containerID
	_, _ = container.AddEnvVar("APP_ENV", "production")
	envVars := container.EnvVars()
	assert.Equal(t, uint(999), envVars[0].ContainerID())
}

func TestContainer_AddEnvVar(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)
	container.SetContainerID(1)

	t.Run("Success", func(t *testing.T) {
		_, err := container.AddEnvVar("APP_ENV", "production")
		require.NoError(t, err)
		assert.Len(t, container.EnvVars(), 1)
		assert.True(t, container.HasEnvVar("APP_ENV"))
	})

	t.Run("Duplicate key", func(t *testing.T) {
		_, err := container.AddEnvVar("APP_ENV", "development")
		assert.ErrorIs(t, err, containererrors.ErrDuplicateEnvVarKey)
	})

	t.Run("Max env vars exceeded", func(t *testing.T) {
		// Add 99 more env vars (already have 1)
		for i := 1; i < MaxEnvVarsPerContainer; i++ {
			key := "VAR_" + string(rune('A'+i%26)) + string(rune('0'+i/26))
			_, _ = container.AddEnvVar(key, "value")
		}
		assert.Len(t, container.EnvVars(), MaxEnvVarsPerContainer)

		// Try to add one more
		_, err := container.AddEnvVar("EXTRA_VAR", "value")
		assert.ErrorIs(t, err, containererrors.ErrMaxEnvVarsExceeded)
	})
}

func TestContainer_UpdateEnvVar(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)
	container.SetContainerID(1)
	_, _ = container.AddEnvVar("APP_ENV", "production")

	t.Run("Success", func(t *testing.T) {
		err := container.UpdateEnvVar("APP_ENV", "development")
		require.NoError(t, err)

		envVar, _ := container.GetEnvVar("APP_ENV")
		assert.Equal(t, "development", envVar.Value())
	})

	t.Run("Env var not found", func(t *testing.T) {
		err := container.UpdateEnvVar("NONEXISTENT", "value")
		assert.ErrorIs(t, err, containererrors.ErrEnvVarNotFound)
	})
}

func TestContainer_DeleteEnvVar(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)
	container.SetContainerID(1)
	_, _ = container.AddEnvVar("APP_ENV", "production")
	_, _ = container.AddEnvVar("DEBUG", "true")

	t.Run("Success", func(t *testing.T) {
		err := container.DeleteEnvVar("APP_ENV")
		require.NoError(t, err)
		assert.Len(t, container.EnvVars(), 1)
		assert.False(t, container.HasEnvVar("APP_ENV"))
		assert.True(t, container.HasEnvVar("DEBUG"))
	})

	t.Run("Env var not found", func(t *testing.T) {
		err := container.DeleteEnvVar("NONEXISTENT")
		assert.ErrorIs(t, err, containererrors.ErrEnvVarNotFound)
	})
}

func TestContainer_AddNetwork(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)
	container.SetContainerID(1)

	internalPort := uint16(8080)
	externalPort := uint16(80)
	networkType, _ := value.NewNetworkType("http")
	externalIP := "0.0.0.0"

	t.Run("Success", func(t *testing.T) {
		_, err := container.AddNetwork(&internalPort, &externalPort, networkType, &externalIP, nil)
		require.NoError(t, err)
		assert.Len(t, container.Networks(), 1)
	})

	t.Run("Duplicate internal port", func(t *testing.T) {
		_, err := container.AddNetwork(&internalPort, nil, networkType, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrDuplicateInternalPort)
	})

	t.Run("Max networks exceeded", func(t *testing.T) {
		// Add 19 more networks (already have 1)
		for i := 1; i < MaxNetworksPerContainer; i++ {
			port := uint16(8081 + i)
			_, _ = container.AddNetwork(&port, nil, networkType, nil, nil)
		}
		assert.Len(t, container.Networks(), MaxNetworksPerContainer)

		// Try to add one more
		extraPort := uint16(9999)
		_, err := container.AddNetwork(&extraPort, nil, networkType, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrMaxNetworksExceeded)
	})
}

func TestContainer_DeleteNetwork(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)
	container.SetContainerID(1)

	port1 := uint16(8080)
	port2 := uint16(3000)
	networkType, _ := value.NewNetworkType("tcp")

	_, _ = container.AddNetwork(&port1, nil, networkType, nil, nil)
	_, _ = container.AddNetwork(&port2, nil, networkType, nil, nil)

	t.Run("Success", func(t *testing.T) {
		err := container.DeleteNetworkByInternalPort(port1)
		require.NoError(t, err)
		assert.Len(t, container.Networks(), 1)
	})

	t.Run("Network not found", func(t *testing.T) {
		nonExistentPort := uint16(9999)
		err := container.DeleteNetworkByInternalPort(nonExistentPort)
		assert.ErrorIs(t, err, containererrors.ErrNetworkNotInContainer)
	})
}

func TestContainer_UpdateGitConfig(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)

	newGitConfig, _ := value.NewGitConfig("https://github.com/user/new-repo.git", "develop", nil)

	err := container.UpdateGitConfig(newGitConfig)
	require.NoError(t, err)
	assert.Equal(t, newGitConfig, container.GitConfig())
	assert.False(t, container.UpdatedAt().IsZero())
}

func TestContainer_UpdateResourceLimits(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)

	newCPU := uint32(2000)
	newMemory := uint32(1024)
	newLimits, _ := value.NewResourceLimits(&newCPU, &newMemory)

	err := container.UpdateResourceLimits(newLimits)
	require.NoError(t, err)
	assert.Equal(t, newLimits, container.ResourceLimits())
	assert.False(t, container.UpdatedAt().IsZero())
}

func TestContainer_UpdateTemplateConfig(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)

	templateID := uint(10)
	templateConfig := map[string]interface{}{
		"framework": "express",
		"version":   "4.18.0",
	}

	err := container.UpdateTemplateConfig(&templateID, templateConfig)
	require.NoError(t, err)
	assert.Equal(t, &templateID, container.TemplateID())
	assert.Equal(t, templateConfig, container.TemplateConfig())
	assert.False(t, container.UpdatedAt().IsZero())
}

func TestContainer_SoftDelete(t *testing.T) {
	slug, _ := value.NewContainerSlug("backend-api")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil)

	t.Run("Success", func(t *testing.T) {
		err := container.SoftDelete()
		require.NoError(t, err)
		assert.True(t, container.IsDeleted())
		assert.NotNil(t, container.DeletedAt())
	})

	t.Run("Already deleted", func(t *testing.T) {
		firstDeletedAt := container.DeletedAt()
		err := container.SoftDelete()
		require.NoError(t, err)
		assert.Equal(t, firstDeletedAt, container.DeletedAt())
	})
}

func TestReconstructContainer(t *testing.T) {
	containerID := uint(100)
	projectID := uint(1)
	name := "Backend API"
	slug, _ := value.NewContainerSlug("backend-api")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	container := ReconstructContainer(
		containerID,
		projectID,
		nil, // templateID
		name,
		slug,
		nil, // stableWindow
		nil, // templateConfig
		gitConfig,
		nil, // gitCommitHash
		nil, // lastBuiltGitCommitHash
		resourceLimits,
		nil, // monthlyBuildTime
		nil, // monthlyBuildCount
		nil, // monthlyUptime
		false,
		nil,        // deletedAt
		time.Now(), // createdAt
		time.Now(), // updatedAt
	)

	assert.NotNil(t, container)
	assert.Equal(t, containerID, container.ContainerID())
	assert.Equal(t, projectID, container.ProjectID())
	assert.Equal(t, name, container.Name())
	assert.False(t, container.IsDeleted())
}
