package github

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	githubAPIBaseURL = "https://api.github.com"
	jwtExpiration    = 10 * time.Minute // GitHub recommends max 10 minutes
	tokenExpiration  = 1 * time.Hour    // Installation tokens expire in 1 hour
)

// InstallationToken represents a GitHub App installation access token
type InstallationToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// InstallationInfo represents basic information about a GitHub App installation
type InstallationInfo struct {
	ID      int64 `json:"id"`
	Account struct {
		Login string `json:"login"`
		Type  string `json:"type"` // "User" or "Organization"
	} `json:"account"`
}

// Client handles GitHub App authentication and API interactions
type Client struct {
	appID      int64
	privateKey *rsa.PrivateKey
	httpClient *http.Client
}

// NewClient creates a new GitHub App client
func NewClient(appID string, privateKeyPath string) (*Client, error) {
	if appID == "" {
		return nil, ErrMissingAppID
	}
	if privateKeyPath == "" {
		return nil, ErrMissingPrivateKey
	}

	// Parse App ID
	appIDInt, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID: %w", err)
	}

	// Load and parse private key
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return nil, err
	}

	return &Client{
		appID:      appIDInt,
		privateKey: privateKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// GenerateJWT creates a JWT for GitHub App authentication
func (c *Client) GenerateJWT() (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(jwtExpiration).Unix(),
		"iss": c.appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailedToGenerateJWT, err)
	}

	return signedToken, nil
}

// CreateInstallationToken generates an access token for a specific installation
func (c *Client) CreateInstallationToken(installationID int64) (*InstallationToken, error) {
	if installationID <= 0 {
		return nil, ErrInvalidInstallationID
	}

	// Generate JWT for authentication
	jwtToken, err := c.GenerateJWT()
	if err != nil {
		return nil, err
	}

	// Create installation token request
	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", githubAPIBaseURL, installationID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToCreateToken, err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFailedToCreateToken, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response: %v", ErrFailedToCreateToken, err)
	}

	if resp.StatusCode != http.StatusCreated {
		// Handle specific error cases
		switch resp.StatusCode {
		case http.StatusNotFound:
			// Installation not found - app may have been uninstalled
			return nil, fmt.Errorf("%w: status %d, body: %s", ErrInstallationNotFound, resp.StatusCode, string(body))
		case http.StatusForbidden:
			// Access forbidden - insufficient permissions
			return nil, fmt.Errorf("%w: status %d, body: %s", ErrInstallationUnauthorized, resp.StatusCode, string(body))
		default:
			// Generic error for other status codes
			return nil, fmt.Errorf("%w: status %d, body: %s", ErrFailedToCreateToken, resp.StatusCode, string(body))
		}
	}

	// Parse response
	var tokenResponse struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("%w: failed to parse response: %v", ErrFailedToCreateToken, err)
	}

	return &InstallationToken{
		Token:     tokenResponse.Token,
		ExpiresAt: tokenResponse.ExpiresAt,
	}, nil
}

// GetInstallationInfo retrieves information about a specific installation
func (c *Client) GetInstallationInfo(installationID int64) (*InstallationInfo, error) {
	if installationID <= 0 {
		return nil, ErrInvalidInstallationID
	}

	// Generate JWT for authentication
	jwtToken, err := c.GenerateJWT()
	if err != nil {
		return nil, err
	}

	// Get installation info
	url := fmt.Sprintf("%s/app/installations/%d", githubAPIBaseURL, installationID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get installation info: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var info InstallationInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse installation info: %w", err)
	}

	return &info, nil
}

