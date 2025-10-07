package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// tektonClient implements the TektonClient interface using HTTP requests.
// It communicates with the Tekton EventListener to trigger deployments.
type tektonClient struct {
	deployURL  string
	authHeader string
	httpClient *http.Client
}

// NewTektonClient creates a new Tekton client using configuration from environment variables.
//
// Required environment variables:
//   - TEKTON_DEPLOY_URL: The Tekton EventListener endpoint URL (e.g., "https://tekton-api.launchpad.kr/deploy")
//   - TEKTON_API_AUTH: The Basic authentication header value (e.g., "Basic base64encodedcredentials")
//
// Returns an error if any required environment variable is missing.
func NewTektonClient() (infrastructure.TektonClient, error) {
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
	}, nil
}

// TriggerDeploy sends a deployment request to the Tekton EventListener.
// It returns the response from Tekton on success (HTTP 202 Accepted).
func (t *tektonClient) TriggerDeploy(ctx context.Context, request *dto.TektonDeployRequest) (*dto.TektonDeployResponse, error) {
	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		// Request marshaling failed - this is a client-side validation error,
		// not a Tekton API response error
		return nil, projecterrors.ErrInvalidDeploymentRequest
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.deployURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, projecterrors.ErrTektonUnavailable
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", t.authHeader)

	// Send request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, projecterrors.ErrTektonUnavailable
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log close error but don't override the main error
			_ = closeErr
		}
	}()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, projecterrors.ErrTektonUnavailable
	}

	// Check status code
	// Tekton EventListener returns 202 Accepted on success
	if resp.StatusCode != http.StatusAccepted {
		// Map different status codes to appropriate errors
		switch resp.StatusCode {
		case http.StatusBadRequest:
			// 400 Bad Request: deployment configuration validation failed
			return nil, projecterrors.ErrTektonDeploymentFailed
		case http.StatusUnauthorized, http.StatusForbidden:
			// 401/403: authentication/authorization failure - infrastructure issue
			// This indicates wrong credentials or ACL changes, not a user error
			// Must be treated as infrastructure failure to trigger ops alerts
			return nil, projecterrors.ErrTektonAuthenticationFailed
		default:
			// Server errors or other issues
			return nil, projecterrors.ErrTektonUnavailable
		}
	}

	// Parse response
	var deployResponse dto.TektonDeployResponse
	if err := json.Unmarshal(responseBody, &deployResponse); err != nil {
		return nil, projecterrors.ErrInvalidTektonResponse
	}

	return &deployResponse, nil
}

// Compile-time assertion that tektonClient implements TektonClient interface
var _ infrastructure.TektonClient = (*tektonClient)(nil)
