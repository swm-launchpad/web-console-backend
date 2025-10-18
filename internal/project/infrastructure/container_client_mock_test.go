package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestMockContainerClient_GetContainerConfig(t *testing.T) {
	client := NewMockContainerClient()
	ctx := context.Background()

	t.Run("Project ID 1: Single container configuration", func(t *testing.T) {
		config, err := client.GetContainerConfig(ctx, 1)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify containers
		assert.Len(t, config.Containers, 1)
		backend := config.Containers[0]
		assert.Equal(t, "backend", backend.Name)
		assert.NotNil(t, backend.Domain)
		assert.Equal(t, "spring-helloworld.launchpad.kr", *backend.Domain)
		assert.Equal(t, "http", backend.HealthCheckType)
		assert.NotNil(t, backend.HealthEndpoint)
		assert.Equal(t, "/", *backend.HealthEndpoint)
		assert.Equal(t, 8080, backend.Port)
		assert.Nil(t, backend.HealthPort)
		assert.Equal(t, "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/spring-helloworld", backend.ImageName)
		assert.Equal(t, "e5c373e", backend.ImageTag)

		// Verify environment variables
		assert.Len(t, backend.EnvVars, 1)
		assert.Equal(t, "production", backend.EnvVars["SPRING_PROFILES_ACTIVE"])

		// Verify resource limits
		assert.Equal(t, "1000m", backend.CPULimit)
		assert.Equal(t, "512Mi", backend.MemoryRequest)
		assert.Equal(t, "1Gi", backend.MemoryLimit)

		// Verify no volume mounts
		assert.Empty(t, backend.VolumeMounts)
	})

	t.Run("Project ID 2: Multi-container configuration with MySQL", func(t *testing.T) {
		config, err := client.GetContainerConfig(ctx, 2)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify containers
		assert.Len(t, config.Containers, 2)

		// Backend container
		backend := config.Containers[0]
		assert.Equal(t, "backend", backend.Name)
		assert.NotNil(t, backend.Domain)
		assert.Equal(t, "spring-helloworld-stack.launchpad.kr", *backend.Domain)
		assert.Equal(t, "http", backend.HealthCheckType)
		assert.NotNil(t, backend.HealthEndpoint)
		assert.Equal(t, "/", *backend.HealthEndpoint)
		assert.Equal(t, 8080, backend.Port)
		assert.Empty(t, backend.VolumeMounts)

		// MySQL container
		mysql := config.Containers[1]
		assert.Equal(t, "mysql", mysql.Name)
		assert.Nil(t, mysql.Domain) // Internal-only
		assert.Equal(t, "tcp", mysql.HealthCheckType)
		assert.Nil(t, mysql.HealthEndpoint) // TCP doesn't need endpoint
		assert.Equal(t, 3306, mysql.Port)
		assert.Equal(t, "mysql", mysql.ImageName)
		assert.Equal(t, "8.0", mysql.ImageTag)

		// Verify MySQL environment variables
		assert.Len(t, mysql.EnvVars, 2)
		assert.Equal(t, "mydb", mysql.EnvVars["MYSQL_DATABASE"])
		assert.Equal(t, "rootpass", mysql.EnvVars["MYSQL_ROOT_PASSWORD"])

		// Verify MySQL resource limits
		assert.Equal(t, "500m", mysql.CPULimit)
		assert.Equal(t, "512Mi", mysql.MemoryRequest)
		assert.Equal(t, "1Gi", mysql.MemoryLimit)

		// Verify MySQL volume mounts (only PVC volume, no ConfigMap)
		assert.Len(t, mysql.VolumeMounts, 1)

		// PVC mount - references volume ID
		dataMount := mysql.VolumeMounts[0]
		assert.Equal(t, uint(1), dataMount.VolumeID)
		assert.Equal(t, "/var/lib/mysql", dataMount.MountPath)
	})

	t.Run("Project ID not found", func(t *testing.T) {
		config, err := client.GetContainerConfig(ctx, 999)
		assert.Error(t, err)
		assert.Nil(t, config)

		// Verify it returns domain error
		assert.True(t, errors.Is(err, projecterrors.ErrProjectNotFound))
	})
}

func TestNewMockContainerClient(t *testing.T) {
	t.Run("Create new mock client", func(t *testing.T) {
		client := NewMockContainerClient()
		require.NotNil(t, client)

		// Verify it can be used
		config, err := client.GetContainerConfig(context.Background(), 1)
		require.NoError(t, err)
		require.NotNil(t, config)
	})
}
