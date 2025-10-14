package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

func TestNewGitConfig_ValidConfigs(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		branch string
		path   *string
	}{
		{
			name:   "HTTPS URL with default branch",
			url:    "https://github.com/user/repo.git",
			branch: "",
			path:   nil,
		},
		{
			name:   "HTTPS URL with custom branch",
			url:    "https://github.com/user/repo.git",
			branch: "develop",
			path:   nil,
		},
		{
			name:   "SSH URL",
			url:    "git@github.com:user/repo.git",
			branch: "main",
			path:   nil,
		},
		{
			name:   "With directory path",
			url:    "https://github.com/user/repo.git",
			branch: "main",
			path:   stringPtr("backend"),
		},
		{
			name:   "Nested directory path",
			url:    "https://github.com/user/repo.git",
			branch: "main",
			path:   stringPtr("apps/backend"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewGitConfig(tt.url, tt.branch, tt.path)
			assert.NoError(t, err)
			assert.Equal(t, tt.url, config.RepositoryURL())

			// Branch defaults to "main" if empty
			expectedBranch := tt.branch
			if expectedBranch == "" {
				expectedBranch = "main"
			}
			assert.Equal(t, expectedBranch, config.Branch())

			assert.Equal(t, tt.path, config.DirectoryPath())
		})
	}
}

func TestNewGitConfig_EmptyURL(t *testing.T) {
	_, err := NewGitConfig("", "main", nil)
	assert.ErrorIs(t, err, containererrors.ErrGitURLRequired)
}

func TestNewGitConfig_InvalidURL(t *testing.T) {
	invalidURLs := []string{
		"not-a-url",
		"ftp://example.com/repo.git",
		"http://",
		"github.com/user/repo.git", // missing protocol
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			_, err := NewGitConfig(url, "main", nil)
			assert.ErrorIs(t, err, containererrors.ErrInvalidGitURL)
		})
	}
}

func TestNewGitConfig_InvalidBranch(t *testing.T) {
	invalidBranches := []string{
		"feature@123",
		"branch name", // space
		"branch\ttab",
	}

	for _, branch := range invalidBranches {
		t.Run(branch, func(t *testing.T) {
			_, err := NewGitConfig("https://github.com/user/repo.git", branch, nil)
			assert.ErrorIs(t, err, containererrors.ErrInvalidGitBranch)
		})
	}
}

func TestNewGitConfig_InvalidPath(t *testing.T) {
	invalidPaths := []string{
		"/absolute/path", // starts with slash
		"path/",          // ends with slash
		"../parent",      // contains ..
		"path/../other",  // contains ..
	}

	for _, path := range invalidPaths {
		t.Run(path, func(t *testing.T) {
			_, err := NewGitConfig("https://github.com/user/repo.git", "main", &path)
			assert.ErrorIs(t, err, containererrors.ErrInvalidGitPath)
		})
	}
}

func TestNewGitConfigWithCommit_ValidCommitHash(t *testing.T) {
	commitHash := "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3" // 40-char SHA-1
	config, err := NewGitConfigWithCommit(
		"https://github.com/user/repo.git",
		"main",
		nil,
		&commitHash,
	)

	assert.NoError(t, err)
	assert.Equal(t, &commitHash, config.CommitHash())
}

func TestNewGitConfigWithCommit_InvalidCommitHash(t *testing.T) {
	invalidHashes := []string{
		"abc123", // too short
		"a94a8fe5ccb19ba61c4c0873d391e987982fbbd3xyz", // invalid characters
		"a94a8fe5ccb19ba61c4c0873d391e987982fbbd",     // 39 characters
	}

	for _, hash := range invalidHashes {
		t.Run(hash, func(t *testing.T) {
			_, err := NewGitConfigWithCommit(
				"https://github.com/user/repo.git",
				"main",
				nil,
				&hash,
			)
			assert.ErrorIs(t, err, containererrors.ErrInvalidGitCommitHash)
		})
	}
}

func TestGitConfig_Equals(t *testing.T) {
	path := "backend"
	config1, _ := NewGitConfig("https://github.com/user/repo.git", "main", &path)
	config2, _ := NewGitConfig("https://github.com/user/repo.git", "main", &path)
	config3, _ := NewGitConfig("https://github.com/user/repo.git", "develop", &path)

	assert.True(t, config1.Equals(config2))
	assert.False(t, config1.Equals(config3))
}

func stringPtr(s string) *string {
	return &s
}
