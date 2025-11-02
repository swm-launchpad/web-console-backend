package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/deployment"
	projectmodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	volumemodel "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume"
	volumevalue "github.com/swm-launchpad/web-console-backend/internal/project/domain/model/volume/value"
)

// Helper function to create test project for deploy service tests
func createTestProjectForDeploy(projectID uint, operationStatus value.ProjectOperationStatus) *projectmodel.Project {
	slug, _ := value.NewProjectSlug("p2025011812000012345678")
	limits, _ := value.NewResourceLimits(1000, 2048, 2048, 1000000)
	now := time.Now()

	return projectmodel.ReconstructProject(
		projectID,
		"Test Project",
		*slug,
		nil, // fqdn
		value.ProjectStatusActive,
		operationStatus,
		nil, // activeDeploymentID
		nil, // plan
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
		deployNamespace:      "deploy-pipeline",
		applicationNamespace: "test-namespace",
		projectServiceName:   "test-service",
	}

	proj := createTestProjectForDeploy(1, value.ProjectOperationStatusNothing)

	containerConfig := &dto.ContainerDeploymentConfig{
		Containers: []dto.ContainerInfo{
			{
				Name:            "app",
				ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/nginx",
				ImageTag:        "latest",
				Port:            80,
				HealthCheckType: "tcp",
				CPULimit:        "1000m",
				MemoryRequest:   "512Mi",
				MemoryLimit:     "1Gi",
			},
		},
	}

	volume, _ := volumemodel.NewVolume(1, "data-volume", 1024)
	volume.SetVolumeID(1)
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
	assert.Equal(t, "p2025011812000012345678", request.DeploymentConfigJSON.ServiceName) // Should use project slug, not hardcoded service name
	assert.Equal(t, "test-namespace", request.DeploymentConfigJSON.Namespace)
	assert.Equal(t, 180, request.DeploymentConfigJSON.StableWindow)
	assert.Len(t, request.DeploymentConfigJSON.Containers, 1)
	assert.Equal(t, "app", request.DeploymentConfigJSON.Containers[0].Name)
	assert.Empty(t, request.DeploymentConfigJSON.ConfigMaps) // ConfigMaps managed at project level (not yet implemented)
	assert.Len(t, request.DeploymentConfigJSON.Volumes, 1)   // Only PVC volumes from VolumeRepository
}

func TestDeployService_buildTektonRequest_WithVolumeMounts(t *testing.T) {
	// Arrange
	service := &deployService{
		deployNamespace:      "deploy-pipeline",
		applicationNamespace: "test-namespace",
		projectServiceName:   "test-service",
	}

	proj := createTestProjectForDeploy(1, value.ProjectOperationStatusNothing)

	containerConfig := &dto.ContainerDeploymentConfig{
		Containers: []dto.ContainerInfo{
			{
				Name:            "mysql",
				ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/mysql",
				ImageTag:        "8.0",
				Port:            3306,
				HealthCheckType: "tcp",
				CPULimit:        "500m",
				MemoryRequest:   "512Mi",
				MemoryLimit:     "1Gi",
				VolumeMounts: []dto.VolumeMount{
					{VolumeID: 1, MountPath: "/var/lib/mysql"},
				},
			},
		},
	}

	volume, _ := volumemodel.NewVolume(1, "mysql-data", 1024)
	volume.SetVolumeID(1)
	slug, _ := volumevalue.NewVolumeSlug("v2025011812000012345678")
	volume.SetSlug(slug)
	volumes := []*volumemodel.Volume{volume}

	// Act
	request, err := service.buildTektonRequest(proj, containerConfig, volumes)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, request)
	assert.Len(t, request.DeploymentConfigJSON.Containers, 1)

	// Verify volume mount was properly converted
	mysql := request.DeploymentConfigJSON.Containers[0]
	assert.Equal(t, "mysql", mysql.Name)
	assert.Len(t, mysql.VolumeMounts, 1)
	assert.Equal(t, "v2025011812000012345678", mysql.VolumeMounts[0].VolumeName)
	assert.Equal(t, []string{"/var/lib/mysql"}, mysql.VolumeMounts[0].MountPaths)

	// Verify volume info
	assert.Len(t, request.DeploymentConfigJSON.Volumes, 1)
	assert.Equal(t, "v2025011812000012345678", request.DeploymentConfigJSON.Volumes[0].Name)
	assert.Equal(t, "1024Mi", *request.DeploymentConfigJSON.Volumes[0].Capacity)
}

