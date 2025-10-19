package model

import (
	"time"

	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
)

type AccountType string

const (
	AccountTypeUser         AccountType = "User"
	AccountTypeOrganization AccountType = "Organization"
)

type InstallationStatus string

const (
	InstallationStatusActive  InstallationStatus = "active"
	InstallationStatusRevoked InstallationStatus = "revoked"
)

// GitHubInstallation represents a GitHub App installation for a user
type GitHubInstallation struct {
	InstallationID int64
	UserID         uint
	AccountLogin   string
	AccountType    AccountType
	Status         InstallationStatus
	CachedToken    *string
	TokenExpiresAt *time.Time
	IsDeleted      bool
	DeletedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      *time.Time
}

// NewGitHubInstallation creates a new GitHub installation
func NewGitHubInstallation(installationID int64, userID uint, accountLogin string, accountType AccountType) (*GitHubInstallation, error) {
	if installationID <= 0 {
		return nil, usererrors.ErrInvalidInstallationID
	}
	if userID == 0 {
		return nil, usererrors.ErrUserIDRequired
	}
	if accountLogin == "" {
		return nil, usererrors.ErrAccountLoginRequired
	}

	now := time.Now()
	return &GitHubInstallation{
		InstallationID: installationID,
		UserID:         userID,
		AccountLogin:   accountLogin,
		AccountType:    accountType,
		Status:         InstallationStatusActive,
		IsDeleted:      false,
		CreatedAt:      now,
		UpdatedAt:      &now,
	}, nil
}

// UpdateToken updates the cached token and expiration time
func (g *GitHubInstallation) UpdateToken(token string, expiresAt time.Time) {
	g.CachedToken = &token
	g.TokenExpiresAt = &expiresAt
	now := time.Now()
	g.UpdatedAt = &now
}

// ClearToken removes the cached token
func (g *GitHubInstallation) ClearToken() {
	g.CachedToken = nil
	g.TokenExpiresAt = nil
	now := time.Now()
	g.UpdatedAt = &now
}

// IsTokenValid checks if the cached token is valid and not expired
func (g *GitHubInstallation) IsTokenValid(bufferMinutes int) bool {
	if g.CachedToken == nil || g.TokenExpiresAt == nil {
		return false
	}

	buffer := time.Duration(bufferMinutes) * time.Minute
	return time.Now().Add(buffer).Before(*g.TokenExpiresAt)
}

// SoftDelete marks the installation as deleted
func (g *GitHubInstallation) SoftDelete() {
	now := time.Now()
	g.IsDeleted = true
	g.DeletedAt = &now
	g.UpdatedAt = &now
	// Clear token on deletion
	g.CachedToken = nil
	g.TokenExpiresAt = nil
}

// MarkAsRevoked marks the installation as revoked (app uninstalled on GitHub)
func (g *GitHubInstallation) MarkAsRevoked() {
	now := time.Now()
	g.Status = InstallationStatusRevoked
	g.UpdatedAt = &now
	// Clear token when marking as revoked
	g.CachedToken = nil
	g.TokenExpiresAt = nil
}

// IsActive checks if the installation is active and not revoked
func (g *GitHubInstallation) IsActive() bool {
	return g.Status == InstallationStatusActive && !g.IsDeleted
}
