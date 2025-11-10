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
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, nil, nil, nil)
	container.SetContainerID(1)

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
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()
	templateID := uint(10)
	templateConfig := map[string]interface{}{
		"framework": "express",
		"version":   "4.18.0",
	}

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, &templateID, templateConfig, nil)

	require.NoError(t, err)
	assert.NotNil(t, container)
	assert.Equal(t, &templateID, container.TemplateID())
	assert.Equal(t, templateConfig, container.TemplateConfig())
}

func TestNewContainer_InvalidInputs(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	t.Run("Invalid project ID", func(t *testing.T) {
		container, err := NewContainer(0, "Backend API", slug, gitConfig, resourceLimits, nil, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrInvalidProjectID)
		assert.Nil(t, container)
	})

	t.Run("Empty name", func(t *testing.T) {
		container, err := NewContainer(1, "", slug, gitConfig, resourceLimits, nil, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrNameRequired)
		assert.Nil(t, container)
	})

	t.Run("Name too long", func(t *testing.T) {
		longName := string(make([]byte, 256))
		for i := range longName {
			longName = longName[:i] + "a" + longName[i+1:]
		}
		container, err := NewContainer(1, longName, slug, gitConfig, resourceLimits, nil, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrNameTooLong)
		assert.Nil(t, container)
	})
}

func TestContainer_SetContainerID(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)

	assert.Equal(t, uint(0), container.ContainerID())

	container.SetContainerID(999)
	assert.Equal(t, uint(999), container.ContainerID())

	// EnvVars and Networks should also have updated containerID
	_, _ = container.AddEnvVar("APP_ENV", "production")
	envVars := container.EnvVars()
	assert.Equal(t, uint(999), envVars[0].ContainerID())
}

func TestContainer_AddEnvVar(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
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
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
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
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
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
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
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
		// Since max is 1 and we already have 1 network, we can't test duplicate port
		// The max networks check happens before duplicate check
		// This test is now covered by the max networks test
		t.Skip("Skipped: with MaxNetworksPerContainer=1, duplicate check is unreachable")
	})

	t.Run("Max networks exceeded", func(t *testing.T) {
		// Already have 1 network from setup, which is the max
		assert.Len(t, container.Networks(), MaxNetworksPerContainer)

		// Try to add one more (should fail since max is 1)
		extraPort := uint16(9999)
		_, err := container.AddNetwork(&extraPort, nil, networkType, nil, nil)
		assert.ErrorIs(t, err, containererrors.ErrMaxNetworksExceeded)
	})
}

func TestContainer_DeleteNetwork(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
	container.SetContainerID(1)

	port1 := uint16(8080)
	networkType, _ := value.NewNetworkType("tcp")

	// Add only 1 network (max allowed)
	_, _ = container.AddNetwork(&port1, nil, networkType, nil, nil)

	t.Run("Success", func(t *testing.T) {
		err := container.DeleteNetworkByInternalPort(port1)
		require.NoError(t, err)
		assert.Len(t, container.Networks(), 0)
	})

	t.Run("Network not found", func(t *testing.T) {
		nonExistentPort := uint16(9999)
		err := container.DeleteNetworkByInternalPort(nonExistentPort)
		assert.ErrorIs(t, err, containererrors.ErrNetworkNotInContainer)
	})
}

func TestContainer_UpdateGitConfig(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)

	newGitConfig, _ := value.NewGitConfig("https://github.com/user/new-repo.git", "develop", nil)

	err := container.UpdateGitConfig(newGitConfig)
	require.NoError(t, err)
	assert.Equal(t, newGitConfig, container.GitConfig())
	assert.False(t, container.UpdatedAt().IsZero())
}

func TestContainer_UpdateResourceLimits(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)

	newCPU := uint32(2000)
	newMemory := uint32(1024)
	newLimits, _ := value.NewResourceLimits(&newCPU, &newMemory)

	err := container.UpdateResourceLimits(newLimits)
	require.NoError(t, err)
	assert.Equal(t, newLimits, container.ResourceLimits())
	assert.False(t, container.UpdatedAt().IsZero())
}

func TestContainer_UpdateTemplateConfig(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)

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
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)

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
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
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
		nil, // githubInstallationID
		gitConfig,
		nil,  // gitCommitHash
		nil,  // lastBuiltGitCommitHash
		true, // needsBuild
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