func TestDeployService_buildTektonRequest_VolumeNotFound(t *testing.T) {
	// Arrange
	service := &deployService{
		deployNamespace:      "deploy-pipeline",
		applicationNamespace: "test-namespace",
		projectServiceName:   "test-service",
	}

	proj := createTestProjectForDeploy(1, value.ProjectOperationStatusNothing)

	containerConfig := &dto.ContainerDeploymentConfig{
		Containers: []dto.ContainerInfo{
			{
				Name:            "app",
				ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/nginx",
				ImageTag:        "latest",
				Port:            80,
				HealthCheckType: "tcp",
				CPULimit:        "1000m",
				MemoryRequest:   "512Mi",
				MemoryLimit:     "1Gi",
				VolumeMounts: []dto.VolumeMount{
					{VolumeID: 999, MountPath: "/data"}, // Non-existent volume
				},
			},
		},
	}

	volume, _ := volumemodel.NewVolume(1, "existing-vol", 1024)
	volume.SetVolumeID(1)
	slug, _ := volumevalue.NewVolumeSlug("v2025011812000012345678")
	volume.SetSlug(slug)
	volumes := []*volumemodel.Volume{volume}

	// Act
	request, err := service.buildTektonRequest(proj, containerConfig, volumes)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, request)
	assert.Contains(t, err.Error(), "volume referenced in mount not found")
}

func TestDeployService_convertVolumesToDTO(t *testing.T) {
	// Arrange
	service := &deployService{
		registryURL: "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com",
	}

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

func TestDeployService_convertContainersToTektonFormat_Success(t *testing.T) {
	// Arrange
	service := &deployService{
		registryURL: "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com",
	}

	healthEndpoint := "/health"
	healthPort := 8080
	domain := "example.com"

	containers := []dto.ContainerInfo{
		{
			Name:            "app-container",
			Domain:          &domain,
			HealthCheckType: "http",
			HealthEndpoint:  &healthEndpoint,
			Port:            3000,
			HealthPort:      &healthPort,
			ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/myapp",
			ImageTag:        "v1.0.0",
			EnvVars:         map[string]string{"ENV": "prod"},
			Secrets:         map[string]string{"API_KEY": "secret123"},
			CPULimit:        "1000m",
			MemoryRequest:   "512Mi",
			MemoryLimit:     "1Gi",
			VolumeMounts: []dto.VolumeMount{
				{VolumeID: 1, MountPath: "/data"},
				{VolumeID: 2, MountPath: "/cache"},
			},
		},
		{
			Name:            "sidecar",
			ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/nginx",
			ImageTag:        "latest",
			Port:            80,
			HealthCheckType: "tcp",
			CPULimit:        "500m",
			MemoryRequest:   "256Mi",
			MemoryLimit:     "512Mi",
			VolumeMounts:    []dto.VolumeMount{},
		},
	}

	volumeMap := map[uint]string{
		1: "data-vol-slug",
		2: "cache-vol-slug",
	}

	// Act
	result, err := service.convertContainersToTektonFormat(containers, volumeMap)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// First container
	assert.Equal(t, "app-container", result[0].Name)
	assert.Equal(t, &domain, result[0].Domain)
	assert.Equal(t, "http", result[0].HealthCheckType)
	assert.Equal(t, &healthEndpoint, result[0].HealthEndpoint)
	assert.Equal(t, 3000, result[0].Port)
	assert.Equal(t, &healthPort, result[0].HealthPort)
	assert.Equal(t, "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/myapp", result[0].ImageName)
	assert.Equal(t, "v1.0.0", result[0].ImageTag)
	assert.Equal(t, map[string]string{"ENV": "prod"}, result[0].EnvVars)
	assert.Equal(t, map[string]string{"API_KEY": "secret123"}, result[0].Secrets)
	assert.Equal(t, "1000m", result[0].CPULimit)
	assert.Equal(t, "512Mi", result[0].MemoryRequest)
	assert.Equal(t, "1Gi", result[0].MemoryLimit)
	assert.Len(t, result[0].VolumeMounts, 2)
	assert.Equal(t, "data-vol-slug", result[0].VolumeMounts[0].VolumeName)
	assert.Equal(t, []string{"/data"}, result[0].VolumeMounts[0].MountPaths)
	assert.Equal(t, "cache-vol-slug", result[0].VolumeMounts[1].VolumeName)
	assert.Equal(t, []string{"/cache"}, result[0].VolumeMounts[1].MountPaths)

	// Second container (no volume mounts)
	assert.Equal(t, "sidecar", result[1].Name)
	assert.Equal(t, "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/nginx", result[1].ImageName)
	assert.Nil(t, result[1].Domain)
	assert.Empty(t, result[1].VolumeMounts)
}

func TestDeployService_convertContainersToTektonFormat_VolumeNotFound(t *testing.T) {
	// Arrange
	service := &deployService{
		registryURL: "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com",
	}

	containers := []dto.ContainerInfo{
		{
			Name:            "app",
			ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/myapp",
			ImageTag:        "latest",
			Port:            3000,
			HealthCheckType: "http",
			CPULimit:        "1000m",
			MemoryRequest:   "512Mi",
			MemoryLimit:     "1Gi",
			VolumeMounts: []dto.VolumeMount{
				{VolumeID: 999, MountPath: "/data"}, // This volume doesn't exist
			},
		},
	}

	volumeMap := map[uint]string{
		1: "existing-vol-slug",
	}

	// Act
	result, err := service.convertContainersToTektonFormat(containers, volumeMap)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "volume referenced in mount not found")
	assert.Contains(t, err.Error(), "volume_id 999")
}

