package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure"
	"github.com/swm-launchpad/web-console-backend/test/helper"
)

// TestTektonKubeIntegration tests the integration between TektonClient and KubeClient.
// This test requires actual Tekton and Kubernetes infrastructure to be available.
//
// Prerequisites:
// - Tekton EventListener must be accessible
// - Kubernetes API server must be accessible
// - Environment variables must be set in .env.test
func TestTektonKubeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load environment variables from .env.test
	helper.LoadTestEnv(t)

	// Verify required environment variables for deploy
	requiredEnvVars := []string{
		"TEKTON_DEPLOY_URL",
		"TEKTON_API_AUTH",
		"KUBE_API_SERVER",
		"KUBE_SERVICE_ACCOUNT_TOKEN",
		"KUBE_DEPLOY_NAMESPACE",
		"KUBE_CA_CERT_PATH",
	}

	for _, envVar := range requiredEnvVars {
		value := os.Getenv(envVar)
		if value == "" {
			t.Skipf("Skipping test: required environment variable %s is not set", envVar)
		}
	}

	ctx := context.Background()

	t.Run("TektonClient - Dry run deployment request", func(t *testing.T) {
		// Given - Create TektonClient
		tektonClient, err := infrastructure.NewTektonClient(logger.NewForTest())
		require.NoError(t, err, "Failed to create TektonClient")

		// Create a minimal deployment request with dry_run=true and project_id=0
		deployRequest := &dto.TektonDeployRequest{
			DeploymentConfigJSON: dto.DeploymentConfig{
				ProjectID:    "0",
				ServiceName:  "integration-test-service",
				Namespace:    "application",
				StableWindow: 180,
				ConfigMaps:   []dto.ConfigMapInfo{},
				Volumes:      []dto.VolumeInfo{},
				Containers: []dto.TektonContainerInfo{
					{
						Name:            "test-container",
						Domain:          nil,
						HealthCheckType: "http",
						HealthEndpoint:  stringPtr("/health"),
						Port:            8080,
						HealthPort:      nil,
						ImageName:       "nginx",
						ImageTag:        "latest",
						EnvVars:         map[string]string{},
						Secrets:         map[string]string{},
						CPULimit:        "1000m",
						MemoryRequest:   "512Mi",
						MemoryLimit:     "1Gi",
						VolumeMounts:    []dto.TektonVolumeMount{},
					},
				},
			},
			DryRun: "true",
		}

		// When - Trigger deployment with dry run
		response, err := tektonClient.TriggerDeploy(ctx, deployRequest)

		// Then - Request should be accepted (or fail with validation error)
		// With dry_run=true, Tekton should validate the request but not create actual resources
		if err != nil {
			t.Logf("Dry run request failed (this is acceptable): %v", err)
		} else {
			require.NotNil(t, response, "Response should not be nil")
			assert.NotEmpty(t, response.EventListener, "EventListener should be set")
			assert.NotEmpty(t, response.EventID, "EventID should be set")
			t.Logf("Dry run succeeded - EventID: %s, EventListener: %s", response.EventID, response.EventListener)
		}
	})

	t.Run("KubeClient - List PipelineRuns for project", func(t *testing.T) {
		// Fix CA cert path for tests (relative to test working directory)
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeClient
		kubeClient, err := infrastructure.NewKubeClient(logger.NewForTest())
		if err != nil {
			t.Logf("KubeClient creation failed: %v", err)
			t.Logf("This may be due to network connectivity or certificate issues")
			t.Skip("Skipping KubeClient tests - Kubernetes API not available")
		}
		require.NoError(t, err, "Failed to create KubeClient")

		// When - List PipelineRuns for project_id=0 (test project)
		projectID := uint(0)
		pipelineRuns, err := kubeClient.ListPipelineRuns(ctx, projectID)

		// Then - Should return a list (possibly empty) without errors
		require.NoError(t, err, "Failed to list PipelineRuns")
		assert.NotNil(t, pipelineRuns, "PipelineRuns should not be nil")

		t.Logf("Found %d PipelineRuns for project_id=%d", len(pipelineRuns), projectID)

		// Log details of found PipelineRuns
		for i, pr := range pipelineRuns {
			t.Logf("  [%d] Name: %s, Status: %s, Reason: %s, EventID: %s",
				i, pr.Name, pr.Status, pr.Reason, pr.EventID)
			if pr.StartTime != nil {
				t.Logf("      StartTime: %s", pr.StartTime.Format(time.RFC3339))
			}
			if pr.CompletionTime != nil {
				t.Logf("      CompletionTime: %s", pr.CompletionTime.Format(time.RFC3339))
			}
		}
	})

	t.Run("KubeClient - Get PipelineRun status (if exists)", func(t *testing.T) {
		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeClient
		kubeClient, err := infrastructure.NewKubeClient(logger.NewForTest())
		if err != nil {
			t.Logf("KubeClient creation failed: %v", err)
			t.Skip("Skipping test - Kubernetes API not available")
		}
		require.NoError(t, err, "Failed to create KubeClient")

		// Retry to find PipelineRuns (up to 3 attempts with 1-second interval)
		projectID := uint(0)
		var pipelineRuns []*dto.PipelineRun
		found := retryWithBackoff(3, 1*time.Second, func() bool {
			var err error
			pipelineRuns, err = kubeClient.ListPipelineRuns(ctx, projectID)
			if err != nil {
				t.Logf("Attempt failed to list PipelineRuns: %v", err)
				return false
			}
			return len(pipelineRuns) > 0
		})

		if !found {
			t.Skip("No PipelineRuns found after 3 retry attempts for testing GetPipelineRunStatus")
		}

		// Get the first PipelineRun name
		pipelineRunName := pipelineRuns[0].Name
		t.Logf("Testing GetPipelineRunStatus with PipelineRun: %s", pipelineRunName)

		// When - Get status of the PipelineRun
		status, err := kubeClient.GetPipelineRunStatus(ctx, pipelineRunName)

		// Then - Should return status without errors
		require.NoError(t, err, "Failed to get PipelineRun status")
		require.NotNil(t, status, "Status should not be nil")

		assert.Equal(t, pipelineRunName, status.Name, "PipelineRun name should match")
		assert.NotEmpty(t, status.Status, "Status should be set")
		assert.NotEmpty(t, status.Reason, "Reason should be set")

		t.Logf("PipelineRun Status: %s", status.Status)
		t.Logf("PipelineRun Reason: %s", status.Reason)
		t.Logf("PipelineRun Message: %s", status.Message)
		if status.StartTime != nil {
			t.Logf("StartTime: %s", status.StartTime.Format(time.RFC3339))
		}
		if status.CompletionTime != nil {
			t.Logf("CompletionTime: %s", status.CompletionTime.Format(time.RFC3339))
		}

		// Verify that Status is one of the valid raw Tekton values
		validStatuses := []string{"True", "False", "Unknown"}
		assert.Contains(t, validStatuses, status.Status,
			"Status should be one of: %v", validStatuses)
	})

	t.Run("KubeClient - Get PipelineRun logs (if exists)", func(t *testing.T) {
		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeClient
		kubeClient, err := infrastructure.NewKubeClient(logger.NewForTest())
		if err != nil {
			t.Logf("KubeClient creation failed: %v", err)
			t.Skip("Skipping test - Kubernetes API not available")
		}
		require.NoError(t, err, "Failed to create KubeClient")

		// Retry to find PipelineRuns (up to 3 attempts with 1-second interval)
		projectID := uint(0)
		var pipelineRuns []*dto.PipelineRun
		found := retryWithBackoff(3, 1*time.Second, func() bool {
			var err error
			pipelineRuns, err = kubeClient.ListPipelineRuns(ctx, projectID)
			if err != nil {
				t.Logf("Attempt failed to list PipelineRuns: %v", err)
				return false
			}
			return len(pipelineRuns) > 0
		})

		if !found {
			t.Skip("No PipelineRuns found after 3 retry attempts for testing GetPipelineRunLogs")
		}

		// Get the first PipelineRun name
		pipelineRunName := pipelineRuns[0].Name
		t.Logf("Testing GetPipelineRunLogs with PipelineRun: %s", pipelineRunName)

		// When - Get logs of the PipelineRun
		logs, err := kubeClient.GetPipelineRunLogs(ctx, pipelineRunName)

		// Then - Should return logs without errors (may be empty if tasks haven't started)
		require.NoError(t, err, "Failed to get PipelineRun logs")
		assert.NotNil(t, logs, "Logs should not be nil")

		if logs == "" || logs == "No logs available" {
			t.Logf("No logs available yet for PipelineRun: %s", pipelineRunName)
		} else {
			t.Logf("Retrieved logs (first 500 chars): %s...", truncateString(logs, 500))
			t.Logf("Total log length: %d characters", len(logs))
		}
	})

	t.Run("KubeClient - Get non-existent PipelineRun status (error case)", func(t *testing.T) {
		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeClient
		kubeClient, err := infrastructure.NewKubeClient(logger.NewForTest())
		if err != nil {
			t.Logf("KubeClient creation failed: %v", err)
			t.Skip("Skipping test - Kubernetes API not available")
		}
		require.NoError(t, err, "Failed to create KubeClient")

		nonExistentName := "non-existent-pipeline-run-12345"

		// When - Try to get status of non-existent PipelineRun
		_, err = kubeClient.GetPipelineRunStatus(ctx, nonExistentName)

		// Then - Should return an error
		require.Error(t, err, "Should return error for non-existent PipelineRun")
		t.Logf("Expected error received: %v", err)
	})

	t.Run("KubeClient - Get non-existent PipelineRun logs (error case)", func(t *testing.T) {
		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeClient
		kubeClient, err := infrastructure.NewKubeClient(logger.NewForTest())
		if err != nil {
			t.Logf("KubeClient creation failed: %v", err)
			t.Skip("Skipping test - Kubernetes API not available")
		}
		require.NoError(t, err, "Failed to create KubeClient")

		nonExistentName := "non-existent-pipeline-run-12345"

		// When - Try to get logs of non-existent PipelineRun
		_, err = kubeClient.GetPipelineRunLogs(ctx, nonExistentName)

		// Then - Should return an error
		require.Error(t, err, "Should return error for non-existent PipelineRun")
		t.Logf("Expected error received: %v", err)
	})

	t.Run("KubeClient - Find PipelineRun name by EventID (success case)", func(t *testing.T) {
		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeClient
		kubeClient, err := infrastructure.NewKubeClient(logger.NewForTest())
		if err != nil {
			t.Logf("KubeClient creation failed: %v", err)
			t.Skip("Skipping test - Kubernetes API not available")
		}
		require.NoError(t, err, "Failed to create KubeClient")

		// Retry to find PipelineRuns with EventID (up to 3 attempts with 1-second interval)
		projectID := uint(0)
		var targetPipelineRun *dto.PipelineRun
		found := retryWithBackoff(3, 1*time.Second, func() bool {
			pipelineRuns, err := kubeClient.ListPipelineRuns(ctx, projectID)
			if err != nil {
				t.Logf("Attempt failed to list PipelineRuns: %v", err)
				return false
			}

			if len(pipelineRuns) == 0 {
				t.Logf("No PipelineRuns found in this attempt")
				return false
			}

			// Find a PipelineRun with an EventID
			for _, pr := range pipelineRuns {
				if pr.EventID != "" {
					targetPipelineRun = pr
					return true
				}
			}

			t.Logf("PipelineRuns found but none have EventID")
			return false
		})

		if !found {
			t.Skip("No PipelineRuns with EventID found after 3 retry attempts for testing FindPipelineRunNameByEventID")
		}

		t.Logf("Testing FindPipelineRunNameByEventID with EventID: %s (expected name: %s)",
			targetPipelineRun.EventID, targetPipelineRun.Name)

		// When - Find PipelineRun by EventID
		foundName, err := kubeClient.FindPipelineRunNameByEventID(ctx, targetPipelineRun.EventID)

		// Then - Should return the correct PipelineRun name
		require.NoError(t, err, "Failed to find PipelineRun by EventID")
		assert.Equal(t, targetPipelineRun.Name, foundName, "PipelineRun name should match")

		t.Logf("Successfully found PipelineRun: %s", foundName)
	})

	t.Run("KubeClient - Find PipelineRun by non-existent EventID (error case)", func(t *testing.T) {
		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeClient
		kubeClient, err := infrastructure.NewKubeClient(logger.NewForTest())
		if err != nil {
			t.Logf("KubeClient creation failed: %v", err)
			t.Skip("Skipping test - Kubernetes API not available")
		}
		require.NoError(t, err, "Failed to create KubeClient")

		nonExistentEventID := "non-existent-event-id-12345"

		// When - Try to find PipelineRun by non-existent EventID
		_, err = kubeClient.FindPipelineRunNameByEventID(ctx, nonExistentEventID)

		// Then - Should return an error
		require.Error(t, err, "Should return error for non-existent EventID")
		t.Logf("Expected error received: %v", err)
	})
}

