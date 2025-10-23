package github

import "errors"

var (
	ErrInvalidPrivateKey        = errors.New("invalid GitHub App private key")
	ErrFailedToGenerateJWT      = errors.New("failed to generate GitHub App JWT")
	ErrFailedToCreateToken      = errors.New("failed to create installation access token")
	ErrInvalidInstallationID    = errors.New("invalid installation ID")
	ErrInstallationNotFound     = errors.New("GitHub installation not found (app may be uninstalled)")
	ErrInstallationUnauthorized = errors.New("unauthorized to access GitHub installation")
	ErrInstallationRevoked      = errors.New("GitHub installation has been revoked")
	ErrTokenExpired             = errors.New("token has expired")
	ErrMissingAppID             = errors.New("GitHub App ID is required")
	ErrMissingPrivateKey        = errors.New("GitHub App private key path is required")
	ErrRepositoryNotFound       = errors.New("repository not found")
	ErrRepositoryForbidden      = errors.New("access to repository forbidden")
)
