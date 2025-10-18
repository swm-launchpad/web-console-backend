package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
)

type GitHubHandler struct {
	connectUseCase              *application.ConnectGitHubUseCase
	disconnectUseCase           *application.DisconnectGitHubUseCase
	getInstallationUseCase      *application.GetGitHubInstallationUseCase
	generateTokenUseCase        *application.GenerateInstallationTokenUseCase
	listRepositoriesUseCase     *application.ListRepositoriesUseCase
	startInstallationUseCase    *application.StartInstallationUseCase
	installationCallbackUseCase *application.InstallationCallbackUseCase
	frontendURL                 string
}

func NewGitHubHandler(
	connectUseCase *application.ConnectGitHubUseCase,
	disconnectUseCase *application.DisconnectGitHubUseCase,
	getInstallationUseCase *application.GetGitHubInstallationUseCase,
	generateTokenUseCase *application.GenerateInstallationTokenUseCase,
	listRepositoriesUseCase *application.ListRepositoriesUseCase,
	startInstallationUseCase *application.StartInstallationUseCase,
	installationCallbackUseCase *application.InstallationCallbackUseCase,
	frontendURL string,
) *GitHubHandler {
	return &GitHubHandler{
		connectUseCase:              connectUseCase,
		disconnectUseCase:           disconnectUseCase,
		getInstallationUseCase:      getInstallationUseCase,
		generateTokenUseCase:        generateTokenUseCase,
		listRepositoriesUseCase:     listRepositoriesUseCase,
		startInstallationUseCase:    startInstallationUseCase,
		installationCallbackUseCase: installationCallbackUseCase,
		frontendURL:                 frontendURL,
	}
}

// ConnectGitHubRequest represents the request body for connecting GitHub
type ConnectGitHubRequest struct {
	InstallationID int64 `json:"installation_id" binding:"required,min=1"`
}

// ConnectGitHub handles GitHub App installation connection
// POST /api/v1/github/connect
func (h *GitHubHandler) ConnectGitHub(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var req ConnectGitHubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, usererrors.ErrValidationFailed, mapUserError, response.WithDetails(map[string]interface{}{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	input := application.ConnectGitHubInput{
		UserID:         userID,
		InstallationID: req.InstallationID,
	}

	output, err := h.connectUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.Created(c, output)
}

// GetInstallations retrieves all GitHub installations for the current user
// GET /api/v1/github/installations
func (h *GitHubHandler) GetInstallations(c *gin.Context) {
	userID := getUserIDFromContext(c)

	input := application.GetGitHubInstallationInput{
		UserID: userID,
	}

	output, err := h.getInstallationUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.OK(c, output)
}

// DisconnectGitHub disconnects a GitHub installation
// DELETE /api/v1/github/installations/:installation_id
func (h *GitHubHandler) DisconnectGitHub(c *gin.Context) {
	userID := getUserIDFromContext(c)

	installationIDStr := c.Param("installation_id")
	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		response.Error(c, usererrors.ErrInvalidInstallationID, mapUserError)
		return
	}

	input := application.DisconnectGitHubInput{
		UserID:         userID,
		InstallationID: installationID,
	}

	if err := h.disconnectUseCase.Execute(c.Request.Context(), input); err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.OK(c, gin.H{
		"message": "GitHub installation disconnected successfully",
	})
}

// GenerateInstallationTokenRequest represents the request body for generating installation token
type GenerateInstallationTokenRequest struct {
	InstallationID int64 `json:"installation_id" binding:"required,min=1"`
}

// GenerateInstallationToken generates an access token for a GitHub installation
// POST /api/v1/github/token
func (h *GitHubHandler) GenerateInstallationToken(c *gin.Context) {
	userID := getUserIDFromContext(c)

	var req GenerateInstallationTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, usererrors.ErrValidationFailed, mapUserError, response.WithDetails(map[string]interface{}{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	input := application.GenerateInstallationTokenInput{
		UserID:         userID,
		InstallationID: req.InstallationID,
	}

	output, err := h.generateTokenUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.OK(c, output)
}

// ListRepositories lists all repositories accessible by the installation
// GET /api/v1/github/installations/:installation_id/repositories
func (h *GitHubHandler) ListRepositories(c *gin.Context) {
	userID := getUserIDFromContext(c)

	installationIDStr := c.Param("installation_id")
	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		response.Error(c, usererrors.ErrInvalidInstallationID, mapUserError)
		return
	}

	input := application.ListRepositoriesInput{
		UserID:         userID,
		InstallationID: installationID,
	}

	output, err := h.listRepositoriesUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.OK(c, output)
}

// StartInstallation initiates the GitHub App installation flow
// GET /api/v1/github/installation/start
func (h *GitHubHandler) StartInstallation(c *gin.Context) {
	userID := getUserIDFromContext(c)

	input := application.StartInstallationInput{
		UserID: userID,
	}

	output, err := h.startInstallationUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	response.OK(c, output)
}

// InstallationCallback handles the GitHub App installation callback
// GET /api/v1/github/installation/callback
func (h *GitHubHandler) InstallationCallback(c *gin.Context) {
	// Parse query parameters
	installationIDStr := c.Query("installation_id")
	setupAction := c.Query("setup_action")
	state := c.Query("state")

	// Validate required parameters
	if installationIDStr == "" || state == "" {
		c.Redirect(302, h.frontendURL+"/github/callback?error=missing_parameters&popup=true")
		return
	}

	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		c.Redirect(302, h.frontendURL+"/github/callback?error=invalid_installation_id&popup=true")
		return
	}

	input := application.InstallationCallbackInput{
		InstallationID: installationID,
		SetupAction:    setupAction,
		State:          state,
	}

	_, err = h.installationCallbackUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		_ = c.Error(fmt.Errorf("installation callback failed: %w", err))
		// Redirect to frontend with error
		c.Redirect(302, h.frontendURL+"/github/callback?error=installation_failed&popup=true")
		return
	}

	// Redirect to frontend callback page with success
	// For simplicity, we're counting 1 installation since we process one at a time
	// Add popup=true to indicate this was opened in a popup window
	c.Redirect(302, h.frontendURL+"/github/callback?success=true&count=1&popup=true")
}

// getUserIDFromContext extracts the user ID from the Gin context
// This assumes the auth middleware has already set the user_id in the context
func getUserIDFromContext(c *gin.Context) uint {
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		return 0
	}

	switch v := userID.(type) {
	case uint:
		return v
	case uint32:
		return uint(v)
	case int:
		return uint(v)
	case float64:
		return uint(v)
	default:
		return 0
	}
}
