package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"go.uber.org/zap"
)

type GetTemplatesOutput struct {
	Templates []TemplateListItem `json:"templates"`
}

type TemplateListItem struct {
	TemplateID   uint     `json:"template_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Categories   []string `json:"categories"`
	DisplayOrder int      `json:"display_order"`
	IconName     string   `json:"icon_name,omitempty"`
	Status       string   `json:"status"`
	RequiresGit  bool     `json:"requires_git"`
	Version      string   `json:"version,omitempty"`
}

type GetTemplatesUseCase struct {
	templateRepo repository.TemplateRepository
	logger       logger.Logger
}

func NewGetTemplatesUseCase(templateRepo repository.TemplateRepository, log logger.Logger) *GetTemplatesUseCase {
	return &GetTemplatesUseCase{
		templateRepo: templateRepo,
		logger:       log,
	}
}

func (uc *GetTemplatesUseCase) Execute(ctx context.Context) (*GetTemplatesOutput, error) {
	uc.logger.Info(ctx, "get templates started")

	// Get only active templates
	templates, err := uc.templateRepo.FindActiveTemplates(ctx)
	if err != nil {
		uc.logger.Error(ctx, "failed to find active templates",
			zap.Error(err),
		)
		return nil, err
	}

	// Convert to output format
	items := make([]TemplateListItem, 0, len(templates))
	for _, template := range templates {
		item := TemplateListItem{
			TemplateID: template.TemplateID(),
			Name:       template.Name(),
			Status:     template.Status().String(),
		}

		// Extract configuration fields if available
		if config := template.TemplateConfig(); config != nil {
			item.Description = config.GetDescription()
			item.Categories = config.GetCategories()
			item.DisplayOrder = config.GetDisplayOrder()
			item.IconName = config.GetIconName()
			item.RequiresGit = config.GetRequiresGit()
			item.Version = config.GetVersion()
		}

		items = append(items, item)
	}

	uc.logger.Info(ctx, "get templates completed",
		zap.Int("count", len(items)),
	)

	return &GetTemplatesOutput{
		Templates: items,
	}, nil
}