func TestReconstructContainer_WithGitHubInstallationID(t *testing.T) {
	containerID := uint(100)
	projectID := uint(1)
	name := "Backend API"
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()
	githubInstallationID := int64(12345)

	container := ReconstructContainer(
		containerID,
		projectID,
		nil, // templateID
		name,
		slug,
		nil, // stableWindow
		nil, // templateConfig
		&githubInstallationID,
		gitConfig,
		nil,  // gitCommitHash
		nil,  // lastBuiltGitCommitHash
		true, // needsBuild
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
	assert.NotNil(t, container.GitHubInstallationID())
	assert.Equal(t, githubInstallationID, *container.GitHubInstallationID())
}

func TestNewContainer_WithGitHubInstallationID(t *testing.T) {
	projectID := uint(1)
	name := "Backend API"
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()
	githubInstallationID := int64(12345)

	container, err := NewContainer(
		projectID,
		name,
		slug,
		gitConfig,
		resourceLimits,
		nil, // templateID
		nil, // templateConfig
		&githubInstallationID,
	)

	assert.NoError(t, err)
	assert.NotNil(t, container)
	assert.Equal(t, projectID, container.ProjectID())
	assert.Equal(t, name, container.Name())
	assert.NotNil(t, container.GitHubInstallationID())
	assert.Equal(t, githubInstallationID, *container.GitHubInstallationID())
}

func TestContainer_SetGitHubInstallationID(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")

	// Create a container without GitHub installation ID
	container, err := NewContainer(
		1,
		"Backend API",
		slug,
		defaultGitConfig(),
		defaultResourceLimits(),
		nil,
		nil,
		nil, // No installation ID initially
	)
	assert.NoError(t, err)
	assert.Nil(t, container.GitHubInstallationID())

	// Set GitHub installation ID
	installationID := int64(12345)
	container.SetGitHubInstallationID(&installationID)

	// Verify it was set
	assert.NotNil(t, container.GitHubInstallationID())
	assert.Equal(t, installationID, *container.GitHubInstallationID())

	// Verify updatedAt was updated
	assert.False(t, container.UpdatedAt().IsZero())

	// Update to a different installation ID
	newInstallationID := int64(67890)
	container.SetGitHubInstallationID(&newInstallationID)

	assert.NotNil(t, container.GitHubInstallationID())
	assert.Equal(t, newInstallationID, *container.GitHubInstallationID())

	// Set to nil (remove installation ID)
	container.SetGitHubInstallationID(nil)
	assert.Nil(t, container.GitHubInstallationID())
}

// ============================================================================
// BuildVar Tests
// ============================================================================

func TestContainer_AddBuildVar(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
	container.SetContainerID(1)

	t.Run("Success", func(t *testing.T) {
		_, err := container.AddBuildVar("NODE_ENV", "production")
		require.NoError(t, err)
		assert.Len(t, container.BuildVars(), 1)
		assert.True(t, container.HasBuildVar("NODE_ENV"))
	})

	t.Run("Duplicate key", func(t *testing.T) {
		_, err := container.AddBuildVar("NODE_ENV", "development")
		assert.ErrorIs(t, err, containererrors.ErrDuplicateBuildVarKey)
	})

	t.Run("Max build vars exceeded", func(t *testing.T) {
		// Add 99 more build vars (already have 1)
		for i := 1; i < MaxBuildVarsPerContainer; i++ {
			key := "VAR_" + string(rune('A'+i%26)) + string(rune('0'+i/26))
			_, _ = container.AddBuildVar(key, "value")
		}
		assert.Len(t, container.BuildVars(), MaxBuildVarsPerContainer)

		// Try to add one more
		_, err := container.AddBuildVar("EXTRA_VAR", "value")
		assert.ErrorIs(t, err, containererrors.ErrMaxBuildVarsExceeded)
	})
}

func TestContainer_UpdateBuildVar(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
	container.SetContainerID(1)
	_, _ = container.AddBuildVar("NODE_ENV", "production")

	t.Run("Success", func(t *testing.T) {
		err := container.UpdateBuildVar("NODE_ENV", "development")
		require.NoError(t, err)

		buildVar, _ := container.GetBuildVar("NODE_ENV")
		assert.Equal(t, "development", buildVar.Value())
	})

	t.Run("Build var not found", func(t *testing.T) {
		err := container.UpdateBuildVar("NONEXISTENT", "value")
		assert.ErrorIs(t, err, containererrors.ErrBuildVarNotFound)
	})
}

func TestContainer_DeleteBuildVar(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
	container.SetContainerID(1)
	_, _ = container.AddBuildVar("NODE_ENV", "production")
	_, _ = container.AddBuildVar("DEBUG", "true")

	t.Run("Success", func(t *testing.T) {
		err := container.DeleteBuildVar("NODE_ENV")
		require.NoError(t, err)
		assert.Len(t, container.BuildVars(), 1)
		assert.False(t, container.HasBuildVar("NODE_ENV"))
		assert.True(t, container.HasBuildVar("DEBUG"))
	})

	t.Run("Build var not found", func(t *testing.T) {
		err := container.DeleteBuildVar("NONEXISTENT")
		assert.ErrorIs(t, err, containererrors.ErrBuildVarNotFound)
	})
}

// ============================================================================
// Cross-Type Key Validation Tests
// ============================================================================

func TestContainer_CrossTypeKeyValidation_EnvVarConflict(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
	container.SetContainerID(1)

	// Add an environment variable
	_, err := container.AddEnvVar("API_KEY", "env_value")
	require.NoError(t, err)

	// Try to add a secret with the same key
	t.Run("Cannot add secret with existing env var key", func(t *testing.T) {
		_, err := container.AddSecret("API_KEY", "secret_value")
		assert.ErrorIs(t, err, containererrors.ErrDuplicateKeyAcrossTypes)
	})

	// Try to add a build var with the same key
	t.Run("Cannot add build var with existing env var key", func(t *testing.T) {
		_, err := container.AddBuildVar("API_KEY", "build_value")
		assert.ErrorIs(t, err, containererrors.ErrDuplicateKeyAcrossTypes)
	})
}

func TestContainer_CrossTypeKeyValidation_SecretConflict(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
	container.SetContainerID(1)

	// Add a secret
	_, err := container.AddSecret("DATABASE_PASSWORD", "secret_value")
	require.NoError(t, err)

	// Try to add an env var with the same key
	t.Run("Cannot add env var with existing secret key", func(t *testing.T) {
		_, err := container.AddEnvVar("DATABASE_PASSWORD", "env_value")
		assert.ErrorIs(t, err, containererrors.ErrDuplicateKeyAcrossTypes)
	})

	// Try to add a build var with the same key
	t.Run("Cannot add build var with existing secret key", func(t *testing.T) {
		_, err := container.AddBuildVar("DATABASE_PASSWORD", "build_value")
		assert.ErrorIs(t, err, containererrors.ErrDuplicateKeyAcrossTypes)
	})
}

func TestContainer_CrossTypeKeyValidation_BuildVarConflict(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
	container.SetContainerID(1)

	// Add a build variable
	_, err := container.AddBuildVar("BUILD_VERSION", "build_value")
	require.NoError(t, err)

	// Try to add an env var with the same key
	t.Run("Cannot add env var with existing build var key", func(t *testing.T) {
		_, err := container.AddEnvVar("BUILD_VERSION", "env_value")
		assert.ErrorIs(t, err, containererrors.ErrDuplicateKeyAcrossTypes)
	})

	// Try to add a secret with the same key
	t.Run("Cannot add secret with existing build var key", func(t *testing.T) {
		_, err := container.AddSecret("BUILD_VERSION", "secret_value")
		assert.ErrorIs(t, err, containererrors.ErrDuplicateKeyAcrossTypes)
	})
}

func TestContainer_CrossTypeKeyValidation_AllThreeTypes(t *testing.T) {
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	container, _ := NewContainer(1, "Backend API", slug, defaultGitConfig(), defaultResourceLimits(), nil, nil, nil)
	container.SetContainerID(1)

	// Add three different keys, one for each type
	_, err := container.AddEnvVar("ENV_VAR", "env_value")
	require.NoError(t, err)

	_, err = container.AddSecret("SECRET_VAR", "secret_value")
	require.NoError(t, err)

	_, err = container.AddBuildVar("BUILD_VAR", "build_value")
	require.NoError(t, err)

	// Verify all three exist independently
	assert.True(t, container.HasEnvVar("ENV_VAR"))
	assert.True(t, container.HasSecret("SECRET_VAR"))
	assert.True(t, container.HasBuildVar("BUILD_VAR"))

	// Verify cross-type conflicts are enforced
	_, err = container.AddEnvVar("SECRET_VAR", "value")
	assert.ErrorIs(t, err, containererrors.ErrDuplicateKeyAcrossTypes)

	_, err = container.AddSecret("BUILD_VAR", "value")
	assert.ErrorIs(t, err, containererrors.ErrDuplicateKeyAcrossTypes)

	_, err = container.AddBuildVar("ENV_VAR", "value")
	assert.ErrorIs(t, err, containererrors.ErrDuplicateKeyAcrossTypes)
}

// TestContainer_UpdateNetwork_Success tests successful network update
func TestContainer_UpdateNetwork_Success(t *testing.T) {
	// Setup container with a network
	projectID := uint(1)
	name := "Test Container"
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, nil, nil, nil)
	require.NoError(t, err)
	container.SetContainerID(1) // Required for network creation

	// Add initial network
	port := uint16(8080)
	netType, _ := value.NewNetworkType("http")
	fqdn := "original.launchpad.kr"
	network, err := container.AddNetwork(&port, nil, netType, nil, &fqdn)
	require.NoError(t, err)
	networkID := network.NetworkID()

	// Update network
	newPort := uint16(9090)
	newNetType, _ := value.NewNetworkType("tcp")
	newFQDN := "updated.launchpad.kr"

	updatedNetwork, err := container.UpdateNetwork(networkID, &newPort, newNetType, &newFQDN)

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, updatedNetwork)
	assert.Equal(t, networkID, updatedNetwork.NetworkID())
	assert.Equal(t, newPort, *updatedNetwork.InternalPort())
	assert.Equal(t, "tcp", updatedNetwork.NetworkType().String())
	assert.Equal(t, newFQDN, *updatedNetwork.FQDN())
}