// ExchangeCodeForInstallation exchanges an OAuth code for installation details
// Returns all installation IDs accessible by the user
func (c *Client) ExchangeCodeForInstallation(code, clientID, clientSecret string) ([]int64, error) {
	// Exchange code for access token
	url := "https://github.com/login/oauth/access_token"
	payload := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("OAuth error: %s", tokenResp.Error)
	}

	// Get user's installations using the access token
	installationsURL := githubAPIBaseURL + "/user/installations"
	req, err = http.NewRequest("GET", installationsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create installations request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get installations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var installationsResp struct {
		Installations []InstallationInfo `json:"installations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&installationsResp); err != nil {
		return nil, fmt.Errorf("failed to decode installations: %w", err)
	}

	if len(installationsResp.Installations) == 0 {
		return nil, fmt.Errorf("no installations found")
	}

	// Return all installation IDs
	installationIDs := make([]int64, len(installationsResp.Installations))
	for i, installation := range installationsResp.Installations {
		installationIDs[i] = installation.ID
	}

	return installationIDs, nil
}

// Repository represents a GitHub repository
type Repository struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	HTMLURL  string `json:"html_url"`
	CloneURL string `json:"clone_url"`
}

// Branch represents a GitHub repository branch
type Branch struct {
	Name      string `json:"name"`
	Protected bool   `json:"protected"`
	Commit    struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	} `json:"commit"`
}

// ListRepositories lists all repositories accessible by the installation
func (c *Client) ListRepositories(installationID int64) ([]Repository, error) {
	// Create installation access token
	token, err := c.CreateInstallationToken(installationID)
	if err != nil {
		return nil, err
	}

	// Request to list repositories with per_page parameter (max 100)
	url := fmt.Sprintf("%s/installation/repositories?per_page=100", githubAPIBaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Handle specific error cases
		switch resp.StatusCode {
		case http.StatusNotFound:
			// Installation not found - app may have been uninstalled
			return nil, fmt.Errorf("%w: status %d, body: %s", ErrInstallationNotFound, resp.StatusCode, string(body))
		case http.StatusForbidden, http.StatusUnauthorized:
			// Access forbidden - insufficient permissions
			return nil, fmt.Errorf("%w: status %d, body: %s", ErrInstallationUnauthorized, resp.StatusCode, string(body))
		default:
			// Generic error for other status codes
			return nil, fmt.Errorf("failed to list repositories: status %d, body: %s", resp.StatusCode, string(body))
		}
	}

	var repoResp struct {
		Repositories []Repository `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return repoResp.Repositories, nil
}

// ListBranches lists all branches for a repository
func (c *Client) ListBranches(installationID int64, owner, repo string) ([]Branch, error) {
	// Create installation access token
	token, err := c.CreateInstallationToken(installationID)
	if err != nil {
		return nil, err
	}

	// Request to list branches with per_page parameter (max 100)
	url := fmt.Sprintf("%s/repos/%s/%s/branches?per_page=100", githubAPIBaseURL, owner, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Handle specific error cases
		switch resp.StatusCode {
		case http.StatusNotFound:
			// 404 means repository not found, owner/repo typo, or repo deleted
			// This is a repository-level error, not an installation-level error
			return nil, fmt.Errorf("%w: status %d, body: %s", ErrRepositoryNotFound, resp.StatusCode, string(body))
		case http.StatusForbidden:
			// 403 means repository exists but not accessible (not granted to installation)
			// This is a repository-level permission error
			return nil, fmt.Errorf("%w: status %d, body: %s", ErrRepositoryAccessDenied, resp.StatusCode, string(body))
		case http.StatusUnauthorized:
			// 401 means installation token is invalid/revoked
			// This is an installation-level error
			return nil, fmt.Errorf("%w: status %d, body: %s", ErrInstallationUnauthorized, resp.StatusCode, string(body))
		default:
			// Generic error for other status codes
			return nil, fmt.Errorf("failed to list branches: status %d, body: %s", resp.StatusCode, string(body))
		}
	}

	var branches []Branch
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return branches, nil
}

// IsTokenExpired checks if a token has expired or will expire within the buffer time
func IsTokenExpired(expiresAt time.Time, bufferMinutes int) bool {
	buffer := time.Duration(bufferMinutes) * time.Minute
	return time.Now().Add(buffer).After(expiresAt)
}

// loadPrivateKey loads and parses the RSA private key from a PEM file
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read key file: %v", ErrInvalidPrivateKey, err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("%w: failed to decode PEM block", ErrInvalidPrivateKey)
	}

	// Try PKCS1 format first
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	// Try PKCS8 format
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: key is not RSA format", ErrInvalidPrivateKey)
	}

	return rsaKey, nil
}
