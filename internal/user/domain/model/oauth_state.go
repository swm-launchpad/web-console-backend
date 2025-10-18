package model

import "time"

// OAuthState represents a temporary state token for OAuth flows (CSRF protection)
type OAuthState struct {
	State          string
	UserID         uint
	InstallationID *int64    // Populated during callback verification
	ExpiresAt      time.Time // State expires after 10 minutes
	CreatedAt      time.Time
	ConsumedAt     *time.Time // One-time use tracking
}

// IsExpired checks if the state has expired
func (s *OAuthState) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsConsumed checks if the state has already been used
func (s *OAuthState) IsConsumed() bool {
	return s.ConsumedAt != nil
}

// CanBeUsed checks if the state is valid and can be consumed
func (s *OAuthState) CanBeUsed() bool {
	return !s.IsExpired() && !s.IsConsumed()
}

// MatchesInstallation checks if the state was issued for the given installation
func (s *OAuthState) MatchesInstallation(installationID int64) bool {
	if s.InstallationID == nil {
		// State not yet bound to installation (shouldn't happen in callback)
		return false
	}
	return *s.InstallationID == installationID
}