func TestDeployService_convertContainersToTektonFormat_NoVolumeMounts(t *testing.T) {
	// Arrange
	service := &deployService{
		registryURL: "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com",
	}

	containers := []dto.ContainerInfo{
		{
			Name:            "simple-app",
			ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/nginx",
			ImageTag:        "latest",
			Port:            80,
			HealthCheckType: "tcp",
			CPULimit:        "500m",
			MemoryRequest:   "256Mi",
			MemoryLimit:     "512Mi",
			VolumeMounts:    []dto.VolumeMount{},
		},
	}

	volumeMap := map[uint]string{}

	// Act
	result, err := service.convertContainersToTektonFormat(containers, volumeMap)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "simple-app", result[0].Name)
	assert.Empty(t, result[0].VolumeMounts)
}

func TestDeployService_convertContainersToTektonFormat_EmptyContainerList(t *testing.T) {
	// Arrange
	service := &deployService{
		registryURL: "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com",
	}
	containers := []dto.ContainerInfo{}
	volumeMap := map[uint]string{}

	// Act
	result, err := service.convertContainersToTektonFormat(containers, volumeMap)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestDeployService_GetDeploymentStatus_WithActiveDeployment(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)

	testLogger := logger.NewForTest()
	service := &deployService{
		projectRepo:    mockProjectRepo,
		deploymentRepo: mockDeploymentRepo,
		logger:         testLogger,
	}

	projectID := uint(1)
	activeDeploymentID := uint(100)

	// Create project with active deployment
	slug, _ := value.NewProjectSlug("p20250118120000abc12345")
	limits, _ := value.NewResourceLimits(1000, 2048, 2048, 1000000)
	now := time.Now()
	proj := projectmodel.ReconstructProject(
		projectID,
		"Test Project",
		*slug,
		nil, // fqdn
		value.ProjectStatusActive,
		value.ProjectOperationStatusDeploying,
		&activeDeploymentID, // active deployment ID
		nil,                 // plan
		*limits,
		now,
		now,
		false,
		nil,
	)

	// Create deployment
	d := deployment.NewDeployment(projectID)
	d.SetDeploymentID(activeDeploymentID)
	eventID := "test-event-123"
	_ = d.InitTektonInfo(&eventID, nil)

	// Mock expectations
	mockProjectRepo.On("FindByID", ctx, projectID).Return(proj, nil)
	mockDeploymentRepo.On("FindByID", ctx, activeDeploymentID).Return(d, nil)

	// Act
	result, err := service.GetDeploymentStatus(ctx, projectID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(activeDeploymentID), uint64(result.DeploymentID))
	assert.Equal(t, projectID, result.ProjectID())
	mockProjectRepo.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
}

