package infrastructure

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewKubeClient_EnvironmentVariables tests that NewKubeClient properly validates
// required environment variables.
func TestNewKubeClient_EnvironmentVariables(t *testing.T) {
	// Save original environment
	originalAPIServer := os.Getenv("KUBE_API_SERVER")
	originalToken := os.Getenv("KUBE_SERVICE_ACCOUNT_TOKEN")
	originalNamespace := os.Getenv("KUBE_DEPLOY_NAMESPACE")

	// Restore environment after test
	defer func() {
		_ = os.Setenv("KUBE_API_SERVER", originalAPIServer)
		_ = os.Setenv("KUBE_SERVICE_ACCOUNT_TOKEN", originalToken)
		_ = os.Setenv("KUBE_DEPLOY_NAMESPACE", originalNamespace)
	}()

	tests := []struct {
		name          string
		apiServer     string
		token         string
		namespace     string
		expectedError string
		shouldSucceed bool
	}{
		{
			name:          "missing KUBE_API_SERVER",
			apiServer:     "",
			token:         "test-token",
			namespace:     "test-namespace",
			expectedError: "KUBE_API_SERVER environment variable is required",
			shouldSucceed: false,
		},
		{
			name:          "missing KUBE_SERVICE_ACCOUNT_TOKEN",
			apiServer:     "https://test-server",
			token:         "",
			namespace:     "test-namespace",
			expectedError: "KUBE_SERVICE_ACCOUNT_TOKEN environment variable is required",
			shouldSucceed: false,
		},
		{
			name:          "missing KUBE_DEPLOY_NAMESPACE",
			apiServer:     "https://test-server",
			token:         "test-token",
			namespace:     "",
			expectedError: "KUBE_DEPLOY_NAMESPACE environment variable is required",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			_ = os.Setenv("KUBE_API_SERVER", tt.apiServer)
			_ = os.Setenv("KUBE_SERVICE_ACCOUNT_TOKEN", tt.token)
			_ = os.Setenv("KUBE_DEPLOY_NAMESPACE", tt.namespace)

			// Create client
			client, err := NewKubeClient()

			if tt.shouldSucceed {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			} else {
				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// Note: Integration tests for actual Kubernetes API calls should be in integration_test.go
// or e2e tests, as they require a real Kubernetes cluster with Tekton installed.