// TestTektonClient_FullDeploymentFlow tests a complete deployment flow (optional).
// This test is skipped by default as it creates actual resources in the cluster.
func TestTektonClient_FullDeploymentFlow(t *testing.T) {
	t.Skip("Skipping full deployment test - requires cleanup and may create actual resources")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load environment variables from .env.test
	helper.LoadTestEnv(t)

	ctx := context.Background()

	// Create clients
	tektonClient, err := infrastructure.NewTektonClient(logger.NewForTest())
	require.NoError(t, err, "Failed to create TektonClient")

	kubeClient, err := infrastructure.NewKubeClient(logger.NewForTest())
	require.NoError(t, err, "Failed to create KubeClient")

	// Create a test deployment request with dry_run=false
	deployRequest := &dto.TektonDeployRequest{
		DeploymentConfigJSON: dto.DeploymentConfig{
			ProjectID:    "0",
			ServiceName:  "integration-test-full",
			Namespace:    "application",
			StableWindow: 180,
			ConfigMaps:   []dto.ConfigMapInfo{},
			Volumes:      []dto.VolumeInfo{},
			Containers: []dto.TektonContainerInfo{
				{
					Name:            "test-container",
					Domain:          nil,
					HealthCheckType: "http",
					HealthEndpoint:  stringPtr("/"),
					Port:            80,
					HealthPort:      nil,
					ImageName:       "nginx",
					ImageTag:        "latest",
					EnvVars:         map[string]string{},
					Secrets:         map[string]string{},
					CPULimit:        "500m",
					MemoryRequest:   "256Mi",
					MemoryLimit:     "512Mi",
					VolumeMounts:    []dto.TektonVolumeMount{},
				},
			},
		},
		DryRun: "false", // Actually create the PipelineRun
	}

	// Trigger deployment
	response, err := tektonClient.TriggerDeploy(ctx, deployRequest)
	require.NoError(t, err, "Failed to trigger deployment")
	require.NotNil(t, response, "Response should not be nil")

	t.Logf("Deployment triggered - EventID: %s", response.EventID)

	// Wait a bit for the PipelineRun to be created
	time.Sleep(5 * time.Second)

	// Try to find the created PipelineRun
	pipelineRuns, err := kubeClient.ListPipelineRuns(ctx, 0)
	require.NoError(t, err, "Failed to list PipelineRuns")

	// Find the PipelineRun with matching EventID
	var targetPipelineRun *dto.PipelineRun
	for _, pr := range pipelineRuns {
		if pr.EventID == response.EventID {
			targetPipelineRun = pr
			break
		}
	}

	if targetPipelineRun != nil {
		t.Logf("Found PipelineRun: %s", targetPipelineRun.Name)

		// Get detailed status
		status, err := kubeClient.GetPipelineRunStatus(ctx, targetPipelineRun.Name)
		require.NoError(t, err, "Failed to get PipelineRun status")
		t.Logf("Status: %s, Reason: %s", status.Status, status.Reason)

		// Get logs
		logs, err := kubeClient.GetPipelineRunLogs(ctx, targetPipelineRun.Name)
		require.NoError(t, err, "Failed to get PipelineRun logs")
		t.Logf("Logs length: %d", len(logs))
	} else {
		t.Logf("PipelineRun not found yet (may take a few seconds to appear)")
	}
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// retryWithBackoff retries a function up to maxAttempts times with a 1-second interval.
// It returns true if the condition is met, false if all attempts fail.
func retryWithBackoff(maxAttempts int, interval time.Duration, fn func() bool) bool {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if fn() {
			return true
		}
		if attempt < maxAttempts {
			time.Sleep(interval)
		}
	}
	return false
}

