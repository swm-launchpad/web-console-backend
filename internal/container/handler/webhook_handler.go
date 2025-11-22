package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/container/application"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	containerservice "github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
	"go.uber.org/zap"
)

// WebhookHandler handles webhook-related HTTP requests
type WebhookHandler struct {
	processWebhookUC         *application.ProcessWebhookUseCase
	enableWebhookUC          *application.EnableWebhookUseCase
	disableWebhookUC         *application.DisableWebhookUseCase
	regenerateWebhookTokenUC *application.RegenerateWebhookTokenUseCase
	containerService         containerservice.ContainerService
	logger                   logger.Logger
}

// NewWebhookHandler creates a new WebhookHandler
func NewWebhookHandler(
	processWebhookUC *application.ProcessWebhookUseCase,
	enableWebhookUC *application.EnableWebhookUseCase,
	disableWebhookUC *application.DisableWebhookUseCase,
	regenerateWebhookTokenUC *application.RegenerateWebhookTokenUseCase,
	containerService containerservice.ContainerService,
	log logger.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		processWebhookUC:         processWebhookUC,
		enableWebhookUC:          enableWebhookUC,
		disableWebhookUC:         disableWebhookUC,
		regenerateWebhookTokenUC: regenerateWebhookTokenUC,
		containerService:         containerService,
		logger:                   log,
	}
}

// ProcessWebhook godoc
// @Summary Process incoming webhook
// @Description Processes webhook call and triggers project deployment
// @Tags webhooks
// @Accept json
// @Produce json
// @Param token path string true "Webhook Token"
// @Success 200 {object} response.Response{data=ProcessWebhookResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /webhooks/{token} [post]
// @Router /webhooks/{token} [get]
func (h *WebhookHandler) ProcessWebhook(c *gin.Context) {
	token := c.Param("token")

	h.logger.Info(c.Request.Context(), "processing webhook request",
		zap.String("token_prefix", token[:8]+"..."),
		zap.String("method", c.Request.Method),
		zap.String("source_ip", c.ClientIP()),
		zap.String("user_agent", c.Request.UserAgent()),
	)

	// Parse request body if POST
	var payload interface{}
	if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&payload); err != nil {
			// Body is optional, so we ignore parsing errors
			h.logger.Debug(c.Request.Context(), "webhook payload parsing failed (optional)",
				zap.Error(err),
			)
		}
	}

	// Process webhook
	sourceIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	req := application.ProcessWebhookRequest{
		WebhookToken: token,
		HTTPMethod:   c.Request.Method,
		SourceIP:     &sourceIP,
		UserAgent:    &userAgent,
	}

	result, err := h.processWebhookUC.Execute(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "webhook processing failed",
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(c.Request.Context(), "webhook processed successfully",
		zap.Uint32("project_id", result.ProjectID),
		zap.Uint32("container_id", result.ContainerID),
		zap.String("container_slug", result.ContainerSlug),
	)

	// TODO: Trigger project deployment here
	// This will be integrated with the deployment system later

	response.OK(c, ProcessWebhookResponse{
		ProjectID:     result.ProjectID,
		ContainerID:   result.ContainerID,
		ContainerSlug: result.ContainerSlug,
		ContainerName: result.ContainerName,
		Message:       "Webhook received successfully. Deployment will be triggered.",
	})
}

// ProcessWebhookResponse represents the webhook processing response
type ProcessWebhookResponse struct {
	ProjectID     uint32 `json:"project_id"`
	ContainerID   uint32 `json:"container_id"`
	ContainerSlug string `json:"container_slug"`
	ContainerName string `json:"container_name"`
	Message       string `json:"message"`
}

