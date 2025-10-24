package infrastructure

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

// TestNewKubeBuildClient_EnvironmentVariables tests that NewKubeBuildClient properly validates
// required environment variables.
func TestNewKubeBuildClient_EnvironmentVariables(t *testing.T) {
	// Save original environment
	originalAPIServer := os.Getenv("KUBE_API_SERVER")
	originalToken := os.Getenv("KUBE_SERVICE_ACCOUNT_TOKEN")
	originalNamespace := os.Getenv("KUBE_BUILD_NAMESPACE")
	originalCACertPath := os.Getenv("KUBE_CA_CERT_PATH")

	// Restore environment after test
	defer func() {
		_ = os.Setenv("KUBE_API_SERVER", originalAPIServer)
		_ = os.Setenv("KUBE_SERVICE_ACCOUNT_TOKEN", originalToken)
		_ = os.Setenv("KUBE_BUILD_NAMESPACE", originalNamespace)
		_ = os.Setenv("KUBE_CA_CERT_PATH", originalCACertPath)
	}()

	tests := []struct {
		name          string
		apiServer     string
		token         string
		namespace     string
		caCertPath    string
		expectedError error
		shouldSucceed bool
	}{
		{
			name:          "missing KUBE_API_SERVER",
			apiServer:     "",
			token:         "test-token",
			namespace:     "build-pipeline",
			caCertPath:    "",
			expectedError: projecterrors.ErrKubernetesUnavailable,
			shouldSucceed: false,
		},
		{
			name:          "missing KUBE_SERVICE_ACCOUNT_TOKEN",
			apiServer:     "https://test-server",
			token:         "",
			namespace:     "build-pipeline",
			caCertPath:    "",
			expectedError: projecterrors.ErrKubernetesUnavailable,
			shouldSucceed: false,
		},
		{
			name:          "missing KUBE_BUILD_NAMESPACE",
			apiServer:     "https://test-server",
			token:         "test-token",
			namespace:     "",
			caCertPath:    "",
			expectedError: projecterrors.ErrKubernetesUnavailable,
			shouldSucceed: false,
		},
		{
			name:          "missing KUBE_CA_CERT_PATH",
			apiServer:     "https://test-server",
			token:         "test-token",
			namespace:     "build-pipeline",
			caCertPath:    "",
			expectedError: projecterrors.ErrKubernetesUnavailable,
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset environment variables for this test
			if tt.apiServer != "" {
				_ = os.Setenv("KUBE_API_SERVER", tt.apiServer)
			} else {
				_ = os.Unsetenv("KUBE_API_SERVER")
			}

			if tt.token != "" {
				_ = os.Setenv("KUBE_SERVICE_ACCOUNT_TOKEN", tt.token)
			} else {
				_ = os.Unsetenv("KUBE_SERVICE_ACCOUNT_TOKEN")
			}

			if tt.namespace != "" {
				_ = os.Setenv("KUBE_BUILD_NAMESPACE", tt.namespace)
			} else {
				_ = os.Unsetenv("KUBE_BUILD_NAMESPACE")
			}

			if tt.caCertPath != "" {
				_ = os.Setenv("KUBE_CA_CERT_PATH", tt.caCertPath)
			} else {
				_ = os.Unsetenv("KUBE_CA_CERT_PATH")
			}

			// Create client
			client, err := NewKubeBuildClient(logger.NewForTest())

			if tt.shouldSucceed {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			} else {
				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Equal(t, tt.expectedError, err)
			}
		})
	}
}

// Note: Integration tests for actual Kubernetes API calls should be in integration_test.go
// or e2e tests, as they require a real Kubernetes cluster with Tekton installed.
