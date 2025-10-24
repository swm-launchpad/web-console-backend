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

	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/dto"
)

// tektonBuildClient implements the TektonBuildClient interface using HTTP requests.
// It communicates with the Tekton EventListener to trigger container image builds.
type tektonBuildClient struct {
	buildURL   string
	authHeader string
	httpClient *http.Client
}

// NewTektonBuildClient creates a new Tekton build client using configuration from environment variables.
//
// Required environment variables:
//   - TEKTON_BUILD_URL: The Tekton EventListener endpoint URL (e.g., "https://tekton-api.launchpad.kr/build")
//   - TEKTON_API_AUTH: The Basic authentication header value (e.g., "Basic base64encodedcredentials")
//
// Returns an error if any required environment variable is missing.
func NewTektonBuildClient() (infrastructure.TektonBuildClient, error) {
	buildURL := os.Getenv("TEKTON_BUILD_URL")
	if buildURL == "" {
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

	return &tektonBuildClient{
		buildURL:   buildURL,
		authHeader: authHeader,
		httpClient: httpClient,
	}, nil
}

// TriggerBuild sends a build request to the Tekton EventListener.
// It returns the response from Tekton on success (HTTP 202 Accepted).
func (t *tektonBuildClient) TriggerBuild(ctx context.Context, request *dto.TektonBuildRequest) (*dto.TektonBuildResponse, error) {
	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		// Request marshaling failed - this is a client-side validation error,
		// not a Tekton API response error
		return nil, projecterrors.ErrInvalidBuildRequest
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.buildURL, bytes.NewBuffer(requestBody))
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
			// 400 Bad Request: build configuration validation failed
			return nil, fmt.Errorf("%w: %s", projecterrors.ErrTektonBuildFailed, errorMsg)
		case http.StatusUnauthorized, http.StatusForbidden:
			// 401/403: authentication/authorization failure - infrastructure issue
			// This indicates wrong credentials or ACL changes, not a user error
			// Must be treated as infrastructure failure to trigger ops alerts
			return nil, fmt.Errorf("%w: %s", projecterrors.ErrTektonAuthenticationFailed, errorMsg)
		default:
			// Server errors or other issues
			return nil, fmt.Errorf("%w: %s", projecterrors.ErrTektonUnavailable, errorMsg)
		}
	}

	// Parse response
	var buildResponse dto.TektonBuildResponse
	if err := json.Unmarshal(responseBody, &buildResponse); err != nil {
		return nil, projecterrors.ErrInvalidTektonResponse
	}

	return &buildResponse, nil
}

// Compile-time assertion that tektonBuildClient implements TektonBuildClient interface
var _ infrastructure.TektonBuildClient = (*tektonBuildClient)(nil)
