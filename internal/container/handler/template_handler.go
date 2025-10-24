package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/container/application"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"go.uber.org/zap"
)

type TemplateHandler struct {
	getTemplatesUseCase *application.GetTemplatesUseCase
	getTemplateUseCase  *application.GetTemplateUseCase
	logger              logger.Logger
}

func NewTemplateHandler(
	getTemplatesUseCase *application.GetTemplatesUseCase,
	getTemplateUseCase *application.GetTemplateUseCase,
	log logger.Logger,
) *TemplateHandler {
	return &TemplateHandler{
		getTemplatesUseCase: getTemplatesUseCase,
		getTemplateUseCase:  getTemplateUseCase,
		logger:              log,
	}
}

// GetTemplates handles GET /api/v1/templates
// Retrieves all active templates
func (h *TemplateHandler) GetTemplates(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "get templates handler started",
		zap.String("handler", "GetTemplates"),
	)

	output, err := h.getTemplatesUseCase.Execute(c.Request.Context())
	if err != nil {
		h.logger.Error(ctx, "get templates use case failed",
			zap.Error(err),
			zap.String("handler", "GetTemplates"),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "get templates handler completed",
		zap.String("handler", "GetTemplates"),
		zap.Int("template_count", len(output.Templates)),
	)

	response.OK(c, output)
}

// GetTemplate handles GET /api/v1/templates/:id
// Retrieves a specific template by ID
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	ctx := c.Request.Context()
	templateIDStr := c.Param("id")

	h.logger.Info(ctx, "get template handler started",
		zap.String("handler", "GetTemplate"),
		zap.String("template_id_str", templateIDStr),
	)

	// Parse template ID from URL
	templateID, err := strconv.ParseUint(templateIDStr, 10, 32)
	if err != nil {
		h.logger.Warn(ctx, "invalid template id parameter",
			zap.Error(err),
			zap.String("handler", "GetTemplate"),
			zap.String("template_id_str", templateIDStr),
		)
		response.Error(c, containererrors.ErrInvalidTemplateID, mapContainerError)
		return
	}

	// Execute use case
	input := application.GetTemplateInput{
		TemplateID: uint(templateID),
	}

	output, err := h.getTemplateUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		h.logger.Error(ctx, "get template use case failed",
			zap.Error(err),
			zap.String("handler", "GetTemplate"),
			zap.Uint("template_id", uint(templateID)),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	h.logger.Info(ctx, "get template handler completed",
		zap.String("handler", "GetTemplate"),
		zap.Uint("template_id", uint(templateID)),
		zap.String("template_name", output.Name),
	)

	response.OK(c, output)
}
