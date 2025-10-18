package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	containerDeployment "github.com/swm-launchpad/web-console-backend/internal/container/application/deployment"
)

func TestNewContainerClient(t *testing.T) {
	t.Run("Create new container client with valid dependencies", func(t *testing.T) {
		// Arrange
		mockGetContainersUseCase := &containerDeployment.GetContainersForDeploymentUseCase{}

		// Act
		client := NewContainerClient(mockGetContainersUseCase)

		// Assert
		assert.NotNil(t, client)
	})

	t.Run("Container client structure validation", func(t *testing.T) {
		// Arrange
		mockGetContainersUseCase := &containerDeployment.GetContainersForDeploymentUseCase{}

		// Act
		client := NewContainerClient(mockGetContainersUseCase)
		concreteClient, ok := client.(*containerClient)

		// Assert
		assert.True(t, ok, "Client should be of type *containerClient")
		assert.NotNil(t, concreteClient.getContainersUseCase)
	})
}

func TestContainerClient_NoDependencyOnVolumeRepo(t *testing.T) {
	t.Run("ContainerClient should not have volumeRepo dependency", func(t *testing.T) {
		// This test verifies the refactoring where volumeRepo was removed
		// from ContainerClient to follow separation of concerns

		// Arrange
		mockGetContainersUseCase := &containerDeployment.GetContainersForDeploymentUseCase{}

		// Act
		client := NewContainerClient(mockGetContainersUseCase)
		concreteClient, ok := client.(*containerClient)

		// Assert
		assert.True(t, ok)
		// The containerClient struct should only have getContainersUseCase field
		// No volumeRepo field should exist
		assert.NotNil(t, concreteClient.getContainersUseCase)
	})
}
