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
// Note: VolumeMounts specify where to mount volumes (e.g., mysql-data PVC) but
//
//	the actual Volumes are created at project level.
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
						VolumeID:  1, // Volume ID (will be mapped to slug by deployment service)
						MountPath: "/var/lib/mysql",
					},
				},
			},
		},
	}
}

// GetContainerBuildConfig returns mock container build configuration based on project ID.
// Different project IDs return different scenarios:
//   - projectID 1: Single container for build
//   - projectID 2: Multi-container for build
//   - Other IDs: Returns projecterrors.ErrProjectNotFound
func (m *MockContainerClient) GetContainerBuildConfig(ctx context.Context, projectID uint) (*dto.ContainerBuildConfig, error) {
	switch projectID {
	case 1:
		return m.getSingleContainerBuildConfig(), nil
	case 2:
		return m.getMultiContainerBuildConfig(), nil
	default:
		return nil, projecterrors.ErrProjectNotFound
	}
}

// getSingleContainerBuildConfig returns a single container build configuration scenario.
func (m *MockContainerClient) getSingleContainerBuildConfig() *dto.ContainerBuildConfig {
	templateBody := "FROM golang:1.21\nCOPY . /app"
	dirPath := "/backend"
	commitHash := "abc1234567890"
	installationID := int64(12345678)

	return &dto.ContainerBuildConfig{
		Containers: []dto.BuildContainerInfo{
			{
				ContainerID:         1,
				Name:                "backend",
				Slug:                "c2025011812000011111111",
				TemplateBody:        &templateBody,
				TemplateConfig:      map[string]interface{}{"go_version": "1.21"},
				GitRepositoryURL:    "https://github.com/test/repo",
				GitBranch:           "main",
				GitDirectoryPath:    &dirPath,
				LastBuiltCommitHash: &commitHash,
				NeedsBuild:          true,
				BuildVars: map[string]string{
					"BUILD_ENV":  "production",
					"GO_VERSION": "1.21",
				},
				InstallationID: &installationID,
			},
		},
	}
}

// getMultiContainerBuildConfig returns a multi-container build configuration scenario.
func (m *MockContainerClient) getMultiContainerBuildConfig() *dto.ContainerBuildConfig {
	templateBody := "FROM golang:1.21\nCOPY . /app"
	mysqlTemplateBody := "FROM mysql:8.0"
	commitHash := "abc1234567890"

	return &dto.ContainerBuildConfig{
		Containers: []dto.BuildContainerInfo{
			{
				ContainerID:         1,
				Name:                "backend",
				Slug:                "c2025011812000011111111",
				TemplateBody:        &templateBody,
				TemplateConfig:      map[string]interface{}{"go_version": "1.21"},
				GitRepositoryURL:    "https://github.com/test/repo",
				GitBranch:           "main",
				GitDirectoryPath:    nil,
				LastBuiltCommitHash: &commitHash,
				NeedsBuild:          true,
				BuildVars:           map[string]string{},
				InstallationID:      nil,
			},
			{
				ContainerID:         2,
				Name:                "mysql",
				Slug:                "c2025011812000022222222",
				TemplateBody:        &mysqlTemplateBody,
				TemplateConfig:      map[string]interface{}{},
				GitRepositoryURL:    "https://github.com/test/repo",
				GitBranch:           "main",
				GitDirectoryPath:    nil,
				LastBuiltCommitHash: nil,
				NeedsBuild:          false,
				BuildVars:           map[string]string{},
				InstallationID:      nil,
			},
		},
	}
}

// GetContainerConfigs returns both build and deployment configurations in a single call.
// Different project IDs return different scenarios:
//   - projectID 1: Single container configs
//   - projectID 2: Multi-container configs
//   - Other IDs: Returns projecterrors.ErrProjectNotFound
func (m *MockContainerClient) GetContainerConfigs(ctx context.Context, projectID uint) (*dto.ContainerBuildConfig, *dto.ContainerDeploymentConfig, error) {
	switch projectID {
	case 1:
		return m.getSingleContainerBuildConfig(), m.getSingleContainerConfig(), nil
	case 2:
		return m.getMultiContainerBuildConfig(), m.getMultiContainerConfig(), nil
	default:
		return nil, nil, projecterrors.ErrProjectNotFound
	}
}

// GetUnifiedContainerConfig returns unified container configuration based on project ID.
// Different project IDs return different scenarios:
//   - projectID 1: Single container unified config
//   - projectID 2: Multi-container unified config
//   - Other IDs: Returns projecterrors.ErrProjectNotFound
func (m *MockContainerClient) GetUnifiedContainerConfig(ctx context.Context, projectID uint) (*dto.UnifiedContainerConfig, error) {
	switch projectID {
	case 1:
		return m.getSingleContainerUnifiedConfig(), nil
	case 2:
		return m.getMultiContainerUnifiedConfig(), nil
	default:
		return nil, projecterrors.ErrProjectNotFound
	}
}