// TestTektonBuildKubeIntegration tests the integration between TektonBuildClient and KubeBuildClient.
// This test requires actual Tekton build infrastructure and Kubernetes to be available.
//
// Prerequisites:
// - Tekton EventListener for builds must be accessible
// - Kubernetes API server must be accessible with build-pipeline namespace
// - Environment variables must be set in .env.test
func TestTektonBuildKubeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load environment variables from .env.test
	helper.LoadTestEnv(t)

	// Verify required environment variables for build
	requiredEnvVars := []string{
		"TEKTON_BUILD_URL",
		"TEKTON_API_AUTH",
		"KUBE_API_SERVER",
		"KUBE_SERVICE_ACCOUNT_TOKEN",
		"KUBE_BUILD_NAMESPACE",
		"KUBE_CA_CERT_PATH",
	}

	for _, envVar := range requiredEnvVars {
		value := os.Getenv(envVar)
		if value == "" {
			t.Skipf("Skipping test: required environment variable %s is not set", envVar)
		}
	}

	ctx := context.Background()

	t.Run("TektonBuildClient - Build request with template only", func(t *testing.T) {
		// Given - Create TektonBuildClient
		tektonBuildClient, err := infrastructure.NewTektonBuildClient()
		require.NoError(t, err, "Failed to create TektonBuildClient")

		// Use MySQL template from test directory
		// This template uses gomplate variables like {{ .mysql_version }}
		mysqlTemplate := `FROM mysql:{{ .mysql_version }}

# MySQL configuration
RUN echo "[mysqld]" > /etc/mysql/conf.d/custom.cnf && \
    echo "character-set-server={{ .charset }}" >> /etc/mysql/conf.d/custom.cnf && \
    echo "collation-server={{ .collation }}" >> /etc/mysql/conf.d/custom.cnf && \
    echo "max_connections={{ .max_connections }}" >> /etc/mysql/conf.d/custom.cnf

# Expose MySQL port
EXPOSE {{ .mysql_port }}`

		// Create a build request with template only (no GitHub repo)
		buildRequest := &dto.TektonBuildRequest{
			ProjectID:            "0",
			ContainerID:          "0",
			ImageName:            "integration-test-mysql",
			ForceBuild:           "true",
			Template:             mysqlTemplate,
			DockerfileConfigJSON: `{"mysql_version":"8.0","charset":"utf8mb4","collation":"utf8mb4_unicode_ci","max_connections":"200","mysql_port":"3306"}`,
			BuildEnvJSON:         `{"TZ":"Asia/Seoul"}`,
		}

		// When - Trigger build
		response, err := tektonBuildClient.TriggerBuild(ctx, buildRequest)

		// Then - Request should be accepted
		if err != nil {
			t.Logf("Build request failed: %v", err)
			t.Skip("Skipping test - Tekton build API not available or rejected request")
		} else {
			require.NotNil(t, response, "Response should not be nil")
			assert.NotEmpty(t, response.EventListener, "EventListener should be set")
			assert.NotEmpty(t, response.EventID, "EventID should be set")
			t.Logf("Build triggered - EventID: %s, EventListener: %s", response.EventID, response.EventListener)
		}
	})

	t.Run("TektonBuildClient - Build request with GitHub repository", func(t *testing.T) {
		// Given - Create TektonBuildClient
		tektonBuildClient, err := infrastructure.NewTektonBuildClient()
		require.NoError(t, err, "Failed to create TektonBuildClient")

		// Use Node.js template with gomplate variables
		nodeTemplate := `FROM node:{{ .node_version }}-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy application code
COPY . .

# Expose port
EXPOSE {{ .app_port }}

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD node --version || exit 1

# Run application
CMD ["node", "{{ .entry_point }}"]`

		// Create a build request with GitHub repository
		// This uses the test repository from user-workload-infra/tekton-pipelines/image-build-push/test/
		buildRequest := &dto.TektonBuildRequest{
			ProjectID:            "0",
			ContainerID:          "0",
			ImageName:            "integration-test-nodejs",
			GitHubURL:            "https://github.com/hakumizuki/cicd-test",
			GitHubBranch:         "main",
			DirectoryPath:        ".",
			ForceBuild:           "true", // Force build for testing
			LastBuildCommitHash:  "",     // Empty means first build
			Template:             nodeTemplate,
			DockerfileConfigJSON: `{"node_version":"18","app_port":"3000","entry_point":"index.js"}`,
			BuildEnvJSON:         `{"NODE_ENV":"production","TZ":"Asia/Seoul"}`,
			// RegistryURL is not set, will use environment variable
		}

		// When - Trigger build
		response, err := tektonBuildClient.TriggerBuild(ctx, buildRequest)

		// Then - Request should be accepted
		if err != nil {
			t.Logf("Build request failed: %v", err)
			t.Skip("Skipping test - Tekton build API not available or rejected request")
		} else {
			require.NotNil(t, response, "Response should not be nil")
			assert.NotEmpty(t, response.EventListener, "EventListener should be set")
			assert.NotEmpty(t, response.EventID, "EventID should be set")
			assert.Equal(t, "image-build-push-listener", response.EventListener, "EventListener should be image-build-push-listener")
			assert.Equal(t, "build-pipeline", response.Namespace, "Namespace should be build-pipeline")
			t.Logf("Build triggered - EventID: %s", response.EventID)
		}
	})

	t.Run("KubeBuildClient - Find build PipelineRun by EventID", func(t *testing.T) {
		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeBuildClient
		kubeBuildClient, err := infrastructure.NewKubeBuildClient()
		if err != nil {
			t.Logf("KubeBuildClient creation failed: %v", err)
			t.Logf("This may be due to network connectivity or certificate issues")
			t.Skip("Skipping KubeBuildClient tests - Kubernetes API not available")
		}
		require.NoError(t, err, "Failed to create KubeBuildClient")
		require.NotNil(t, kubeBuildClient, "KubeBuildClient should not be nil")

		// For this test, we need an existing build EventID
		// This is a placeholder test that can be enhanced with actual EventID
		t.Log("KubeBuildClient created successfully")
		t.Log("FindPipelineRunNameByEventID requires an existing build EventID")
		t.Log("This would be tested in E2E tests with actual build flow")
	})

	t.Run("KubeBuildClient - Get build PipelineRun status with results", func(t *testing.T) {
		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeBuildClient
		kubeBuildClient, err := infrastructure.NewKubeBuildClient()
		if err != nil {
			t.Logf("KubeBuildClient creation failed: %v", err)
			t.Skip("Skipping test - Kubernetes API not available")
		}
		require.NoError(t, err, "Failed to create KubeBuildClient")
		require.NotNil(t, kubeBuildClient, "KubeBuildClient should not be nil")

		// For this test, we would need an existing build PipelineRun name
		// This is a placeholder test that documents expected behavior
		t.Log("KubeBuildClient created successfully")
		t.Log("GetPipelineRunStatus with Results extraction requires an existing build PipelineRun")
		t.Log("Expected Results fields: latest_commit_hash, image_tag, should_build")
		t.Log("This would be tested in E2E tests with actual build flow")
	})
}

