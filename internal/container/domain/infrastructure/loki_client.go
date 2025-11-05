package infrastructure

import (
	"context"
	"io"
	"time"
)

// LokiClient defines the interface for streaming logs from Loki
type LokiClient interface {
	// StreamPipelineRunLogs streams logs for a specific Tekton PipelineRun
	// excludeTasks specifies which task logs should be excluded (e.g., ["ecr-repository-check"])
	// Returns an io.ReadCloser that streams logs from Loki via WebSocket
	// The caller is responsible for closing the returned ReadCloser
	StreamPipelineRunLogs(ctx context.Context, pipelineRunName string, excludeTasks []string) (io.ReadCloser, error)

	// QueryPipelineRunLogsHTTP queries historical logs for a completed PipelineRun
	// Uses Loki's query_range HTTP API to retrieve logs within the specified time range
	// startTime and endTime define the time range to query
	// limit specifies the maximum number of log entries to return
	// Returns an io.ReadCloser containing the log data
	// The caller is responsible for closing the returned ReadCloser
	QueryPipelineRunLogsHTTP(ctx context.Context, pipelineRunName string, excludeTasks []string, startTime, endTime time.Time, limit int) (io.ReadCloser, error)
}
