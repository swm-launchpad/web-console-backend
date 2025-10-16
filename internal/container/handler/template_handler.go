package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/container/application"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

type TemplateHandler struct {
	getTemplatesUseCase *application.GetTemplatesUseCase
	getTemplateUseCase  *application.GetTemplateUseCase
}

func NewTemplateHandler(
	getTemplatesUseCase *application.GetTemplatesUseCase,
	getTemplateUseCase *application.GetTemplateUseCase,
) *TemplateHandler {
	return &TemplateHandler{
		getTemplatesUseCase: getTemplatesUseCase,
		getTemplateUseCase:  getTemplateUseCase,
	}
}

// GetTemplates handles GET /api/v1/templates
// Retrieves all active templates
func (h *TemplateHandler) GetTemplates(c *gin.Context) {
	output, err := h.getTemplatesUseCase.Execute(c.Request.Context())
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, output)
}

// GetTemplate handles GET /api/v1/templates/:id
// Retrieves a specific template by ID
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	// Parse template ID from URL
	templateIDStr := c.Param("id")
	templateID, err := strconv.ParseUint(templateIDStr, 10, 32)
	if err != nil {
		response.Error(c, containererrors.ErrInvalidTemplateID, mapContainerError)
		return
	}

	// Execute use case
	input := application.GetTemplateInput{
		TemplateID: uint(templateID),
	}

	output, err := h.getTemplateUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapContainerError)
		return
	}

	response.OK(c, output)
}
