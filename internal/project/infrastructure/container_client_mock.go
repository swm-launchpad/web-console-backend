package infrastructure

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// MockContainerClient is a mock implementation of ContainerClient interface.
// It returns predefined data based on project ID for testing purposes.
type MockContainerClient struct{}

// NewMockContainerClient creates a new MockContainerClient instance.
func NewMockContainerClient() infrastructure.ContainerClient {
	return &MockContainerClient{}
}

// GetAllContainerInfo returns mock container deployment information based on project ID.
// Different project IDs return different scenarios:
//   - projectID 1: Single container (spring-helloworld)
//   - projectID 2: Multi-container with MySQL (spring-helloworld + mysql)
//   - Other IDs: Returns error (project not found)
func (m *MockContainerClient) GetAllContainerInfo(ctx context.Context, projectID uint) (*dto.ContainerDeploymentInfo, error) {
	switch projectID {
	case 1:
		return m.getSingleContainerDeployment(), nil
	case 2:
		return m.getMultiContainerDeployment(), nil
	default:
		return nil, &MockProjectNotFoundError{ProjectID: projectID}
	}
}

// getSingleContainerDeployment returns a single container deployment scenario.
// Based on Tekton README example: spring-helloworld single container.
func (m *MockContainerClient) getSingleContainerDeployment() *dto.ContainerDeploymentInfo {
	domain := "spring-helloworld.launchpad.kr"
	healthEndpoint := "/"
	stableWindow := 300

	return &dto.ContainerDeploymentInfo{
		ProjectID:    "b7c376f2-4cb4-4056-89be-33f562c09a63",
		ServiceName:  "spring-helloworld",
		Namespace:    "application",
		StableWindow: &stableWindow,
		ConfigMaps:   []dto.ConfigMapInfo{},
		Volumes:      []dto.VolumeInfo{},
		Containers: []dto.ContainerInfo{
			{
				Name:            "backend",
				Domain:          &domain,
				HealthCheckType: "http",
				HealthEndpoint:  &healthEndpoint,
				Port:            8080,
				HealthPort:      nil,
				ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/spring-helloworld",
				ImageTag:        "e5c373e",
				EnvVars: map[string]string{
					"SPRING_PROFILES_ACTIVE": "production",
				},
				Secrets:       map[string]string{},
				CPULimit:      "1000m",
				MemoryRequest: "512Mi",
				MemoryLimit:   "1Gi",
				VolumeMounts:  []dto.VolumeMount{},
			},
		},
	}
}

// getMultiContainerDeployment returns a multi-container deployment scenario.
// Based on Tekton README example: spring-helloworld + mysql stack.
func (m *MockContainerClient) getMultiContainerDeployment() *dto.ContainerDeploymentInfo {
	domain := "spring-helloworld-stack.launchpad.kr"
	healthEndpoint := "/"
	pvcType := "pvc"
	configMapType := "config_map"
	mysqlInitdbConfigMapName := "mysql-initdb-config"
	mysqlDataCapacity := "1Gi"

	return &dto.ContainerDeploymentInfo{
		ProjectID:   "bdeb92-ebwe9fjf32ir239s",
		ServiceName: "spring-helloworld-stack",
		Namespace:   "application",
		ConfigMaps: []dto.ConfigMapInfo{
			{
				Name: mysqlInitdbConfigMapName,
				Data: map[string]string{
					"init.sql": "CREATE DATABASE IF NOT EXISTS mydb;\nUSE mydb;\nCREATE TABLE users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(50));",
				},
			},
		},
		Volumes: []dto.VolumeInfo{
			{
				Name:          "mysql-initdb",
				Type:          &configMapType,
				ConfigMapName: &mysqlInitdbConfigMapName,
			},
			{
				Name:     "mysql-data",
				Type:     &pvcType,
				Capacity: &mysqlDataCapacity,
			},
		},
		Containers: []dto.ContainerInfo{
			{
				Name:            "backend",
				Domain:          &domain,
				HealthCheckType: "http",
				HealthEndpoint:  &healthEndpoint,
				Port:            8080,
				HealthPort:      nil,
				ImageName:       "957833999474.dkr.ecr.ap-northeast-2.amazonaws.com/spring-helloworld",
				ImageTag:        "e5c373e",
				EnvVars:         map[string]string{},
				Secrets:         map[string]string{},
				CPULimit:        "1000m",
				MemoryRequest:   "512Mi",
				MemoryLimit:     "1Gi",
				VolumeMounts:    []dto.VolumeMount{},
			},
			{
				Name:            "mysql",
				Domain:          nil, // Internal-only container
				HealthCheckType: "tcp",
				HealthEndpoint:  nil,
				Port:            3306,
				HealthPort:      nil,
				ImageName:       "mysql",
				ImageTag:        "8.0",
				EnvVars: map[string]string{
					"MYSQL_DATABASE":      "mydb",
					"MYSQL_ROOT_PASSWORD": "rootpass",
				},
				Secrets:       map[string]string{},
				CPULimit:      "500m",
				MemoryRequest: "512Mi",
				MemoryLimit:   "1Gi",
				VolumeMounts: []dto.VolumeMount{
					{
						VolumeName: "mysql-initdb",
						MountPaths: []string{"/docker-entrypoint-initdb.d"},
					},
					{
						VolumeName: "mysql-data",
						MountPaths: []string{"/var/lib/mysql"},
					},
				},
			},
		},
	}
}

// MockProjectNotFoundError represents an error when a project is not found in the mock.
type MockProjectNotFoundError struct {
	ProjectID uint
}

// Error implements the error interface.
func (e *MockProjectNotFoundError) Error() string {
	return "mock: project not found"
}
