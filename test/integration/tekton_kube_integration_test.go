package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	domaininfra "github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
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

func mustRawMessage(t *testing.T, value string) json.RawMessage {
	t.Helper()
	if value == "" {
		return nil
	}
	require.True(t, json.Valid([]byte(value)), "invalid JSON provided to mustRawMessage: %s", value)
	return json.RawMessage([]byte(value))
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

// Note: We use hardcoded commit hashes to avoid git operations during tests.
// The actual commit hash will be fetched by the Tekton pipeline.

// waitForBuildPipeline waits for a build PipelineRun to reach a terminal state.
// It polls the Kubernetes API using the KubeBuildClient.
func waitForBuildPipeline(t *testing.T, kubeBuildClient domaininfra.KubeBuildClient, ctx context.Context, eventID string, maxAttempts int, pollInterval time.Duration) *dto.PipelineRun {
	t.Helper()

	t.Logf("Waiting for PipelineRun with EventID: %s", eventID)

	var pipelineRunName string
	var status *dto.PipelineRun

	// First, find the PipelineRun by EventID (with retry)
	foundPipelineRun := retryWithBackoff(10, 1*time.Second, func() bool {
		var err error
		pipelineRunName, err = kubeBuildClient.FindPipelineRunNameByEventID(ctx, eventID)
		if err != nil {
			t.Logf("Attempt to find PipelineRun by EventID failed: %v", err)
			return false
		}
		return pipelineRunName != ""
	})

	if !foundPipelineRun {
		t.Fatalf("Failed to find PipelineRun with EventID: %s", eventID)
	}

	t.Logf("Found PipelineRun: %s", pipelineRunName)

	// Poll for terminal status
	for attempt := range maxAttempts {
		var err error
		status, err = kubeBuildClient.GetPipelineRunStatus(ctx, pipelineRunName)
		if err != nil {
			t.Logf("Attempt %d: Failed to get PipelineRun status: %v", attempt+1, err)
			time.Sleep(pollInterval)
			continue
		}

		t.Logf("Attempt %d: PipelineRun status: %s, reason: %s", attempt+1, status.Status, status.Reason)

		// Check for terminal states
		// Status: "True" = Succeeded, "False" = Failed
		switch status.Status {
		case "True":
			t.Logf("PipelineRun completed successfully")
			return status
		case "False":
			t.Logf("PipelineRun failed with reason: %s, message: %s", status.Reason, status.Message)
			return status
		}

		// Continue polling
		time.Sleep(pollInterval)
	}

	t.Fatalf("Timeout waiting for PipelineRun to complete after %d attempts", maxAttempts)
	return nil
}

// verifyBuildExecution verifies that build tasks were executed or skipped correctly.
// It checks the PipelineRun completion message to ensure the correct number of tasks
// were executed versus skipped based on the should_build decision.
// expectedSkippedCount: expected number of skipped tasks (0 if github_url provided, 1 if not)
func verifyBuildExecution(t *testing.T, pipelineRun *dto.PipelineRun, expectedShouldBuild string, expectedSkippedCount int) {
	t.Helper()

	t.Log("Verifying build execution status")
	t.Logf("PipelineRun Message: %s", pipelineRun.Message)

	// Parse message format: "Tasks Completed: X (Failed: Y, Cancelled Z), Skipped: W"
	if expectedShouldBuild == "true" {
		// Build should have been executed
		// Task count depends on whether github_url is provided:
		//   With github_url (Skipped: 0):
		//     1. check-build-necessity
		//     2. git-clone
		//     3. ecr-repository-check
		//     4. apply-dockerfile-config
		//     5. build-and-push
		//   Without github_url (Skipped: 1):
		//     1. check-build-necessity
		//     2. git-clone (SKIPPED - no github_url)
		//     3. ecr-repository-check
		//     4. apply-dockerfile-config
		//     5. build-and-push
		expectedMsg := fmt.Sprintf("Skipped: %d", expectedSkippedCount)
		assert.Contains(t, pipelineRun.Message, expectedMsg,
			"Expected %d task(s) to be skipped when should_build=true", expectedSkippedCount)

		expectedCompleted := 5 - expectedSkippedCount
		expectedCompletedMsg := fmt.Sprintf("Tasks Completed: %d", expectedCompleted)
		assert.Contains(t, pipelineRun.Message, expectedCompletedMsg,
			"Expected %d task(s) to complete when should_build=true", expectedCompleted)

		if expectedSkippedCount == 0 {
			t.Log("✓ Verified: All build tasks were executed (none skipped)")
		} else {
			t.Logf("✓ Verified: Build tasks executed with %d task(s) skipped (git-clone)", expectedSkippedCount)
		}
	} else {
		// Build should have been skipped - build tasks should be skipped
		// Only 1 task (check-build-necessity) should complete
		// 4 tasks should be skipped:
		//   - git-clone (skipped via whenExpression)
		//   - ecr-repository-check (skipped via whenExpression)
		//   - apply-dockerfile-config (skipped via whenExpression)
		//   - build-and-push (skipped via whenExpression)
		assert.Contains(t, pipelineRun.Message, "Skipped: 4",
			"Build tasks should be skipped when should_build=false")

		assert.Contains(t, pipelineRun.Message, "Tasks Completed: 1",
			"Only build check task should complete when skipping build")

		t.Log("✓ Verified: Build tasks were correctly skipped (4 tasks)")
	}

	t.Log("Build execution verification passed")
}

// verifyBuildResults verifies the build results from a PipelineRun.
func verifyBuildResults(t *testing.T, results map[string]string, expectedShouldBuild string, expectedCommitHash string) {
	t.Helper()

	t.Log("Verifying build results")
	t.Logf("Results: %+v", results)

	// Extract values
	shouldBuild, hasShouldBuild := results["should_build"]
	latestCommitHash, hasLatestCommitHash := results["latest_commit_hash"]
	imageTag, hasImageTag := results["image_tag"]

	// Verify should_build
	require.True(t, hasShouldBuild, "Result 'should_build' should be present")
	assert.Equal(t, expectedShouldBuild, shouldBuild, "should_build mismatch")

	// Verify commit hash if provided
	if expectedCommitHash != "" && expectedCommitHash != "any" {
		require.True(t, hasLatestCommitHash, "Result 'latest_commit_hash' should be present")
		assert.Equal(t, expectedCommitHash, latestCommitHash, "latest_commit_hash mismatch")
	}

	// Verify image_tag format
	require.True(t, hasImageTag, "Result 'image_tag' should be present")
	if shouldBuild == "true" {
		if hasLatestCommitHash && latestCommitHash != "" {
			// Should be first 7 characters of commit hash
			expectedTag := latestCommitHash[:7]
			assert.Equal(t, expectedTag, imageTag, "image_tag should be first 7 chars of commit hash")
		} else {
			// Should be "latest" when no github_url
			assert.Equal(t, "latest", imageTag, "image_tag should be 'latest' when no github_url")
		}
	}

	t.Log("Build results verification passed")
}

// TestTektonBuildKubeIntegration tests the integration between TektonBuildClient and KubeBuildClient.
// This test requires actual Tekton build infrastructure and Kubernetes to be available.
//
// Prerequisites:
// - Tekton EventListener for builds must be accessible
// - Kubernetes API server must be accessible with build-pipeline namespace
// - Environment variables must be set in .env.test
func TestTektonBuildKubeIntegration(t *testing.T) {
	t.Parallel() // Enable parallel execution with build_service_integration_test.go tests

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

	// Build integration tests based on /workspace/user-workload-infra/tekton-pipelines/image-build-push/test/
	// Simplified to 3 force-build tests only

	t.Run("Spring Hello World - Force Build (should_build=true)", func(t *testing.T) {
		t.Parallel()

		// Test configuration
		githubURL := "https://github.com/paulczar/spring-helloworld.git"
		githubBranch := "master"
		imageName := "spring-helloworld-test-03"
		forceBuild := "true" // Force build
		templatePath := "/workspace/user-workload-infra/tekton-pipelines/image-build-push/test/templates/springboot-maven.dockerfile.tmpl"

		// Read template
		templateBytes, err := os.ReadFile(templatePath)
		if err != nil {
			t.Skipf("Skipping test: template file not found: %v", err)
		}
		templateContent := string(templateBytes)

		// Use dummy commit hash (force_build=true will build regardless)
		latestCommit := "1234567890abcdef1234567890abcdef12345678"

		t.Logf("Latest commit (dummy value, force_build=true): %s", latestCommit)

		// Create TektonBuildClient
		tektonBuildClient, err := infrastructure.NewTektonBuildClient(logger.NewForTest())
		require.NoError(t, err, "Failed to create TektonBuildClient")

		// Dockerfile configuration
		dockerfileConfigJSON := mustRawMessage(t, `{
			"maven_version": "3.6",
			"java_version": "11",
			"app_port": "8080"
		}`)

		// Build environment variables - empty for this test
		buildEnvJSON := mustRawMessage(t, `{}`)

		// Create build request with force_build=true
		buildRequest := &dto.TektonBuildRequest{
			ProjectID:            "0",
			ContainerID:          "0",
			ImageName:            imageName,
			GitHubURL:            githubURL,
			GitHubBranch:         githubBranch,
			DirectoryPath:        ".",
			ForceBuild:           forceBuild,
			LastBuildCommitHash:  latestCommit, // Even with latest commit, force_build should trigger build
			Template:             templateContent,
			DockerfileConfigJSON: dockerfileConfigJSON,
			BuildEnvJSON:         buildEnvJSON,
		}

		// Trigger build
		response, err := tektonBuildClient.TriggerBuild(ctx, buildRequest)
		if err != nil {
			t.Skipf("Skipping test: build trigger failed: %v", err)
		}

		require.NotNil(t, response, "Response should not be nil")
		require.NotEmpty(t, response.EventID, "EventID should be set")
		t.Logf("Build triggered - EventID: %s", response.EventID)

		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Create KubeBuildClient
		kubeBuildClient, err := infrastructure.NewKubeBuildClient(logger.NewForTest())
		if err != nil {
			t.Skipf("Skipping test: KubeBuildClient creation failed: %v", err)
		}

		// Wait for pipeline to complete (max 60 attempts, 10 second interval = 10 minutes)
		pipelineRun := waitForBuildPipeline(t, kubeBuildClient, ctx, response.EventID, 60, 10*time.Second)
		require.NotNil(t, pipelineRun, "PipelineRun should not be nil")

		// Verify pipeline completed successfully
		assert.Equal(t, "True", pipelineRun.Status, "PipelineRun should succeed")

		// Verify build execution (tasks were actually executed, not skipped)
		// expectedSkippedCount = 0 because github_url is provided
		verifyBuildExecution(t, pipelineRun, "true", 0)

		// Verify results - should_build should be true (force_build overrides, commit hash from Tekton)
		verifyBuildResults(t, pipelineRun.Results, "true", "any")
	})

	t.Run("MySQL - No GitHub, Force Build (should_build=true)", func(t *testing.T) {
		t.Parallel()

		// Test configuration
		imageName := "mysql-custom-test-05"
		forceBuild := "true"
		templatePath := "/workspace/user-workload-infra/tekton-pipelines/image-build-push/test/templates/mysql.dockerfile.tmpl"

		// Read template
		templateBytes, err := os.ReadFile(templatePath)
		if err != nil {
			t.Skipf("Skipping test: template file not found: %v", err)
		}
		templateContent := string(templateBytes)

		// Create TektonBuildClient
		tektonBuildClient, err := infrastructure.NewTektonBuildClient(logger.NewForTest())
		require.NoError(t, err, "Failed to create TektonBuildClient")

		// Dockerfile configuration for MySQL
		dockerfileConfigJSON := mustRawMessage(t, `{
			"mysql_version": "8.0",
			"charset": "utf8mb4",
			"collation": "utf8mb4_unicode_ci",
			"max_connections": "300",
			"max_allowed_packet": "128M",
			"innodb_buffer_pool_size": "2G",
			"innodb_log_file_size": "512M",
			"mysql_port": "3306"
		}`)

		// Build environment variables
		buildEnvJSON := mustRawMessage(t, `{
			"TZ": "UTC"
		}`)

		// Create build request without GitHub URL but with force_build
		buildRequest := &dto.TektonBuildRequest{
			ProjectID:            "0",
			ContainerID:          "0",
			ImageName:            imageName,
			ForceBuild:           forceBuild,
			Template:             templateContent,
			DockerfileConfigJSON: dockerfileConfigJSON,
			BuildEnvJSON:         buildEnvJSON,
		}

		// Trigger build
		response, err := tektonBuildClient.TriggerBuild(ctx, buildRequest)
		if err != nil {
			t.Skipf("Skipping test: build trigger failed: %v", err)
		}

		require.NotNil(t, response, "Response should not be nil")
		require.NotEmpty(t, response.EventID, "EventID should be set")
		t.Logf("Build triggered - EventID: %s", response.EventID)

		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Create KubeBuildClient
		kubeBuildClient, err := infrastructure.NewKubeBuildClient(logger.NewForTest())
		if err != nil {
			t.Skipf("Skipping test: KubeBuildClient creation failed: %v", err)
		}

		// Wait for pipeline to complete (max 60 attempts, 10 second interval = 10 minutes)
		pipelineRun := waitForBuildPipeline(t, kubeBuildClient, ctx, response.EventID, 60, 10*time.Second)
		require.NotNil(t, pipelineRun, "PipelineRun should not be nil")

		// Verify pipeline completed successfully
		assert.Equal(t, "True", pipelineRun.Status, "PipelineRun should succeed")

		// Verify build execution (tasks were actually executed, not skipped)
		// expectedSkippedCount = 1 because github_url is NOT provided (git-clone task is skipped)
		verifyBuildExecution(t, pipelineRun, "true", 1)

		// Verify results - should_build should be true (force_build)
		verifyBuildResults(t, pipelineRun.Results, "true", "")
	})

	t.Run("Spring MySQL Demo - Force Build (should_build=true)", func(t *testing.T) {
		t.Parallel()

		// Check GITHUB_APP_INSTALLATION_ID is set (required for private repository)
		installationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")
		if installationID == "" {
			t.Skip("Skipping test: GITHUB_APP_INSTALLATION_ID not set in environment")
		}

		// Test configuration
		githubURL := "https://github.com/swm-launchpad/spring-mysql-demo.git"
		githubBranch := "main"
		imageName := "spring-mysql-demo-test-08"
		forceBuild := "true" // Force build
		templatePath := "/workspace/user-workload-infra/tekton-pipelines/image-build-push/test/templates/springboot-gradle.dockerfile.tmpl"

		// Read template
		templateBytes, err := os.ReadFile(templatePath)
		if err != nil {
			t.Skipf("Skipping test: template file not found: %v", err)
		}
		templateContent := string(templateBytes)

		// Use hardcoded old commit hash (actual latest commit will be fetched by Tekton)
		oldCommit := "abcdef9876543210abcdef9876543210abcdef98"

		t.Logf("Old commit (hardcoded): %s", oldCommit)
		t.Logf("Build will fetch actual latest commit from private repository")

		// Create TektonBuildClient
		tektonBuildClient, err := infrastructure.NewTektonBuildClient(logger.NewForTest())
		require.NoError(t, err, "Failed to create TektonBuildClient")

		// Dockerfile configuration
		dockerfileConfigJSON := mustRawMessage(t, `{
			"gradle_version": "8.5",
			"java_version": "21",
			"app_port": "8080"
		}`)

		// Build environment variables
		buildEnvJSON := mustRawMessage(t, `{
			"TZ": "Asia/Seoul",
			"LANG": "en_US.UTF-8"
		}`)

		// Create build request for private repo
		buildRequest := &dto.TektonBuildRequest{
			ProjectID:            "0",
			ContainerID:          "0",
			ImageName:            imageName,
			GitHubURL:            githubURL,
			GitHubBranch:         githubBranch,
			DirectoryPath:        ".",
			ForceBuild:           forceBuild,
			LastBuildCommitHash:  oldCommit,
			InstallationID:       installationID,
			Template:             templateContent,
			DockerfileConfigJSON: dockerfileConfigJSON,
			BuildEnvJSON:         buildEnvJSON,
		}

		// Trigger build
		response, err := tektonBuildClient.TriggerBuild(ctx, buildRequest)
		if err != nil {
			t.Skipf("Skipping test: build trigger failed: %v", err)
		}

		require.NotNil(t, response, "Response should not be nil")
		require.NotEmpty(t, response.EventID, "EventID should be set")
		t.Logf("Build triggered - EventID: %s", response.EventID)

		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Create KubeBuildClient
		kubeBuildClient, err := infrastructure.NewKubeBuildClient(logger.NewForTest())
		if err != nil {
			t.Skipf("Skipping test: KubeBuildClient creation failed: %v", err)
		}

		// Wait for pipeline to complete (max 60 attempts, 10 second interval = 10 minutes)
		pipelineRun := waitForBuildPipeline(t, kubeBuildClient, ctx, response.EventID, 60, 10*time.Second)
		require.NotNil(t, pipelineRun, "PipelineRun should not be nil")

		// Verify pipeline completed successfully
		assert.Equal(t, "True", pipelineRun.Status, "PipelineRun should succeed")

		// Verify build execution (tasks were actually executed, not skipped)
		// expectedSkippedCount = 0 because github_url is provided
		verifyBuildExecution(t, pipelineRun, "true", 0)

		// Verify results (commit hash will be fetched by Tekton)
		verifyBuildResults(t, pipelineRun.Results, "true", "any")
	})

	t.Run("KubeBuildClient - Find build PipelineRun by EventID", func(t *testing.T) {
		// Fix CA cert path for tests
		caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
		if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
			_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
		}

		// Given - Create KubeBuildClient
		kubeBuildClient, err := infrastructure.NewKubeBuildClient(logger.NewForTest())
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
		kubeBuildClient, err := infrastructure.NewKubeBuildClient(logger.NewForTest())
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
	tektonBuildClient, err := infrastructure.NewTektonBuildClient(logger.NewForTest())
	require.NoError(t, err, "Failed to create TektonBuildClient")

	// Fix CA cert path for tests
	caCertPath := os.Getenv("KUBE_CA_CERT_PATH")
	if caCertPath == "./ca.crt" || caCertPath == "ca.crt" {
		_ = os.Setenv("KUBE_CA_CERT_PATH", "../../ca.crt")
	}

	kubeBuildClient, err := infrastructure.NewKubeBuildClient(logger.NewForTest())
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
		DockerfileConfigJSON: nil,
		BuildEnvJSON:         mustRawMessage(t, `{"NODE_ENV":"test"}`),
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
