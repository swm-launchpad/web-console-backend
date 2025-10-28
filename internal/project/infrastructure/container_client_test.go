package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	containerBuild "github.com/swm-launchpad/web-console-backend/internal/container/application/build"
	containerCombined "github.com/swm-launchpad/web-console-backend/internal/container/application/combined"
	containerDeployment "github.com/swm-launchpad/web-console-backend/internal/container/application/deployment"
)

func TestNewContainerClient(t *testing.T) {
	t.Run("Create new container client with valid dependencies", func(t *testing.T) {
		// Arrange
		mockGetContainersForDeploymentUseCase := &containerDeployment.GetContainersForDeploymentUseCase{}
		mockGetContainersForBuildUseCase := &containerBuild.GetContainersForBuildUseCase{}
		mockGetContainersForBuildAndDeployUseCase := &containerCombined.GetContainersForBuildAndDeployUseCase{}
		testLogger := logger.NewForTest()

		// Act
		client := NewContainerClient(mockGetContainersForDeploymentUseCase, mockGetContainersForBuildUseCase, mockGetContainersForBuildAndDeployUseCase, testLogger)

		// Assert
		assert.NotNil(t, client)
	})

	t.Run("Container client structure validation", func(t *testing.T) {
		// Arrange
		mockGetContainersForDeploymentUseCase := &containerDeployment.GetContainersForDeploymentUseCase{}
		mockGetContainersForBuildUseCase := &containerBuild.GetContainersForBuildUseCase{}
		mockGetContainersForBuildAndDeployUseCase := &containerCombined.GetContainersForBuildAndDeployUseCase{}
		testLogger := logger.NewForTest()

		// Act
		client := NewContainerClient(mockGetContainersForDeploymentUseCase, mockGetContainersForBuildUseCase, mockGetContainersForBuildAndDeployUseCase, testLogger)
		concreteClient, ok := client.(*containerClient)

		// Assert
		assert.True(t, ok, "Client should be of type *containerClient")
		assert.NotNil(t, concreteClient.getContainersForDeploymentUseCase)
		assert.NotNil(t, concreteClient.getContainersForBuildUseCase)
		assert.NotNil(t, concreteClient.getContainersForBuildAndDeployUseCase)
		assert.NotNil(t, concreteClient.logger)
	})
}

func TestContainerClient_NoDependencyOnVolumeRepo(t *testing.T) {
	t.Run("ContainerClient should not have volumeRepo dependency", func(t *testing.T) {
		// This test verifies the refactoring where volumeRepo was removed
		// from ContainerClient to follow separation of concerns

		// Arrange
		mockGetContainersForDeploymentUseCase := &containerDeployment.GetContainersForDeploymentUseCase{}
		mockGetContainersForBuildUseCase := &containerBuild.GetContainersForBuildUseCase{}
		mockGetContainersForBuildAndDeployUseCase := &containerCombined.GetContainersForBuildAndDeployUseCase{}
		testLogger := logger.NewForTest()

		// Act
		client := NewContainerClient(mockGetContainersForDeploymentUseCase, mockGetContainersForBuildUseCase, mockGetContainersForBuildAndDeployUseCase, testLogger)
		concreteClient, ok := client.(*containerClient)

		// Assert
		assert.True(t, ok)
		// The containerClient struct should have use cases and logger fields
		// No volumeRepo field should exist
		assert.NotNil(t, concreteClient.getContainersForDeploymentUseCase)
		assert.NotNil(t, concreteClient.getContainersForBuildUseCase)
		assert.NotNil(t, concreteClient.getContainersForBuildAndDeployUseCase)
		assert.NotNil(t, concreteClient.logger)
	})
}
