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
	volumevalue "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume/value"
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
		nil, // activeDeploymentID
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
	}

	volume, _ := volumemodel.NewVolume(1, "data-volume", 1024)
	slug, _ := volumevalue.NewVolumeSlug("v2025011812000012345678")
	volume.SetSlug(slug)
	volumes := []*volumemodel.Volume{volume}

	// Act
	request, err := service.buildTektonRequest(proj, containerConfig, volumes)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, request)
	assert.Equal(t, "false", request.DryRun)
	assert.Equal(t, "1", request.DeploymentConfigJSON.ProjectID)
	assert.Equal(t, "test-project", request.DeploymentConfigJSON.ServiceName) // Should use project slug, not hardcoded service name
	assert.Equal(t, "test-namespace", request.DeploymentConfigJSON.Namespace)
	assert.Equal(t, 180, request.DeploymentConfigJSON.StableWindow)
	assert.Len(t, request.DeploymentConfigJSON.Containers, 1)
	assert.Equal(t, "app", request.DeploymentConfigJSON.Containers[0].Name)
	assert.Empty(t, request.DeploymentConfigJSON.ConfigMaps) // ConfigMaps managed at project level (not yet implemented)
	assert.Len(t, request.DeploymentConfigJSON.Volumes, 1)   // Only PVC volumes from VolumeRepository
}

func TestDeployService_convertVolumesToDTO(t *testing.T) {
	// Arrange
	service := &deployService{}

	volume1, _ := volumemodel.NewVolume(1, "data-vol", 1024)
	slug1, _ := volumevalue.NewVolumeSlug("v2025011812000012345678")
	volume1.SetSlug(slug1)

	volume2, _ := volumemodel.NewVolume(1, "cache-vol", 512)
	slug2, _ := volumevalue.NewVolumeSlug("v2025011812000087654321")
	volume2.SetSlug(slug2)

	volumes := []*volumemodel.Volume{volume1, volume2}

	// Act
	result := service.convertVolumesToDTO(volumes)

	// Assert
	assert.Len(t, result, 2)
	// First volume (data-vol)
	assert.Equal(t, "v2025011812000012345678", result[0].Name)
	assert.Equal(t, "1024Mi", *result[0].Capacity)
	// Second volume (cache-vol)
	assert.Equal(t, "v2025011812000087654321", result[1].Name)
	assert.Equal(t, "512Mi", *result[1].Capacity)
}

func TestDeployService_updateDeploymentStatus_Running(t *testing.T) {
	// Test the status transition logic for Running status
	// This test focuses on the deployment status changes without external dependencies

	d := deployment.NewDeployment(1)
	d.SetDeploymentID(1)
	eventID := "event-123"
	_ = d.InitTektonInfo(&eventID, nil)

	// Simulate Running status
	runName := "deploy-run-abc"
	_ = d.InitTektonInfo(nil, &runName)
	err := d.UpdateRunningStatus(nil, nil)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, deployment.DeploymentStatusRunning, d.Status())
	r, exists := d.TektonPipelineRunName()
	assert.True(t, exists)
	assert.Equal(t, "deploy-run-abc", r)
}

func TestDeployService_updateDeploymentStatus_Success(t *testing.T) {
	// Test the status transition logic for Success status

	d := deployment.NewDeployment(1)
	d.SetDeploymentID(1)
	eventID := "event-123"
	_ = d.InitTektonInfo(&eventID, nil)
	runName := "deploy-run-abc"
	_ = d.InitTektonInfo(nil, &runName)
	_ = d.UpdateRunningStatus(nil, nil)

	// Simulate Success status
	msg := "Deployment succeeded"
	err := d.UpdateCompleteStatus(deployment.DeploymentStatusSuccess, &msg, time.Now())

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, deployment.DeploymentStatusSuccess, d.Status())
	assert.True(t, d.IsCompleted())
}

func TestDeployService_updateDeploymentStatus_Failed(t *testing.T) {
	// Test the status transition logic for Failed status

	d := deployment.NewDeployment(1)
	d.SetDeploymentID(1)
	eventID := "event-123"
	_ = d.InitTektonInfo(&eventID, nil)
	runName := "deploy-run-abc"
	_ = d.InitTektonInfo(nil, &runName)
	_ = d.UpdateRunningStatus(nil, nil)

	// Simulate Failed status
	msg := "Task failed: build-and-deploy"
	err := d.UpdateCompleteStatus(deployment.DeploymentStatusFailed, &msg, time.Now())

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, deployment.DeploymentStatusFailed, d.Status())
	assert.True(t, d.IsCompleted())
}

func TestDeployService_updateDeploymentStatus_AlreadyCompleted(t *testing.T) {
	// Test that completed status cannot be changed

	d := deployment.NewDeployment(1)
	d.SetDeploymentID(1)
	eventID := "event-123"
	_ = d.InitTektonInfo(&eventID, nil)
	runName := "deploy-run-abc"
	_ = d.InitTektonInfo(nil, &runName)
	_ = d.UpdateRunningStatus(nil, nil)
	msg := "Already completed"
	_ = d.UpdateCompleteStatus(deployment.DeploymentStatusSuccess, &msg, time.Now())

	// Try to fail after completion - should return error
	failMsg := "Should not overwrite"
	err := d.UpdateCompleteStatus(deployment.DeploymentStatusFailed, &failMsg, time.Now())

	// Assert
	assert.Error(t, err)
	// Status should remain success (not changed to failed)
	assert.Equal(t, deployment.DeploymentStatusSuccess, d.Status())
}
