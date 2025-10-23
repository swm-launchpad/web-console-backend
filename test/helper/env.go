package helper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

// LoadTestEnv loads environment variables from .env.test file
// It tries multiple paths to find the .env.test file
func LoadTestEnv(t *testing.T) {
	t.Helper()

	// Try multiple paths to find .env.test
	paths := []string{
		".env.test",       // Current directory
		"../.env.test",    // Parent directory (for test/integration or test/e2e)
		"../../.env.test", // Two levels up
	}

	var err error
	loaded := false

	for _, path := range paths {
		err = godotenv.Load(path)
		if err == nil {
			loaded = true
			t.Logf("Loaded environment variables from: %s", path)
			break
		}
	}

	// If all paths fail, check if file exists in any of the paths
	if !loaded {
		// Check if .env.test exists in the project root
		rootPath := findProjectRoot()
		if rootPath != "" {
			envPath := filepath.Join(rootPath, ".env.test")
			err = godotenv.Load(envPath)
			if err == nil {
				loaded = true
				t.Logf("Loaded environment variables from: %s", envPath)
			}
		}
	}

	// If still not loaded, it's okay - environment variables might already be set
	// This is useful for CI/CD environments
	if !loaded {
		t.Logf("Warning: Failed to load .env.test file (environment variables might already be set): %v", err)
	}
}

// findProjectRoot finds the project root by looking for go.mod file
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up the directory tree to find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			break
		}
		dir = parent
	}

	return ""
}
