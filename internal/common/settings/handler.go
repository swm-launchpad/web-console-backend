package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
)

// SettingsHandler handles HTTP requests for settings
type SettingsHandler struct {
	settingsService SettingsService
	logger          logger.Logger
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsService SettingsService, logger logger.Logger) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
		logger:          logger,
	}
}

// PublicSettingsResponse represents the public settings response
type PublicSettingsResponse struct {
	Pricing interface{} `json:"pricing"`
	Limits  interface{} `json:"limits"`
	Beta    interface{} `json:"beta"`
}

// GetPublicSettings returns all public settings (no authentication required)
// @Summary Get public settings
// @Description Get all public system settings including pricing and limits
// @Tags settings
// @Accept json
// @Produce json
// @Success 200 {object} PublicSettingsResponse
// @Failure 500 {object} response.ErrorResponse
// @Router /api/v1/settings/public [get]
func (h *SettingsHandler) GetPublicSettings(c *gin.Context) {
	ctx := c.Request.Context()

	// Get all settings by category
	pricingSettings, err := h.settingsService.GetByCategory("pricing")
	if err != nil {
		h.logger.Error(ctx, "Failed to get pricing settings")
		response.Error(c, err, nil)
		return
	}

	limitsSettings, err := h.settingsService.GetByCategory("limits")
	if err != nil {
		h.logger.Error(ctx, "Failed to get limits settings")
		response.Error(c, err, nil)
		return
	}

	betaSettings, err := h.settingsService.GetByCategory("beta")
	if err != nil {
		h.logger.Error(ctx, "Failed to get beta settings")
		response.Error(c, err, nil)
		return
	}

	// Convert settings to map format for easy frontend consumption
	pricingMap := make(map[string]string)
	for _, s := range pricingSettings {
		pricingMap[s.Key] = s.Value
	}

	limitsMap := make(map[string]string)
	for _, s := range limitsSettings {
		limitsMap[s.Key] = s.Value
	}

	betaMap := make(map[string]string)
	for _, s := range betaSettings {
		betaMap[s.Key] = s.Value
	}

	resp := PublicSettingsResponse{
		Pricing: pricingMap,
		Limits:  limitsMap,
		Beta:    betaMap,
	}

	c.JSON(http.StatusOK, resp)
}