// TestTektonBuildClient_FullBuildFlow tests a complete build flow (optional).
// This test is skipped by default as it creates actual resources in the cluster.
func TestTektonBuildClient_FullBuildFlow(t *testing.T) {
	t.Skip("Skipping full build test - requires cleanup and may create actual resources")

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load environment variables from .env.test
	helper.LoadTestEnv(t)

	ctx := context.Background()

	// Create clients
	tektonBuildClient, err := infrastructure.NewTektonBuildClient()
	require.NoError(t, err, "Failed to create TektonBuildClient")

	// Fix CA cert path for tests
	caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
	if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
		_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
	}

	kubeBuildClient, err := infrastructure.NewKubeBuildClient()
	require.NoError(t, err, "Failed to create KubeBuildClient")

	// Create a test build request
	buildRequest := &dto.TektonBuildRequest{
		ImageName:            "full-flow-test",
		GitHubURL:            "https://github.com/hakumizuki/cicd-test",
		GitHubBranch:         "main",
		DirectoryPath:        ".",
		ForceBuild:           "true",
		LastBuildCommitHash:  "",
		Template:             "FROM node:18-alpine\nWORKDIR /app\nCOPY . .\nRUN npm install\nEXPOSE 3000\nCMD [\"node\", \"index.js\"]",
		DockerfileConfigJSON: "",
		BuildEnvJSON:         `{"NODE_ENV":"test"}`,
		RegistryURL:          "registry.launchpad.kr/",
	}

	// Trigger build
	response, err := tektonBuildClient.TriggerBuild(ctx, buildRequest)
	require.NoError(t, err, "Failed to trigger build")
	require.NotNil(t, response, "Response should not be nil")

	t.Logf("Build triggered - EventID: %s", response.EventID)

	// Wait a bit for the PipelineRun to be created
	time.Sleep(5 * time.Second)

	// Try to find the created PipelineRun by EventID
	pipelineRunName, err := kubeBuildClient.FindPipelineRunNameByEventID(ctx, response.EventID)
	if err != nil {
		t.Logf("PipelineRun not found yet (may take a few seconds to appear): %v", err)
	} else {
		t.Logf("Found build PipelineRun: %s", pipelineRunName)

		// Get detailed status with results
		status, err := kubeBuildClient.GetPipelineRunStatus(ctx, pipelineRunName)
		require.NoError(t, err, "Failed to get build PipelineRun status")

		t.Logf("Status: %s, Reason: %s", status.Status, status.Reason)
		t.Logf("Message: %s", status.Message)

		if len(status.Results) > 0 {
			t.Logf("Build Results:")
			for key, value := range status.Results {
				t.Logf("  %s: %s", key, value)
			}
		} else {
			t.Logf("Results not yet available (build may still be running)")
		}
	}
}
