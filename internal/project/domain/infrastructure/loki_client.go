package infrastructure

import (
	"context"
	"io"
	"time"
)

// LokiClient defines the interface for querying logs from Loki
type LokiClient interface {
	// StreamApplicationLogs streams real-time logs from application pods
	// projectSlug identifies the Knative service (e.g., "spring-helloworld")
	// since specifies the start time for streaming logs
	// Returns an io.ReadCloser that streams logs from Loki via WebSocket
	// The caller is responsible for closing the returned ReadCloser
	StreamApplicationLogs(ctx context.Context, projectSlug string, since time.Time) (io.ReadCloser, error)

	// QueryApplicationLogsHistory queries historical logs with pagination (backward direction)
	// projectSlug identifies the Knative service
	// before specifies the end time for the query (if zero, uses current time)
	// limit specifies the maximum number of log entries to return
	// Returns a slice of log entries ordered by timestamp (oldest first)
	QueryApplicationLogsHistory(ctx context.Context, projectSlug string, before time.Time, limit int) ([]ApplicationLogEntry, error)

	// QueryApplicationLogsAfter queries logs after a specific timestamp (forward direction)
	// Used for forward pagination when scrolling to bottom
	// projectSlug identifies the Knative service
	// after specifies the start time for the query
	// limit specifies the maximum number of log entries to return
	// Returns a slice of log entries ordered by timestamp (oldest first)
	QueryApplicationLogsAfter(ctx context.Context, projectSlug string, after time.Time, limit int) ([]ApplicationLogEntry, error)

	// QueryApplicationLogsHistoryRaw queries historical logs as raw Loki JSON stream (backward direction)
	// projectSlug identifies the Knative service
	// before specifies the end time for the query (if zero, uses current time)
	// limit specifies the maximum number of log entries to return
	// Returns an io.ReadCloser with raw Loki JSON response
	// The caller is responsible for closing the returned ReadCloser
	QueryApplicationLogsHistoryRaw(ctx context.Context, projectSlug string, before time.Time, limit int) (io.ReadCloser, error)

	// QueryApplicationLogsAfterRaw queries logs as raw Loki JSON stream (forward direction)
	// projectSlug identifies the Knative service
	// after specifies the start time for the query
	// limit specifies the maximum number of log entries to return
	// Returns an io.ReadCloser with raw Loki JSON response
	// The caller is responsible for closing the returned ReadCloser
	QueryApplicationLogsAfterRaw(ctx context.Context, projectSlug string, after time.Time, limit int) (io.ReadCloser, error)
}

// ApplicationLogEntry represents a single log entry from an application container
type ApplicationLogEntry struct {
	Timestamp     time.Time `json:"timestamp"`      // ISO 8601 timestamp (for UI display)
	TimestampNs   string    `json:"timestamp_ns"`   // Nanosecond timestamp string from Loki (for deduplication)
	ContainerName string    `json:"container_name"` // e.g., "nginx-proxy", "backend", "database"
	LogLine       string    `json:"log_line"`
}
