package infrastructure

import (
	"context"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
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

// GetContainerConfig returns mock container configuration based on project ID.
// Different project IDs return different scenarios:
//   - projectID 1: Single container (spring-helloworld)
//   - projectID 2: Multi-container with MySQL (spring-helloworld + mysql)
//   - Other IDs: Returns projecterrors.ErrProjectNotFound
func (m *MockContainerClient) GetContainerConfig(ctx context.Context, projectID uint) (*dto.ContainerDeploymentConfig, error) {
	switch projectID {
	case 1:
		return m.getSingleContainerConfig(), nil
	case 2:
		return m.getMultiContainerConfig(), nil
	default:
		return nil, projecterrors.ErrProjectNotFound
	}
}

// getSingleContainerConfig returns a single container configuration scenario.
// Based on Tekton README example: spring-helloworld single container.
// Note: Project metadata (project_id, service_name, namespace, stable_window) are NOT included.
// Note: ConfigMaps and Volumes are managed at project level, not by ContainerClient.
func (m *MockContainerClient) getSingleContainerConfig() *dto.ContainerDeploymentConfig {
	domain := "spring-helloworld.launchpad.kr"
	healthEndpoint := "/"

	return &dto.ContainerDeploymentConfig{
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

// getMultiContainerConfig returns a multi-container configuration scenario.
// Based on Tekton README example: spring-helloworld + mysql stack.
// Note: Project metadata (project_id, service_name, namespace, stable_window) are NOT included.
// Note: ConfigMaps and Volumes are managed at project level, not by ContainerClient.
// Note: VolumeMounts specify where to mount volumes (mysql-initdb-config, mysql-data) but
//
//	the actual ConfigMaps and Volumes are created at project level.
func (m *MockContainerClient) getMultiContainerConfig() *dto.ContainerDeploymentConfig {
	domain := "spring-helloworld-stack.launchpad.kr"
	healthEndpoint := "/"

	return &dto.ContainerDeploymentConfig{
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
						VolumeName: "mysql-initdb-config", // Direct reference to ConfigMap name
						MountPaths: []string{"/docker-entrypoint-initdb.d"},
					},
					{
						VolumeName: "mysql-data", // Reference to PVC volume name
						MountPaths: []string{"/var/lib/mysql"},
					},
				},
			},
		},
	}
}

// Compile-time assertion that MockContainerClient implements ContainerClient interface
var _ infrastructure.ContainerClient = (*MockContainerClient)(nil)
