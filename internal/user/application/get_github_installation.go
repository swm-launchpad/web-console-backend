package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

type GetGitHubInstallationInput struct {
	UserID uint
}

type GitHubInstallationOutput struct {
	InstallationID int64      `json:"installation_id"`
	AccountLogin   string     `json:"account_login"`
	AccountType    string     `json:"account_type"`
	HasValidToken  bool       `json:"has_valid_token"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type GetGitHubInstallationOutput struct {
	Installations []GitHubInstallationOutput `json:"installations"`
}

type GetGitHubInstallationUseCase struct {
	installationRepo repository.GitHubInstallationRepository
}

func NewGetGitHubInstallationUseCase(
	installationRepo repository.GitHubInstallationRepository,
) *GetGitHubInstallationUseCase {
	return &GetGitHubInstallationUseCase{
		installationRepo: installationRepo,
	}
}

func (uc *GetGitHubInstallationUseCase) Execute(ctx context.Context, input GetGitHubInstallationInput) (*GetGitHubInstallationOutput, error) {
	// Get all installations for the user
	installations, err := uc.installationRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	// Map to output
	outputs := make([]GitHubInstallationOutput, 0, len(installations))
	for _, installation := range installations {
		output := GitHubInstallationOutput{
			InstallationID: installation.InstallationID,
			AccountLogin:   installation.AccountLogin,
			AccountType:    string(installation.AccountType),
			HasValidToken:  installation.IsTokenValid(5), // 5 minute buffer
			CreatedAt:      installation.CreatedAt,
		}

		if installation.TokenExpiresAt != nil {
			output.TokenExpiresAt = installation.TokenExpiresAt
		}

		outputs = append(outputs, output)
	}

	return &GetGitHubInstallationOutput{
		Installations: outputs,
	}, nil
}
