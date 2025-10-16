package value

import (
	"regexp"

	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

// GitConfig represents Git repository configuration
// It is a value object that encapsulates Git-related settings
type GitConfig struct {
	repositoryURL string
	branch        string
	directoryPath *string
	commitHash    *string
}

var (
	// Git URL patterns
	gitHTTPRegex = regexp.MustCompile(`^https?://[\w\-.]+(/[\w\-./_]+)?(\.git)?$`)
	gitSSHRegex  = regexp.MustCompile(`^git@[\w\-.]+:[\w\-./_]+(\.git)?$`)
	gitSSH2Regex = regexp.MustCompile(`^ssh://git@[\w\-.]+(/[\w\-./_]+)?(\.git)?$`)

	// Git branch name pattern
	gitBranchRegex = regexp.MustCompile(`^[\w\-./]+$`)

	// Git directory path pattern (no leading/trailing slashes, no ..)
	gitPathRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-./]+$`)
)

// NewGitConfig creates a new GitConfig with validation
// URL can be empty for template-based containers (e.g., databases)
func NewGitConfig(url, branch string, dirPath *string) (GitConfig, error) {
	// Validate repository URL only if provided (empty URL allowed for template-based containers)
	if url != "" {
		if !isValidGitURL(url) {
			return GitConfig{}, containererrors.ErrInvalidGitURL
		}
	}

	// Validate branch (default to "main" if empty)
	if branch == "" {
		branch = "main"
	}
	if !gitBranchRegex.MatchString(branch) {
		return GitConfig{}, containererrors.ErrInvalidGitBranch
	}

	// Validate directory path if provided
	if dirPath != nil && *dirPath != "" {
		if !isValidGitPath(*dirPath) {
			return GitConfig{}, containererrors.ErrInvalidGitPath
		}
	}

	return GitConfig{
		repositoryURL: url,
		branch:        branch,
		directoryPath: dirPath,
		commitHash:    nil,
	}, nil
}

// NewGitConfigWithCommit creates a GitConfig with commit hash
func NewGitConfigWithCommit(url, branch string, dirPath *string, commitHash *string) (GitConfig, error) {
	config, err := NewGitConfig(url, branch, dirPath)
	if err != nil {
		return GitConfig{}, err
	}

	// Validate commit hash if provided (40-character SHA-1)
	if commitHash != nil && *commitHash != "" {
		if len(*commitHash) != 40 {
			return GitConfig{}, containererrors.ErrInvalidGitCommitHash
		}
		if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(*commitHash) {
			return GitConfig{}, containererrors.ErrInvalidGitCommitHash
		}
	}

	config.commitHash = commitHash
	return config, nil
}

// RepositoryURL returns the repository URL
func (g GitConfig) RepositoryURL() string {
	return g.repositoryURL
}

// Branch returns the branch name
func (g GitConfig) Branch() string {
	return g.branch
}

// DirectoryPath returns the directory path (may be nil)
func (g GitConfig) DirectoryPath() *string {
	return g.directoryPath
}

// CommitHash returns the commit hash (may be nil)
func (g GitConfig) CommitHash() *string {
	return g.commitHash
}

// WithCommitHash returns a new GitConfig with the specified commit hash
func (g GitConfig) WithCommitHash(commitHash *string) (GitConfig, error) {
	return NewGitConfigWithCommit(g.repositoryURL, g.branch, g.directoryPath, commitHash)
}

// Equals checks if two GitConfigs are equal
func (g GitConfig) Equals(other GitConfig) bool {
	if g.repositoryURL != other.repositoryURL || g.branch != other.branch {
		return false
	}

	// Compare directory paths
	if (g.directoryPath == nil) != (other.directoryPath == nil) {
		return false
	}
	if g.directoryPath != nil && *g.directoryPath != *other.directoryPath {
		return false
	}

	// Compare commit hashes
	if (g.commitHash == nil) != (other.commitHash == nil) {
		return false
	}
	if g.commitHash != nil && *g.commitHash != *other.commitHash {
		return false
	}

	return true
}

// isValidGitURL checks if the URL is a valid Git repository URL
func isValidGitURL(url string) bool {
	return gitHTTPRegex.MatchString(url) ||
		gitSSHRegex.MatchString(url) ||
		gitSSH2Regex.MatchString(url)
}

// isValidGitPath checks if the path is a valid Git directory path
func isValidGitPath(path string) bool {
	// Empty path is valid
	if path == "" {
		return true
	}

	// No leading or trailing slashes
	if path[0] == '/' || path[len(path)-1] == '/' {
		return false
	}

	// No ".." for security
	if regexp.MustCompile(`\.\.`).MatchString(path) {
		return false
	}

	// Match allowed characters
	return gitPathRegex.MatchString(path)
}
