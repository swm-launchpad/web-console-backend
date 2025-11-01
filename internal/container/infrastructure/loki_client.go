package infrastructure

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
)

// lokiClient implements the LokiClient interface
type lokiClient struct {
	baseURL  string
	username string
	password string
	orgID    string
	logger   logger.Logger
}

// NewLokiClient creates a new Loki client instance
func NewLokiClient(cfg *config.Config, log logger.Logger) *lokiClient {
	return &lokiClient{
		baseURL:  cfg.Loki.URL,
		username: cfg.Loki.Username,
		password: cfg.Loki.Password,
		orgID:    cfg.Loki.OrgID,
		logger:   log,
	}
}

// StreamPipelineRunLogs streams logs for a specific Tekton PipelineRun via Loki WebSocket
func (c *lokiClient) StreamPipelineRunLogs(ctx context.Context, pipelineRunName string, excludeTasks []string) (io.ReadCloser, error) {
	// Build LogQL query
	query := c.buildLogQLQuery(pipelineRunName, excludeTasks)

	// Convert http(s):// to ws(s)://
	wsURL := strings.Replace(c.baseURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL = wsURL + "/loki/api/v1/tail"

	// Add query parameters
	params := url.Values{}
	params.Add("query", query)
	params.Add("limit", "1000") // Maximum entries to buffer
	fullURL := wsURL + "?" + params.Encode()

	// Prepare headers
	headers := http.Header{}

	// Add Basic Auth
	if c.username != "" && c.password != "" {
		auth := c.username + ":" + c.password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		headers.Add("Authorization", "Basic "+encodedAuth)
	}

	// Add X-Scope-OrgID header
	if c.orgID != "" {
		headers.Add("X-Scope-OrgID", c.orgID)
	}

	c.logger.Info(ctx, "Connecting to Loki WebSocket",
		zap.String("url", wsURL),
		zap.String("query", query),
	)

	// Establish WebSocket connection
	dialer := websocket.Dialer{}
	conn, _, err := dialer.DialContext(ctx, fullURL, headers)
	if err != nil {
		c.logger.Error(ctx, "Failed to connect to Loki WebSocket",
			zap.String("url", fullURL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to Loki: %w", err)
	}

	// Create a pipe for streaming data
	pr, pw := io.Pipe()

	// Start goroutine to read from WebSocket and write to pipe
	go func() {
		defer func() {
			_ = conn.Close()
		}()
		defer func() {
			_ = pw.Close()
		}()

		for {
			select {
			case <-ctx.Done():
				c.logger.Debug(ctx, "Context cancelled, closing Loki WebSocket connection")
				return
			default:
				messageType, message, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						c.logger.Debug(ctx, "Loki WebSocket closed normally")
					} else {
						c.logger.Warn(ctx, "Error reading from Loki WebSocket",
							zap.Error(err),
						)
						pw.CloseWithError(err)
					}
					return
				}

				// Only process text messages (logs are sent as text)
				if messageType == websocket.TextMessage {
					// Write log message to pipe
					if _, err := pw.Write(message); err != nil {
						c.logger.Warn(ctx, "Error writing to pipe",
							zap.Error(err),
						)
						return
					}
					// Add newline for readability
					if _, err := pw.Write([]byte("\n")); err != nil {
						c.logger.Warn(ctx, "Error writing newline to pipe",
							zap.Error(err),
						)
						return
					}
				}
			}
		}
	}()

	return pr, nil
}

// buildLogQLQuery constructs a LogQL query for filtering PipelineRun logs
func (c *lokiClient) buildLogQLQuery(pipelineRunName string, excludeTasks []string) string {
	// Base query: match all pods in build-pipeline namespace with the PipelineRun name
	query := fmt.Sprintf(`{namespace="build-pipeline",pod=~"%s-.*"}`, pipelineRunName)

	// Add exclusion filters for specified tasks
	for _, task := range excludeTasks {
		// Exclude pods containing the task name
		query += fmt.Sprintf(` !~ "%s"`, task)
	}

	return query
}
