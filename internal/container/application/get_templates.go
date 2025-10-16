package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
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
}

func NewGetTemplatesUseCase(templateRepo repository.TemplateRepository) *GetTemplatesUseCase {
	return &GetTemplatesUseCase{
		templateRepo: templateRepo,
	}
}

func (uc *GetTemplatesUseCase) Execute(ctx context.Context) (*GetTemplatesOutput, error) {
	// Get only active templates
	templates, err := uc.templateRepo.FindActiveTemplates(ctx)
	if err != nil {
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

	return &GetTemplatesOutput{
		Templates: items,
	}, nil
}
