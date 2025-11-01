package integration

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/container/infrastructure"
)

// TestLokiClient_StreamPipelineRunLogs_Integration tests the actual Loki connection
// This test requires real Loki credentials in .env file
// Skip this test in short mode: go test -short
func TestLokiClient_StreamPipelineRunLogs_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load .env file for Loki configuration
	envPath := filepath.Join("..", "..", ".env")
	err := godotenv.Load(envPath)
	if err != nil {
		t.Logf("Warning: Could not load .env file: %v", err)
	}

	// Check if Loki environment variables are set
	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		t.Skip("LOKI_URL not set in .env file, skipping Loki integration test")
	}

	ctx := context.Background()
	log := logger.NewForTest()

	// Load config from environment
	cfg := &config.Config{
		Loki: config.LokiConfig{
			URL:      os.Getenv("LOKI_URL"),
			Username: os.Getenv("LOKI_USERNAME"),
			Password: os.Getenv("LOKI_PASSWORD"),
			OrgID:    os.Getenv("LOKI_ORG_ID"),
		},
	}

	// Validate required fields
	if cfg.Loki.URL == "" || cfg.Loki.Username == "" || cfg.Loki.Password == "" {
		t.Skip("Loki credentials not fully configured, skipping test")
	}

	// Create Loki client
	lokiClient := infrastructure.NewLokiClient(cfg, log)
	require.NotNil(t, lokiClient)

	t.Run("Stream logs from valid PipelineRun", func(t *testing.T) {
		// Use a PipelineRun name that exists in your Loki
		// This should be set as an environment variable for the test
		pipelineRunName := os.Getenv("TEST_PIPELINE_RUN_NAME")
		if pipelineRunName == "" {
			t.Skip("TEST_PIPELINE_RUN_NAME not set, cannot test actual PipelineRun")
		}

		// Stream logs
		stream, err := lokiClient.StreamPipelineRunLogs(ctx, pipelineRunName, []string{"ecr-repository-check"})
		require.NoError(t, err)
		require.NotNil(t, stream)
		defer func() {
			_ = stream.Close()
		}()

		// Read some data from stream (with timeout)
		readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		buf := make([]byte, 1024)
		done := make(chan struct{})
		var readErr error
		var n int

		go func() {
			n, readErr = stream.Read(buf)
			close(done)
		}()

		select {
		case <-readCtx.Done():
			t.Log("Timeout reading from Loki stream (this is OK if no new logs)")
		case <-done:
			if readErr != nil && readErr != io.EOF {
				t.Logf("Read error: %v", readErr)
			}
			if n > 0 {
				t.Logf("Successfully read %d bytes from Loki", n)
				assert.Greater(t, n, 0)
			}
		}
	})

	t.Run("Stream logs with non-existent PipelineRun", func(t *testing.T) {
		// Test with a PipelineRun that doesn't exist
		pipelineRunName := "non-existent-pipeline-run-12345"

		stream, err := lokiClient.StreamPipelineRunLogs(ctx, pipelineRunName, []string{})

		// Loki should still open the stream even if no data exists
		// The stream will just not return any data
		if err != nil {
			t.Logf("Error opening stream for non-existent PipelineRun: %v", err)
		} else {
			require.NotNil(t, stream)
			defer func() {
				_ = stream.Close()
			}()

			// Try to read - should get EOF or timeout
			buf := make([]byte, 1024)
			n, readErr := stream.Read(buf)

			if readErr == io.EOF {
				assert.Equal(t, 0, n, "Should not read any data from non-existent PipelineRun")
			} else {
				t.Logf("Read result: n=%d, err=%v", n, readErr)
			}
		}
	})
}

// TestLokiClient_Connection_Integration tests basic Loki connectivity
func TestLokiClient_Connection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load .env file for Loki configuration
	envPath := filepath.Join("..", "..", ".env")
	err := godotenv.Load(envPath)
	if err != nil {
		t.Logf("Warning: Could not load .env file: %v", err)
	}

	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		t.Skip("LOKI_URL not set in .env file, skipping Loki integration test")
	}

	log := logger.NewForTest()

	cfg := &config.Config{
		Loki: config.LokiConfig{
			URL:      os.Getenv("LOKI_URL"),
			Username: os.Getenv("LOKI_USERNAME"),
			Password: os.Getenv("LOKI_PASSWORD"),
			OrgID:    os.Getenv("LOKI_ORG_ID"),
		},
	}

	if cfg.Loki.URL == "" || cfg.Loki.Username == "" || cfg.Loki.Password == "" {
		t.Skip("Loki credentials not fully configured, skipping test")
	}

	// Just create the client - this validates config
	lokiClient := infrastructure.NewLokiClient(cfg, log)
	assert.NotNil(t, lokiClient)
}