// TestContainer_UpdateNetwork_PartialUpdate tests updating only some fields
func TestContainer_UpdateNetwork_PartialUpdate(t *testing.T) {
	// Setup
	projectID := uint(1)
	name := "Test Container"
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, nil, nil, nil)
	container.SetContainerID(1)
	require.NoError(t, err)

	// Add network
	port := uint16(8080)
	netType, _ := value.NewNetworkType("http")
	fqdn := "original.launchpad.kr"
	network, err := container.AddNetwork(&port, nil, netType, nil, &fqdn)
	require.NoError(t, err)
	networkID := network.NetworkID()

	// Update only FQDN
	newFQDN := "updated.launchpad.kr"
	updatedNetwork, err := container.UpdateNetwork(networkID, nil, "", &newFQDN)

	// Verify - port and type unchanged
	assert.NoError(t, err)
	assert.Equal(t, port, *updatedNetwork.InternalPort())
	assert.Equal(t, "http", updatedNetwork.NetworkType().String())
	assert.Equal(t, newFQDN, *updatedNetwork.FQDN())
}

// TestContainer_UpdateNetwork_NotFound tests updating non-existent network
func TestContainer_UpdateNetwork_NotFound(t *testing.T) {
	// Setup
	projectID := uint(1)
	name := "Test Container"
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, nil, nil, nil)
	container.SetContainerID(1)
	require.NoError(t, err)

	// Try to update non-existent network
	nonExistentID := uint(99999)
	newPort := uint16(9090)

	_, err = container.UpdateNetwork(nonExistentID, &newPort, "", nil)

	assert.ErrorIs(t, err, containererrors.ErrNetworkNotFound)
}

