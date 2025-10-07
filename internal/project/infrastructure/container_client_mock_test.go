package infrastructure

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockContainerClient_GetAllContainerInfo(t *testing.T) {
	client := NewMockContainerClient()
	ctx := context.Background()

	t.Run("Project ID 1: Single container deployment", func(t *testing.T) {
		info, err := client.GetAllContainerInfo(ctx, 1)
		require.NoError(t, err)
		require.NotNil(t, info)

		// Verify service info
		assert.Equal(t, "b7c376f2-4cb4-4056-89be-33f562c09a63", info.ProjectID)
		assert.Equal(t, "spring-helloworld", info.ServiceName)
		assert.Equal(t, "application", info.Namespace)
		assert.NotNil(t, info.StableWindow)
		assert.Equal(t, 300, *info.StableWindow)

		// Verify containers
		assert.Len(t, info.Containers, 1)
		backend := info.Containers[0]
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

		// Verify no ConfigMaps
		assert.Empty(t, info.ConfigMaps)

		// Verify no Volumes
		assert.Empty(t, info.Volumes)
	})

	t.Run("Project ID 2: Multi-container deployment with MySQL", func(t *testing.T) {
		info, err := client.GetAllContainerInfo(ctx, 2)
		require.NoError(t, err)
		require.NotNil(t, info)

		// Verify service info
		assert.Equal(t, "bdeb92-ebwe9fjf32ir239s", info.ProjectID)
		assert.Equal(t, "spring-helloworld-stack", info.ServiceName)
		assert.Equal(t, "application", info.Namespace)

		// Verify ConfigMaps
		assert.Len(t, info.ConfigMaps, 1)
		configMap := info.ConfigMaps[0]
		assert.Equal(t, "mysql-initdb-config", configMap.Name)
		assert.Contains(t, configMap.Data, "init.sql")
		assert.Contains(t, configMap.Data["init.sql"], "CREATE DATABASE IF NOT EXISTS mydb")

		// Verify Volumes
		assert.Len(t, info.Volumes, 2)

		// First volume: ConfigMap type
		mysqlInitdb := info.Volumes[0]
		assert.Equal(t, "mysql-initdb", mysqlInitdb.Name)
		assert.NotNil(t, mysqlInitdb.Type)
		assert.Equal(t, "config_map", *mysqlInitdb.Type)
		assert.NotNil(t, mysqlInitdb.ConfigMapName)
		assert.Equal(t, "mysql-initdb-config", *mysqlInitdb.ConfigMapName)

		// Second volume: PVC type
		mysqlData := info.Volumes[1]
		assert.Equal(t, "mysql-data", mysqlData.Name)
		assert.NotNil(t, mysqlData.Type)
		assert.Equal(t, "pvc", *mysqlData.Type)
		assert.NotNil(t, mysqlData.Capacity)
		assert.Equal(t, "1Gi", *mysqlData.Capacity)

		// Verify containers
		assert.Len(t, info.Containers, 2)

		// Backend container
		backend := info.Containers[0]
		assert.Equal(t, "backend", backend.Name)
		assert.NotNil(t, backend.Domain)
		assert.Equal(t, "spring-helloworld-stack.launchpad.kr", *backend.Domain)
		assert.Equal(t, "http", backend.HealthCheckType)
		assert.NotNil(t, backend.HealthEndpoint)
		assert.Equal(t, "/", *backend.HealthEndpoint)
		assert.Equal(t, 8080, backend.Port)
		assert.Empty(t, backend.VolumeMounts)

		// MySQL container
		mysql := info.Containers[1]
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

		// Verify MySQL volume mounts
		assert.Len(t, mysql.VolumeMounts, 2)

		initdbMount := mysql.VolumeMounts[0]
		assert.Equal(t, "mysql-initdb", initdbMount.VolumeName)
		assert.Equal(t, []string{"/docker-entrypoint-initdb.d"}, initdbMount.MountPaths)

		dataMount := mysql.VolumeMounts[1]
		assert.Equal(t, "mysql-data", dataMount.VolumeName)
		assert.Equal(t, []string{"/var/lib/mysql"}, dataMount.MountPaths)
	})

	t.Run("Project ID not found", func(t *testing.T) {
		info, err := client.GetAllContainerInfo(ctx, 999)
		assert.Error(t, err)
		assert.Nil(t, info)

		// Verify error type
		mockErr, ok := err.(*MockProjectNotFoundError)
		assert.True(t, ok)
		assert.Equal(t, uint(999), mockErr.ProjectID)
		assert.Equal(t, "mock: project not found", mockErr.Error())
	})

	t.Run("Verify interface implementation", func(t *testing.T) {
		// This test verifies that MockContainerClient implements ContainerClient interface
		var _ interface{} = client
		// If this compiles, the interface is correctly implemented
	})
}

func TestNewMockContainerClient(t *testing.T) {
	t.Run("Create new mock client", func(t *testing.T) {
		client := NewMockContainerClient()
		require.NotNil(t, client)

		// Verify it can be used
		info, err := client.GetAllContainerInfo(context.Background(), 1)
		require.NoError(t, err)
		require.NotNil(t, info)
	})
}

func TestMockProjectNotFoundError(t *testing.T) {
	t.Run("Error message", func(t *testing.T) {
		err := &MockProjectNotFoundError{ProjectID: 123}
		assert.Equal(t, "mock: project not found", err.Error())
	})

	t.Run("Error contains project ID", func(t *testing.T) {
		err := &MockProjectNotFoundError{ProjectID: 456}
		assert.Equal(t, uint(456), err.ProjectID)
	})
}
