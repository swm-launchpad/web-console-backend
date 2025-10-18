package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/github"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
)

type OAuthCallbackInput struct {
	Code  string
	State string
}

type InstallationResult struct {
	InstallationID int64     `json:"installation_id"`
	AccountLogin   string    `json:"account_login"`
	AccountType    string    `json:"account_type"`
	IsNew          bool      `json:"is_new"`
	CreatedAt      time.Time `json:"created_at"`
}

type OAuthCallbackOutput struct {
	UserID        uint                 `json:"user_id"`
	Installations []InstallationResult `json:"installations"`
}

type OAuthCallbackUseCase struct {
	config           *config.Config
	githubClient     *github.Client
	installationRepo repository.GitHubInstallationRepository
	txManager        db.TxManager
}

func NewOAuthCallbackUseCase(
	cfg *config.Config,
	githubClient *github.Client,
	installationRepo repository.GitHubInstallationRepository,
	txManager db.TxManager,
) *OAuthCallbackUseCase {
	return &OAuthCallbackUseCase{
		config:           cfg,
		githubClient:     githubClient,
		installationRepo: installationRepo,
		txManager:        txManager,
	}
}

func (uc *OAuthCallbackUseCase) Execute(ctx context.Context, input OAuthCallbackInput) (*OAuthCallbackOutput, error) {
	// Validate state format and extract user ID
	// State format: "base64_random:user_id"
	parts := strings.Split(input.State, ":")
	if len(parts) != 2 {
		return nil, usererrors.ErrInvalidState
	}

	userID, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return nil, usererrors.ErrInvalidState
	}

	// Exchange code for installation IDs
	installationIDs, err := uc.githubClient.ExchangeCodeForInstallation(
		input.Code,
		uc.config.GitHubApp.ClientID,
		uc.config.GitHubApp.ClientSecret,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", usererrors.ErrGitHubOAuthFailed, err)
	}

	var results []InstallationResult

	// Process all installations
	err = uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		for _, installationID := range installationIDs {
			// Get installation info from GitHub API
			installationInfo, err := uc.githubClient.GetInstallationInfo(installationID)
			if err != nil {
				// Log error but continue with other installations
				continue
			}

			// Check if installation already exists
			exists, err := uc.installationRepo.ExistsByInstallationID(txCtx, installationID)
			if err != nil {
				return fmt.Errorf("failed to check installation existence: %w", err)
			}

			isNew := !exists
			var installation *model.GitHubInstallation

			if exists {
				// Update existing installation (reactivate if revoked)
				installation, err = uc.installationRepo.FindByInstallationID(txCtx, installationID)
				if err != nil {
					return fmt.Errorf("failed to find installation: %w", err)
				}

				// Reactivate if revoked
				if installation.Status == model.InstallationStatusRevoked {
					installation.Status = model.InstallationStatusActive
					now := time.Now()
					installation.UpdatedAt = &now

					if err := uc.installationRepo.Update(txCtx, installation); err != nil {
						return fmt.Errorf("failed to update installation: %w", err)
					}
				}
			} else {
				// Create new installation
				accountType := model.AccountTypeUser
				if installationInfo.Account.Type == "Organization" {
					accountType = model.AccountTypeOrganization
				}

				installation, err = model.NewGitHubInstallation(
					installationID,
					uint(userID),
					installationInfo.Account.Login,
					accountType,
				)
				if err != nil {
					return err
				}

				if err := uc.installationRepo.Create(txCtx, installation); err != nil {
					return fmt.Errorf("failed to create installation: %w", err)
				}
			}

			results = append(results, InstallationResult{
				InstallationID: installation.InstallationID,
				AccountLogin:   installation.AccountLogin,
				AccountType:    string(installation.AccountType),
				IsNew:          isNew,
				CreatedAt:      installation.CreatedAt,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &OAuthCallbackOutput{
		UserID:        uint(userID),
		Installations: results,
	}, nil
}