func TestDeployService_GetDeploymentStatus_NoActiveDeployment(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)

	testLogger := logger.NewForTest()
	service := &deployService{
		projectRepo:    mockProjectRepo,
		deploymentRepo: mockDeploymentRepo,
		logger:         testLogger,
	}

	projectID := uint(1)

	// Create project without active deployment
	slug, _ := value.NewProjectSlug("p20250118120000abc12345")
	limits, _ := value.NewResourceLimits(1000, 2048, 2048, 1000000)
	now := time.Now()
	proj := projectmodel.ReconstructProject(
		projectID,
		"Test Project",
		*slug,
		nil, // fqdn
		value.ProjectStatusActive,
		value.ProjectOperationStatusNothing,
		nil, // no active deployment ID
		nil, // plan
		*limits,
		now,
		now,
		false,
		nil,
	)

	// Create latest deployment
	d := deployment.NewDeployment(projectID)
	d.SetDeploymentID(50)
	eventID := "old-event-123"
	_ = d.InitTektonInfo(&eventID, nil)
	summary := "Deployment completed"
	finishedAt := time.Now()
	_ = d.UpdateCompleteStatus(deployment.DeploymentStatusSuccess, &summary, finishedAt)

	// Mock expectations - should fall back to latest deployment
	mockProjectRepo.On("FindByID", ctx, projectID).Return(proj, nil)
	mockDeploymentRepo.On("FindLatestByProjectID", ctx, projectID).Return(d, nil)

	// Act
	result, err := service.GetDeploymentStatus(ctx, projectID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(50), uint64(result.DeploymentID))
	assert.True(t, result.IsCompleted())
	mockProjectRepo.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
}

func TestDeployService_GetDeploymentStatus_ProjectNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)

	testLogger := logger.NewForTest()
	service := &deployService{
		projectRepo:    mockProjectRepo,
		deploymentRepo: mockDeploymentRepo,
		logger:         testLogger,
	}

	projectID := uint(1)

	// Mock expectations
	mockProjectRepo.On("FindByID", ctx, projectID).Return(nil, projecterrors.ErrProjectNotFound)

	// Act
	result, err := service.GetDeploymentStatus(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, projecterrors.ErrProjectNotFound)
	mockProjectRepo.AssertExpectations(t)
}

func TestDeployService_RefreshActiveDeployment_Success(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockKubeClient := new(infrastructure.MockKubeClient)
	txManager := db.NewStubTxManager()
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:      txManager,
		projectRepo:    mockProjectRepo,
		deploymentRepo: mockDeploymentRepo,
		kubeClient:     mockKubeClient,
		logger:         testLogger,
	}

	projectID := uint(1)
	activeDeploymentID := uint(100)

	// Create project with active deployment
	slug, _ := value.NewProjectSlug("p20250118120000abc12345")
	limits, _ := value.NewResourceLimits(1000, 2048, 2048, 1000000)
	now := time.Now()
	proj := projectmodel.ReconstructProject(
		projectID,
		"Test Project",
		*slug,
		nil, // fqdn
		value.ProjectStatusActive,
		value.ProjectOperationStatusDeploying,
		&activeDeploymentID, // active deployment ID
		nil,                 // plan
		*limits,
		now,
		now,
		false,
		nil,
	)

	// Create deployment with Tekton info
	d := deployment.NewDeployment(projectID)
	d.SetDeploymentID(activeDeploymentID)
	eventID := "test-event-123"
	runName := "test-run-123"
	_ = d.InitTektonInfo(&eventID, &runName)

	// Create Kubernetes status
	startTime := time.Now().Add(-5 * time.Minute)
	finishedTime := time.Now()
	kubeStatus := &dto.PipelineRun{
		Name:           runName,
		Status:         "True",
		Reason:         "Succeeded",
		Message:        "All tasks completed",
		StartTime:      &startTime,
		CompletionTime: &finishedTime,
	}

	// Mock expectations
	mockProjectRepo.On("FindByID", ctx, projectID).Return(proj, nil)
	mockDeploymentRepo.On("FindByID", ctx, activeDeploymentID).Return(d, nil)
	mockKubeClient.On("GetPipelineRunStatus", ctx, runName).Return(kubeStatus, nil)
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).Return(proj, nil)
	mockProjectRepo.On("Save", mock.Anything, mock.AnythingOfType("*model.Project")).Return(nil)
	mockDeploymentRepo.On("Save", mock.Anything, mock.AnythingOfType("*deployment.Deployment")).Return(nil)

	// Act
	result, err := service.RefreshActiveDeployment(ctx, projectID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint64(activeDeploymentID), uint64(result.DeploymentID))
	assert.True(t, result.IsCompleted())
	mockProjectRepo.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
	mockKubeClient.AssertExpectations(t)
}