// TestContainer_UpdateNetwork_InvalidPort tests updating with invalid port
func TestContainer_UpdateNetwork_InvalidPort(t *testing.T) {
	// Setup
	projectID := uint(1)
	name := "Test Container"
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, nil, nil, nil)
	container.SetContainerID(1)
	require.NoError(t, err)

	// Add network
	port := uint16(8080)
	netType, _ := value.NewNetworkType("http")
	network, err := container.AddNetwork(&port, nil, netType, nil, nil)
	require.NoError(t, err)
	networkID := network.NetworkID()

	// Try to update with invalid port
	invalidPort := uint16(0)
	_, err = container.UpdateNetwork(networkID, &invalidPort, "", nil)

	assert.ErrorIs(t, err, containererrors.ErrInvalidPort)
}

// TestContainer_UpdateNetwork_InvalidFQDN tests updating with invalid FQDN
func TestContainer_UpdateNetwork_InvalidFQDN(t *testing.T) {
	// Setup
	projectID := uint(1)
	name := "Test Container"
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, nil, nil, nil)
	container.SetContainerID(1)
	require.NoError(t, err)

	// Add network
	port := uint16(8080)
	netType, _ := value.NewNetworkType("http")
	network, err := container.AddNetwork(&port, nil, netType, nil, nil)
	require.NoError(t, err)
	networkID := network.NetworkID()

	// Try to update with reserved subdomain
	reservedFQDN := "api.launchpad.kr"
	_, err = container.UpdateNetwork(networkID, nil, "", &reservedFQDN)

	assert.ErrorIs(t, err, containererrors.ErrReservedFQDN)
}

// TestContainer_UpdateNetwork_DeletedContainer tests updating in deleted container
func TestContainer_UpdateNetwork_DeletedContainer(t *testing.T) {
	// Setup
	projectID := uint(1)
	name := "Test Container"
	slug, _ := value.NewContainerSlug("c2025011812000055555555")
	gitConfig := defaultGitConfig()
	resourceLimits := defaultResourceLimits()

	container, err := NewContainer(projectID, name, slug, gitConfig, resourceLimits, nil, nil, nil)
	container.SetContainerID(1)
	require.NoError(t, err)

	// Add network
	port := uint16(8080)
	netType, _ := value.NewNetworkType("http")
	network, err := container.AddNetwork(&port, nil, netType, nil, nil)
	require.NoError(t, err)
	networkID := network.NetworkID()

	// Soft delete container
	err = container.SoftDelete()
	require.NoError(t, err)

	// Try to update network
	newPort := uint16(9090)
	_, err = container.UpdateNetwork(networkID, &newPort, "", nil)

	assert.ErrorIs(t, err, containererrors.ErrContainerNotActive)
}