// EnableWebhook godoc
// @Summary Enable webhook for container
// @Description Enables webhook and generates a webhook token for the container
// @Tags containers,webhooks
// @Accept json
// @Produce json
// @Param slug path string true "Container Slug"
// @Success 200 {object} response.Response{data=EnableWebhookResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/containers/{slug}/webhook/enable [post]
func (h *WebhookHandler) EnableWebhook(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.containerService.GetContainerBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(c.Request.Context(), "failed to get container by slug",
			zap.Error(err),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// TODO: Get user ID from JWT token
	// For now, using placeholder
	userID := uint32(1)

	req := application.EnableWebhookRequest{
		ContainerID: uint32(container.ContainerID()),
		UserID:      userID,
	}

	result, err := h.enableWebhookUC.Execute(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "enable webhook failed",
			zap.Uint("container_id", container.ContainerID()),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(c.Request.Context(), "webhook enabled successfully",
		zap.Uint("container_id", container.ContainerID()),
	)

	response.OK(c, EnableWebhookResponse{
		WebhookToken: result.WebhookToken,
		WebhookURL:   buildWebhookURL(c.Request, result.WebhookToken),
	})
}

// EnableWebhookResponse represents the enable webhook response
type EnableWebhookResponse struct {
	WebhookToken string `json:"webhook_token"`
	WebhookURL   string `json:"webhook_url"`
}

// DisableWebhook godoc
// @Summary Disable webhook for container
// @Description Disables webhook for the container
// @Tags containers,webhooks
// @Accept json
// @Produce json
// @Param slug path string true "Container Slug"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/containers/{slug}/webhook/disable [post]
func (h *WebhookHandler) DisableWebhook(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.containerService.GetContainerBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(c.Request.Context(), "failed to get container by slug",
			zap.Error(err),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// TODO: Get user ID from JWT token
	// For now, using placeholder
	userID := uint32(1)

	req := application.DisableWebhookRequest{
		ContainerID: uint32(container.ContainerID()),
		UserID:      userID,
	}

	err = h.disableWebhookUC.Execute(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "disable webhook failed",
			zap.Uint("container_id", container.ContainerID()),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(c.Request.Context(), "webhook disabled successfully",
		zap.Uint("container_id", container.ContainerID()),
	)

	response.OK(c, gin.H{
		"message": "Webhook disabled successfully",
	})
}

// RegenerateWebhookToken godoc
// @Summary Regenerate webhook token
// @Description Generates a new webhook token for the container
// @Tags containers,webhooks
// @Accept json
// @Produce json
// @Param slug path string true "Container Slug"
// @Success 200 {object} response.Response{data=RegenerateWebhookTokenResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/containers/{slug}/webhook/regenerate [post]
func (h *WebhookHandler) RegenerateWebhookToken(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		response.Error(c, containererrors.ErrMissingField, mapContainerError)
		return
	}

	// Get container by slug
	container, err := h.containerService.GetContainerBySlug(c.Request.Context(), slug)
	if err != nil {
		h.logger.Error(c.Request.Context(), "failed to get container by slug",
			zap.Error(err),
			zap.String("slug", slug),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// TODO: Get user ID from JWT token
	// For now, using placeholder
	userID := uint32(1)

	req := application.RegenerateWebhookTokenRequest{
		ContainerID: uint32(container.ContainerID()),
		UserID:      userID,
	}

	result, err := h.regenerateWebhookTokenUC.Execute(c.Request.Context(), req)
	if err != nil {
		h.logger.Error(c.Request.Context(), "regenerate webhook token failed",
			zap.Uint("container_id", container.ContainerID()),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(c.Request.Context(), "webhook token regenerated successfully",
		zap.Uint("container_id", container.ContainerID()),
	)

	response.OK(c, RegenerateWebhookTokenResponse{
		WebhookToken: result.WebhookToken,
		WebhookURL:   buildWebhookURL(c.Request, result.WebhookToken),
	})
}

// RegenerateWebhookTokenResponse represents the regenerate token response
type RegenerateWebhookTokenResponse struct {
	WebhookToken string `json:"webhook_token"`
	WebhookURL   string `json:"webhook_url"`
}

// buildWebhookURL constructs the full webhook URL
func buildWebhookURL(req *http.Request, token string) string {
	scheme := "https"
	if req.TLS == nil {
		scheme = "http"
	}
	host := req.Host
	return scheme + "://" + host + "/api/v1/webhooks/" + token
}
