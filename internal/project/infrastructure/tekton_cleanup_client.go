package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"go.uber.org/zap"
)

// tektonCleanupClient implements the TektonCleanupClient interface using HTTP requests.
// It communicates with the Tekton cleanup API to trigger resource cleanup.
type tektonCleanupClient struct {
	cleanupURL string
	authHeader string
	httpClient *http.Client
	logger     logger.Logger
}

// tektonCleanupRequest represents the request body for the cleanup API.
type tektonCleanupRequest struct {
	ProjectID string `json:"project_id"`
	DryRun    string `json:"dry_run"`
	Namespace string `json:"namespace"`
}

// NewTektonCleanupClient creates a new Tekton cleanup client using configuration from environment variables.
//
// Required environment variables:
//   - TEKTON_CLEANUP_URL: The Tekton cleanup endpoint URL (default: "https://tekton-api.launchpad.kr/cleanup")
//   - TEKTON_API_AUTH: The Basic authentication header value (e.g., "Basic base64encodedcredentials")
//
// Returns an error if any required environment variable is missing.
func NewTektonCleanupClient(log logger.Logger) (infrastructure.TektonCleanupClient, error) {
	cleanupURL := os.Getenv("TEKTON_CLEANUP_URL")
	if cleanupURL == "" {
		// Default to the standard cleanup endpoint
		cleanupURL = "https://tekton-api.launchpad.kr/cleanup"
	}

	authHeader := os.Getenv("TEKTON_API_AUTH")
	if authHeader == "" {
		return nil, projecterrors.ErrTektonUnavailable
	}

	// Create HTTP client with reasonable timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &tektonCleanupClient{
		cleanupURL: cleanupURL,
		authHeader: authHeader,
		httpClient: httpClient,
		logger:     log,
	}, nil
}

// TriggerCleanup sends a cleanup request to the Tekton cleanup API.
// This is a fire-and-forget operation that triggers resource cleanup asynchronously.
func (t *tektonCleanupClient) TriggerCleanup(ctx context.Context, projectID, namespace string) error {
	t.logger.Info(ctx, "tekton cleanup client trigger started",
		zap.String("project_id", projectID),
		zap.String("namespace", namespace),
	)

	// If namespace is empty, use default
	if namespace == "" {
		namespace = "application"
	}

	// Create request body
	requestBody := tektonCleanupRequest{
		ProjectID: projectID,
		DryRun:    "false", // Always perform actual cleanup
		Namespace: namespace,
	}

	// Marshal request to JSON
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		t.logger.Error(ctx, "tekton cleanup client request marshaling failed",
			zap.String("project_id", projectID),
			zap.Error(err),
		)
		return projecterrors.ErrInvalidDeploymentRequest
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.cleanupURL, bytes.NewBuffer(requestBodyJSON))
	if err != nil {
		t.logger.Error(ctx, "tekton cleanup client http request creation failed",
			zap.String("cleanup_url", t.cleanupURL),
			zap.Error(err),
		)
		return projecterrors.ErrTektonUnavailable
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", t.authHeader)

	t.logger.Info(ctx, "tekton cleanup client sending http request",
		zap.String("method", http.MethodPost),
		zap.String("url", t.cleanupURL),
		zap.String("project_id", projectID),
	)

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		t.logger.Error(ctx, "tekton cleanup client http request failed",
			zap.String("cleanup_url", t.cleanupURL),
			zap.Error(err),
		)
		return projecterrors.ErrTektonUnavailable
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.logger.Error(ctx, "tekton cleanup client failed to close response body",
				zap.Error(closeErr),
			)
		}
	}()

	// Read response body for logging
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.logger.Error(ctx, "tekton cleanup client failed to read response body",
			zap.Error(err),
		)
		return projecterrors.ErrTektonUnavailable
	}

	t.logger.Info(ctx, "tekton cleanup client received http response",
		zap.Int("status_code", resp.StatusCode),
		zap.Int("response_size", len(responseBody)),
	)

	// Check status code - accept any 2xx status as success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Extract error message from response body for debugging
		var errorMsg string
		if len(responseBody) > 0 {
			errorMsg = string(responseBody)
		} else {
			errorMsg = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		}

		// Map different status codes to appropriate errors
		switch resp.StatusCode {
		case http.StatusBadRequest:
			t.logger.Error(ctx, "tekton cleanup request validation failed",
				zap.Int("status_code", resp.StatusCode),
				zap.String("error_message", errorMsg),
			)
			return fmt.Errorf("%w: %s", projecterrors.ErrTektonDeploymentFailed, errorMsg)
		case http.StatusUnauthorized, http.StatusForbidden:
			t.logger.Error(ctx, "tekton cleanup authentication/authorization failed",
				zap.Int("status_code", resp.StatusCode),
				zap.String("error_message", errorMsg),
			)
			return fmt.Errorf("%w: %s", projecterrors.ErrTektonAuthenticationFailed, errorMsg)
		default:
			t.logger.Error(ctx, "tekton cleanup server error",
				zap.Int("status_code", resp.StatusCode),
				zap.String("error_message", errorMsg),
			)
			return fmt.Errorf("%w: %s", projecterrors.ErrTektonUnavailable, errorMsg)
		}
	}

	t.logger.Info(ctx, "tekton cleanup client trigger completed successfully",
		zap.String("project_id", projectID),
		zap.Int("status_code", resp.StatusCode),
	)

	return nil
}

// Compile-time assertion that tektonCleanupClient implements TektonCleanupClient interface
var _ infrastructure.TektonCleanupClient = (*tektonCleanupClient)(nil)
