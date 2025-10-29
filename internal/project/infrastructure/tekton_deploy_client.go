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
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
	"go.uber.org/zap"
)

// tektonClient implements the TektonClient interface using HTTP requests.
// It communicates with the Tekton EventListener to trigger deployments.
type tektonClient struct {
	deployURL  string
	authHeader string
	httpClient *http.Client
	logger     logger.Logger
}

// NewTektonDeployClient creates a new Tekton client using configuration from environment variables.
//
// Required environment variables:
//   - TEKTON_DEPLOY_URL: The Tekton EventListener endpoint URL (e.g., "https://tekton-api.launchpad.kr/deploy")
//   - TEKTON_API_AUTH: The Basic authentication header value (e.g., "Basic base64encodedcredentials")
//
// Returns an error if any required environment variable is missing.
func NewTektonDeployClient(log logger.Logger) (infrastructure.TektonClient, error) {
	deployURL := os.Getenv("TEKTON_DEPLOY_URL")
	if deployURL == "" {
		return nil, projecterrors.ErrTektonUnavailable
	}

	authHeader := os.Getenv("TEKTON_API_AUTH")
	if authHeader == "" {
		return nil, projecterrors.ErrTektonUnavailable
	}

	// Create HTTP client with reasonable timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &tektonClient{
		deployURL:  deployURL,
		authHeader: authHeader,
		httpClient: httpClient,
		logger:     log,
	}, nil
}

// TriggerDeploy sends a deployment request to the Tekton EventListener.
// It returns the response from Tekton on success (HTTP 202 Accepted).
func (t *tektonClient) TriggerDeploy(ctx context.Context, request *dto.TektonDeployRequest) (*dto.TektonDeployResponse, error) {
	t.logger.Info(ctx, "tekton client trigger deploy started",
		zap.String("service_name", request.DeploymentConfigJSON.ServiceName),
		zap.String("project_id", request.DeploymentConfigJSON.ProjectID),
		zap.String("namespace", request.DeploymentConfigJSON.Namespace),
		zap.Int("container_count", len(request.DeploymentConfigJSON.Containers)),
	)

	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		// Request marshaling failed - this is a client-side validation error,
		// not a Tekton API response error
		t.logger.Error(ctx, "tekton client request marshaling failed",
			zap.String("service_name", request.DeploymentConfigJSON.ServiceName),
			zap.Error(err),
		)
		return nil, projecterrors.ErrInvalidDeploymentRequest
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.deployURL, bytes.NewBuffer(requestBody))
	if err != nil {
		t.logger.Error(ctx, "tekton client http request creation failed",
			zap.String("deploy_url", t.deployURL),
			zap.Error(err),
		)
		return nil, projecterrors.ErrTektonUnavailable
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", t.authHeader)

	t.logger.Info(ctx, "tekton client sending http request",
		zap.String("method", http.MethodPost),
		zap.String("url", t.deployURL),
	)

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		t.logger.Error(ctx, "tekton client http request failed",
			zap.String("deploy_url", t.deployURL),
			zap.Error(err),
		)
		return nil, projecterrors.ErrTektonUnavailable
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log close error but don't override the main error
			t.logger.Error(ctx, "tekton client failed to close response body",
				zap.Error(closeErr),
			)
		}
	}()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.logger.Error(ctx, "tekton client failed to read response body",
			zap.Error(err),
		)
		return nil, projecterrors.ErrTektonUnavailable
	}

	t.logger.Info(ctx, "tekton client received http response",
		zap.Int("status_code", resp.StatusCode),
		zap.Int("response_size", len(responseBody)),
	)

	// Check status code
	// Tekton EventListener returns 202 Accepted on success
	if resp.StatusCode != http.StatusAccepted {
		// Extract error message from response body for debugging
		var errorMsg string
		if len(responseBody) > 0 {
			errorMsg = string(responseBody)
		} else {
			errorMsg = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		}

		// Map different status codes to appropriate errors
		// Always include Tekton's response for debugging
		switch resp.StatusCode {
		case http.StatusBadRequest:
			// 400 Bad Request: deployment configuration validation failed
			t.logger.Error(ctx, "tekton deployment request validation failed",
				zap.Int("status_code", resp.StatusCode),
				zap.String("error_message", errorMsg),
				zap.Error(projecterrors.ErrTektonDeploymentFailed),
			)
			return nil, fmt.Errorf("%w: %s", projecterrors.ErrTektonDeploymentFailed, errorMsg)
		case http.StatusUnauthorized, http.StatusForbidden:
			// 401/403: authentication/authorization failure - infrastructure issue
			// This indicates wrong credentials or ACL changes, not a user error
			// Must be treated as infrastructure failure to trigger ops alerts
			t.logger.Error(ctx, "tekton authentication/authorization failed",
				zap.Int("status_code", resp.StatusCode),
				zap.String("error_message", errorMsg),
				zap.Error(projecterrors.ErrTektonAuthenticationFailed),
			)
			return nil, fmt.Errorf("%w: %s", projecterrors.ErrTektonAuthenticationFailed, errorMsg)
		default:
			// Server errors or other issues
			t.logger.Error(ctx, "tekton server error",
				zap.Int("status_code", resp.StatusCode),
				zap.String("error_message", errorMsg),
				zap.Error(projecterrors.ErrTektonUnavailable),
			)
			return nil, fmt.Errorf("%w: %s", projecterrors.ErrTektonUnavailable, errorMsg)
		}
	}

	// Parse response
	var deployResponse dto.TektonDeployResponse
	if err := json.Unmarshal(responseBody, &deployResponse); err != nil {
		t.logger.Error(ctx, "tekton client response parsing failed",
			zap.String("response_body", string(responseBody)),
			zap.Error(err),
		)
		return nil, projecterrors.ErrInvalidTektonResponse
	}

	t.logger.Info(ctx, "tekton client trigger deploy completed",
		zap.String("service_name", request.DeploymentConfigJSON.ServiceName),
		zap.String("event_id", deployResponse.EventID),
	)

	return &deployResponse, nil
}

// Compile-time assertion that tektonClient implements TektonClient interface
var _ infrastructure.TektonClient = (*tektonClient)(nil)
