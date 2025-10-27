package build

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/container/domain/infrastructure/repository"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

type GetContainersForBuildInput struct {
	ProjectID uint
}

type BuildContainerOutput struct {
	ContainerID         uint                   `json:"container_id"`
	Name                string                 `json:"name"`
	Slug                string                 `json:"slug"`
	TemplateID          *uint                  `json:"template_id,omitempty"`
	TemplateBody        *string                `json:"template_body,omitempty"`
	TemplateConfig      map[string]interface{} `json:"template_config"`
	GitRepositoryURL    string                 `json:"git_repository_url"`
	GitBranch           string                 `json:"git_branch"`
	GitDirectoryPath    *string                `json:"git_directory_path,omitempty"`
	LastBuiltCommitHash *string                `json:"last_built_commit_hash,omitempty"`
	NeedsBuild          bool                   `json:"needs_build"`
	BuildVars           map[string]string      `json:"build_vars"`
	InstallationID      *int64                 `json:"installation_id,omitempty"`
}

type GetContainersForBuildOutput struct {
	Containers []BuildContainerOutput `json:"containers"`
}

type GetContainersForBuildUseCase struct {
	containerService   service.ContainerService
	templateRepository repository.TemplateRepository
}

func NewGetContainersForBuildUseCase(
	containerService service.ContainerService,
	templateRepository repository.TemplateRepository,
) *GetContainersForBuildUseCase {
	return &GetContainersForBuildUseCase{
		containerService:   containerService,
		templateRepository: templateRepository,
	}
}

func (uc *GetContainersForBuildUseCase) Execute(ctx context.Context, input GetContainersForBuildInput) (*GetContainersForBuildOutput, error) {
	// Get all containers for the project via service
	containers, err := uc.containerService.ListContainersByProjectID(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	// Build output
	output := &GetContainersForBuildOutput{
		Containers: make([]BuildContainerOutput, 0, len(containers)),
	}

	for _, container := range containers {
		// Build build vars map
		buildVars := make(map[string]string)
		for _, bv := range container.BuildVars() {
			buildVars[bv.Key()] = bv.Value()
		}

		// Get template body if template is configured
		var templateBody *string
		if container.TemplateID() != nil {
			template, err := uc.templateRepository.FindByID(ctx, *container.TemplateID())
			if err != nil {
				// Template ID is set but template not found - this indicates data inconsistency
				// or infrastructure failure. Return error instead of silently continuing.
				return nil, err
			}
			templateBody = template.TemplateBody()
		}

		containerOutput := BuildContainerOutput{
			ContainerID:         container.ContainerID(),
			Name:                container.Name(),
			Slug:                container.Slug().String(),
			TemplateID:          container.TemplateID(),
			TemplateBody:        templateBody,
			TemplateConfig:      container.TemplateConfig(),
			GitRepositoryURL:    container.GitConfig().RepositoryURL(),
			GitBranch:           container.GitConfig().Branch(),
			GitDirectoryPath:    container.GitConfig().DirectoryPath(),
			LastBuiltCommitHash: container.LastBuiltGitCommitHash(),
			NeedsBuild:          container.NeedsBuild(),
			BuildVars:           buildVars,
			InstallationID:      container.GitHubInstallationID(),
		}

		output.Containers = append(output.Containers, containerOutput)
	}

	return output, nil
}