func TestDeployService_RefreshActiveDeployment_NoActiveDeployment(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)

	testLogger := logger.NewForTest()
	service := &deployService{
		projectRepo:    mockProjectRepo,
		deploymentRepo: mockDeploymentRepo,
		logger:         testLogger,
	}

	projectID := uint(1)

	// Create project without active deployment
	slug, _ := value.NewProjectSlug("p20250118120000abc12345")
	limits, _ := value.NewResourceLimits(1000, 2048, 2048, 1000000)
	now := time.Now()
	proj := projectmodel.ReconstructProject(
		projectID,
		"Test Project",
		*slug,
		nil, // fqdn
		value.ProjectStatusActive,
		value.ProjectOperationStatusNothing,
		nil, // no active deployment ID
		nil, // plan
		*limits,
		now,
		now,
		false,
		nil,
	)

	// Mock expectations
	mockProjectRepo.On("FindByID", ctx, projectID).Return(proj, nil)

	// Act
	result, err := service.RefreshActiveDeployment(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, projecterrors.ErrDeploymentNotFound)
	mockProjectRepo.AssertExpectations(t)
}

func TestDeployService_RefreshActiveDeployment_ProjectNotFound(t *testing.T) {
	// Arrange
	ctx := context.Background()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)

	testLogger := logger.NewForTest()
	service := &deployService{
		projectRepo:    mockProjectRepo,
		deploymentRepo: mockDeploymentRepo,
		logger:         testLogger,
	}

	projectID := uint(1)

	// Mock expectations
	mockProjectRepo.On("FindByID", ctx, projectID).Return(nil, projecterrors.ErrProjectNotFound)

	// Act
	result, err := service.RefreshActiveDeployment(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, projecterrors.ErrProjectNotFound)
	mockProjectRepo.AssertExpectations(t)
}

// TestDeployProject_RejectsPendingImageTag tests that standalone deploy rejects containers with "pending" image tag
func TestDeployProject_RejectsPendingImageTag(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:       mockTxManager,
		projectRepo:     mockProjectRepo,
		deploymentRepo:  mockDeploymentRepo,
		containerClient: mockContainerClient,
		logger:          testLogger,
	}

	// Mock: Project in 'nothing' state
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusNothing)
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).Return(proj, nil)

	// Mock: Deployment creation succeeds
	mockDeploymentRepo.On("Create", mock.Anything, mock.AnythingOfType("*deployment.Deployment")).
		Run(func(args mock.Arguments) {
			d := args.Get(1).(*deployment.Deployment)
			d.SetDeploymentID(1)
		}).
		Return(nil)

	// Mock: handleDeployFailure will need to find and save the deployment
	mockDeploymentRepo.On("FindByID", mock.Anything, uint(1)).
		Return(deployment.NewDeployment(projectID), nil)

	mockDeploymentRepo.On("Save", mock.Anything, mock.AnythingOfType("*deployment.Deployment")).
		Return(nil)

	// Mock: Project save succeeds (will be called twice: initial deploy + handleDeployFailure)
	mockProjectRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	// Mock: GetContainerConfig returns container with "pending" image tag (never built)
	containerConfig := &dto.ContainerDeploymentConfig{
		Containers: []dto.ContainerInfo{
			{
				Name:      "app",
				ImageName: "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/nginx",
				ImageTag:  "pending", // ← Never been built
				Port:      80,
			},
		},
	}
	mockContainerClient.On("GetContainerConfig", mock.Anything, projectID).
		Return(containerConfig, nil)

	// Act
	result, err := service.DeployProject(ctx, projectID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "has never been built")
	mockProjectRepo.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)

	// Note: handleDeployFailure should have been called to rollback project status
	// In production, the deployment would be marked as backend_trigger_failed
	// and project status would be reset to 'nothing'
}