// getSingleContainerUnifiedConfig returns a unified config for single container scenario
func (m *MockContainerClient) getSingleContainerUnifiedConfig() *dto.UnifiedContainerConfig {
	domain := "spring-helloworld.launchpad.kr"
	healthEndpoint := "/"
	commitHash := "abc1234567890"
	dirPath := "/backend"
	installationID := int64(12345678)

	return &dto.UnifiedContainerConfig{
		ProjectID: 1,
		Containers: []dto.UnifiedContainerInfo{
			{
				ContainerID:          1,
				Name:                 "backend",
				Slug:                 "c2025011812000011111111",
				TemplateID:           nil,
				TemplateBody:         nil,
				TemplateConfig:       map[string]interface{}{"go_version": "1.21"},
				GitRepositoryURL:     "https://github.com/test/repo",
				GitBranch:            "main",
				GitDirectoryPath:     &dirPath,
				LastBuiltCommitHash:  &commitHash,
				NeedsBuild:           true,
				BuildVars:            map[string]string{"BUILD_ENV": "production"},
				GitHubInstallationID: &installationID,
				ImageName:            "c2025011812000011111111",
				ImageTag:             "abc1234",
				CPULimit:             nil,
				MemoryLimit:          nil,
				EnvVars:              map[string]string{"SPRING_PROFILES_ACTIVE": "production"},
				Secrets:              map[string]string{},
				Networks:             []dto.NetworkInfo{},
				Mounts:               []dto.MountInfo{},
				Domain:               &domain,
				HealthCheckType:      "http",
				HealthEndpoint:       &healthEndpoint,
				Port:                 8080,
				HealthPort:           nil,
			},
		},
	}
}

// getMultiContainerUnifiedConfig returns a unified config for multi-container scenario
func (m *MockContainerClient) getMultiContainerUnifiedConfig() *dto.UnifiedContainerConfig {
	domain := "spring-helloworld-stack.launchpad.kr"
	healthEndpoint := "/"
	commitHash := "abc1234567890"

	return &dto.UnifiedContainerConfig{
		ProjectID: 2,
		Containers: []dto.UnifiedContainerInfo{
			{
				ContainerID:          1,
				Name:                 "backend",
				Slug:                 "c2025011812000011111111",
				TemplateID:           nil,
				TemplateBody:         nil,
				TemplateConfig:       map[string]interface{}{"go_version": "1.21"},
				GitRepositoryURL:     "https://github.com/test/repo",
				GitBranch:            "main",
				GitDirectoryPath:     nil,
				LastBuiltCommitHash:  &commitHash,
				NeedsBuild:           true,
				BuildVars:            map[string]string{},
				GitHubInstallationID: nil,
				ImageName:            "c2025011812000011111111",
				ImageTag:             "abc1234",
				CPULimit:             nil,
				MemoryLimit:          nil,
				EnvVars:              map[string]string{},
				Secrets:              map[string]string{},
				Networks:             []dto.NetworkInfo{},
				Mounts:               []dto.MountInfo{},
				Domain:               &domain,
				HealthCheckType:      "http",
				HealthEndpoint:       &healthEndpoint,
				Port:                 8080,
				HealthPort:           nil,
			},
			{
				ContainerID:          2,
				Name:                 "mysql",
				Slug:                 "c2025011812000022222222",
				TemplateID:           nil,
				TemplateBody:         nil,
				TemplateConfig:       map[string]interface{}{},
				GitRepositoryURL:     "https://github.com/test/repo",
				GitBranch:            "main",
				GitDirectoryPath:     nil,
				LastBuiltCommitHash:  nil,
				NeedsBuild:           false,
				BuildVars:            map[string]string{},
				GitHubInstallationID: nil,
				ImageName:            "c2025011812000022222222",
				ImageTag:             "pending",
				CPULimit:             nil,
				MemoryLimit:          nil,
				EnvVars:              map[string]string{"MYSQL_DATABASE": "mydb", "MYSQL_ROOT_PASSWORD": "rootpass"},
				Secrets:              map[string]string{},
				Networks:             []dto.NetworkInfo{},
				Mounts:               []dto.MountInfo{{VolumeID: 1, MountPath: "/var/lib/mysql"}},
				Domain:               nil,
				HealthCheckType:      "tcp",
				HealthEndpoint:       nil,
				Port:                 3306,
				HealthPort:           nil,
			},
		},
	}
}

// Compile-time assertion that MockContainerClient implements ContainerClient interface
var _ infrastructure.ContainerClient = (*MockContainerClient)(nil)
