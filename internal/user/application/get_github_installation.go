package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
	"go.uber.org/zap"
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
	logger           logger.Logger
}

func NewGetGitHubInstallationUseCase(
	installationRepo repository.GitHubInstallationRepository,
	log logger.Logger,
) *GetGitHubInstallationUseCase {
	return &GetGitHubInstallationUseCase{
		installationRepo: installationRepo,
		logger:           log,
	}
}

func (uc *GetGitHubInstallationUseCase) Execute(ctx context.Context, input GetGitHubInstallationInput) (*GetGitHubInstallationOutput, error) {
	uc.logger.Info(ctx, "get github installation started",
		zap.Uint("user_id", input.UserID),
	)

	// Get all installations for the user
	installations, err := uc.installationRepo.FindByUserID(ctx, input.UserID)
	if err != nil {
		uc.logger.Error(ctx, "failed to find installations by user ID",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
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

	uc.logger.Info(ctx, "get github installation completed",
		zap.Uint("user_id", input.UserID),
		zap.Int("installation_count", len(outputs)),
	)

	return &GetGitHubInstallationOutput{
		Installations: outputs,
	}, nil
}