// TestDeployProject_AllowsBuiltContainers tests that standalone deploy allows containers with valid image tags
func TestDeployProject_AllowsBuiltContainers(t *testing.T) {
	// Arrange
	ctx := context.Background()
	projectID := uint(1)

	mockTxManager := db.NewStubTxManager()
	mockProjectRepo := new(repository.MockProjectRepository)
	mockDeploymentRepo := new(repository.MockDeploymentRepository)
	mockVolumeRepo := new(repository.MockVolumeRepository)
	mockContainerClient := new(infrastructure.MockContainerClient)
	mockTektonClient := new(infrastructure.MockTektonClient)
	testLogger := logger.NewForTest()

	service := &deployService{
		txManager:       mockTxManager,
		projectRepo:     mockProjectRepo,
		deploymentRepo:  mockDeploymentRepo,
		volumeRepo:      mockVolumeRepo,
		containerClient: mockContainerClient,
		tektonClient:    mockTektonClient,
		deployNamespace: "test-namespace",
		logger:          testLogger,
	}

	// Mock: Project in 'nothing' state
	proj := createTestProjectForDeploy(projectID, value.ProjectOperationStatusNothing)
	mockProjectRepo.On("FindByIDForUpdate", mock.Anything, projectID).Return(proj, nil)

	// Mock: Deployment creation succeeds
	mockDeploymentRepo.On("Create", mock.Anything, mock.AnythingOfType("*deployment.Deployment")).
		Run(func(args mock.Arguments) {
			d := args.Get(1).(*deployment.Deployment)
			d.SetDeploymentID(1)
		}).
		Return(nil)

	// Mock: Project save succeeds
	mockProjectRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

	// Mock: GetContainerConfig returns container with valid image tag (already built)
	containerConfig := &dto.ContainerDeploymentConfig{
		Containers: []dto.ContainerInfo{
			{
				Name:            "app",
				ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/nginx",
				ImageTag:        "abc1234", // ← Valid commit hash tag
				Port:            80,
				HealthCheckType: "tcp",
				CPULimit:        "1000m",
				MemoryRequest:   "512Mi",
				MemoryLimit:     "1Gi",
			},
		},
	}
	mockContainerClient.On("GetContainerConfig", mock.Anything, projectID).
		Return(containerConfig, nil)

	// Mock: Volumes
	mockVolumeRepo.On("FindByProjectID", mock.Anything, projectID).
		Return([]*volumemodel.Volume{}, nil)

	// Mock: Tekton trigger succeeds
	mockTektonClient.On("TriggerDeploy", mock.Anything, mock.Anything).
		Return(&dto.TektonDeployResponse{EventID: "test-event-123"}, nil)

	// Mock: Deployment save succeeds
	mockDeploymentRepo.On("Save", mock.Anything, mock.AnythingOfType("*deployment.Deployment")).
		Return(nil)

	// Act
	result, err := service.DeployProject(ctx, projectID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.DeploymentID)
	mockProjectRepo.AssertExpectations(t)
	mockDeploymentRepo.AssertExpectations(t)
	mockContainerClient.AssertExpectations(t)
	mockVolumeRepo.AssertExpectations(t)
	mockTektonClient.AssertExpectations(t)

	// Note: Background monitoring goroutine starts but we don't wait for it in this unit test
}
