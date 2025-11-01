package infrastructure

import (
	"context"
	"io"
)

// LokiClient defines the interface for streaming logs from Loki
type LokiClient interface {
	// StreamPipelineRunLogs streams logs for a specific Tekton PipelineRun
	// excludeTasks specifies which task logs should be excluded (e.g., ["ecr-repository-check"])
	// Returns an io.ReadCloser that streams logs from Loki via WebSocket
	// The caller is responsible for closing the returned ReadCloser
	StreamPipelineRunLogs(ctx context.Context, pipelineRunName string, excludeTasks []string) (io.ReadCloser, error)
}
