package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	volumemodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
)

// Helper function to create test project for deploy service tests
func createTestProjectForDeploy(projectID uint, operationStatus value.ProjectOperationStatus) *projectmodel.Project {
	slug, _ := value.NewProjectSlug("test-project")
	limits, _ := value.NewResourceLimits(1000, 2048, 2048, 1000000)
	now := time.Now()

	return projectmodel.ReconstructProject(
		projectID,
		"Test Project",
		*slug,
		value.ProjectStatusActive,
		operationStatus,
		*limits,
		now,
		now,
		false,
		nil,
	)
}

func TestDeployService_buildTektonRequest(t *testing.T) {
	// Arrange
	service := &deployService{
		deployNamespace:    "test-namespace",
		projectServiceName: "test-service",
	}

	proj := createTestProjectForDeploy(1, value.ProjectOperationStatusNothing)

	containerConfig := &dto.ContainerDeploymentConfig{
		Containers: []dto.ContainerInfo{
			{
				Name:      "app",
				ImageName: "nginx",
				ImageTag:  "latest",
			},
		},
		ConfigMaps: []dto.ConfigMapInfo{
			{
				Name: "app-config",
				Data: map[string]string{"key": "value"},
			},
		},
		Volumes: []dto.VolumeInfo{
			{
				Name: "config-vol",
			},
		},
	}

	volume, _ := volumemodel.NewVolume(1, "data-volume", 1024)
	volumes := []*volumemodel.Volume{volume}

	// Act
	request, err := service.buildTektonRequest(proj, containerConfig, volumes)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, request)
	assert.Equal(t, "false", request.DryRun)
	assert.Equal(t, "1", request.DeploymentConfigJSON.ProjectID)
	assert.Equal(t, "test-service", request.DeploymentConfigJSON.ServiceName)
	assert.Equal(t, "test-namespace", request.DeploymentConfigJSON.Namespace)
	assert.Equal(t, 180, request.DeploymentConfigJSON.StableWindow)
	assert.Len(t, request.DeploymentConfigJSON.Containers, 1)
	assert.Equal(t, "app", request.DeploymentConfigJSON.Containers[0].Name)
	assert.Len(t, request.DeploymentConfigJSON.ConfigMaps, 1)
	assert.Len(t, request.DeploymentConfigJSON.Volumes, 2) // 1 from container config + 1 from volumes
}

func TestDeployService_convertVolumesToDTO(t *testing.T) {
	// Arrange
	service := &deployService{}

	volume1, _ := volumemodel.NewVolume(1, "data-vol", 1024)
	volume2, _ := volumemodel.NewVolume(1, "cache-vol", 512)
	volumes := []*volumemodel.Volume{volume1, volume2}

	// Act
	result := service.convertVolumesToDTO(volumes)

	// Assert
	assert.Len(t, result, 2)
	assert.Equal(t, "data-vol", result[0].Name)
	assert.Equal(t, "pvc", *result[0].Type)
	assert.Equal(t, "1024Mi", *result[0].Capacity)
	assert.Equal(t, "cache-vol", result[1].Name)
	assert.Equal(t, "512Mi", *result[1].Capacity)
}

func TestDeployService_updateDeploymentStatus_Running(t *testing.T) {
	// Test the status transition logic for Running status
	// This test focuses on the deployment status changes without external dependencies

	d := deployment.NewDeployment(1)
	d.SetDeploymentID(1)
	_ = d.MarkAsTracking("event-123")

	// Simulate Running status
	err := d.MarkAsRunning("deploy-run-abc")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, deployment.DeploymentStatusRunning, d.Status())
	assert.Equal(t, "deploy-run-abc", d.TektonPipelineRunName())
}

func TestDeployService_updateDeploymentStatus_Success(t *testing.T) {
	// Test the status transition logic for Success status

	d := deployment.NewDeployment(1)
	d.SetDeploymentID(1)
	_ = d.MarkAsTracking("event-123")
	_ = d.MarkAsRunning("deploy-run-abc")

	// Simulate Success status
	err := d.Complete("Deployment succeeded")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, deployment.DeploymentStatusSuccess, d.Status())
	assert.True(t, d.IsCompleted())
}

func TestDeployService_updateDeploymentStatus_Failed(t *testing.T) {
	// Test the status transition logic for Failed status

	d := deployment.NewDeployment(1)
	d.SetDeploymentID(1)
	_ = d.MarkAsTracking("event-123")
	_ = d.MarkAsRunning("deploy-run-abc")

	// Simulate Failed status
	err := d.Fail("Task failed: build-and-deploy")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, deployment.DeploymentStatusFailed, d.Status())
	assert.True(t, d.IsCompleted())
}

func TestDeployService_updateDeploymentStatus_AlreadyCompleted(t *testing.T) {
	// Test that completed status cannot be changed

	d := deployment.NewDeployment(1)
	d.SetDeploymentID(1)
	_ = d.MarkAsTracking("event-123")
	_ = d.MarkAsRunning("deploy-run-abc")
	_ = d.Complete("Already completed")

	// Try to fail after completion - should return error
	err := d.Fail("Should not overwrite")

	// Assert
	assert.Error(t, err)
	// Status should remain success (not changed to failed)
	assert.Equal(t, deployment.DeploymentStatusSuccess, d.Status())
}
