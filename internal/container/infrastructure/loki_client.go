package infrastructure

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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
	params.Add("limit", "2000") // Maximum entries to buffer (increased for safety margin)

	// Set start time to 1 hour ago to retrieve recent logs only
	// This prevents mixing logs from different builds
	startTime := time.Now().Add(-1 * time.Hour)
	params.Add("start", fmt.Sprintf("%d", startTime.UnixNano()))

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
		c.logger.Debug(ctx, "Outer goroutine STARTED")
		defer func() {
			c.logger.Debug(ctx, "Outer goroutine EXITING (defer cleanup complete)")
		}()
		defer func() {
			_ = pw.Close()
		}()
		defer func() {
			c.logger.Debug(ctx, "Defer: Starting cleanup sequence")

			// Step 1: Force any blocked ReadMessage() to timeout immediately
			// SetReadDeadline is the ONLY way to interrupt a blocked ReadMessage() call
			c.logger.Debug(ctx, "Defer: Setting read deadline to NOW")
			_ = conn.SetReadDeadline(time.Now())

			// Step 2: Send proper WebSocket close frame to Loki server
			// This signals Loki to decrement its connection counter
			c.logger.Debug(ctx, "Defer: Sending close frame to Loki")
			closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			writeDeadline := time.Now().Add(time.Second)
			if err := conn.WriteControl(websocket.CloseMessage, closeMsg, writeDeadline); err != nil {
				c.logger.Debug(ctx, "Defer: Failed to send close frame to Loki", zap.Error(err))
			} else {
				c.logger.Debug(ctx, "Defer: Close frame sent successfully")
			}

			// Step 3: Close the connection
			c.logger.Debug(ctx, "Defer: Closing WebSocket connection")
			_ = conn.Close()
			c.logger.Debug(ctx, "Defer: WebSocket connection closed")
		}()

		// WebSocket message structure
		type wsMessage struct {
			messageType int
			message     []byte
			err         error
		}

		// Channel for WebSocket messages
		msgChan := make(chan wsMessage, 1)

		// Read WebSocket messages in a separate goroutine
		go func() {
			c.logger.Debug(ctx, "Inner read goroutine STARTED")
			defer func() {
				c.logger.Debug(ctx, "Inner read goroutine EXITING")
			}()

			for {
				c.logger.Debug(ctx, "Inner: Calling ReadMessage() - WILL BLOCK until message or deadline")
				messageType, message, err := conn.ReadMessage()
				c.logger.Debug(ctx, "Inner: ReadMessage() returned",
					zap.Error(err),
					zap.Int("messageType", messageType),
					zap.Int("messageSize", len(message)),
				)

				c.logger.Debug(ctx, "Inner: Attempting to send to msgChan...")
				select {
				case msgChan <- wsMessage{messageType, message, err}:
					c.logger.Debug(ctx, "Inner: Successfully sent to msgChan")
					if err != nil {
						c.logger.Debug(ctx, "Inner: Error detected, returning from inner goroutine")
						return // Exit on error
					}
				case <-ctx.Done():
					c.logger.Debug(ctx, "Inner: ctx.Done() detected while trying to send to msgChan, returning")
					return // Exit on context cancellation
				}
			}
		}()

		// Monitor both context cancellation and WebSocket messages
		for {
			select {
			case <-ctx.Done():
				c.logger.Debug(ctx, "Outer: ctx.Done() received, initiating shutdown")
				c.logger.Debug(ctx, "Outer: Returning to execute defers")
				return

			case msg := <-msgChan:
				if msg.err != nil {
					if websocket.IsCloseError(msg.err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						c.logger.Debug(ctx, "Loki WebSocket closed normally")
					} else {
						c.logger.Warn(ctx, "Error reading from Loki WebSocket",
							zap.Error(msg.err),
						)
						pw.CloseWithError(msg.err)
					}
					return
				}

				// Only process text messages (logs are sent as text)
				if msg.messageType == websocket.TextMessage {
					// Write log message to pipe
					if _, err := pw.Write(msg.message); err != nil {
						c.logger.Warn(ctx, "Error writing to pipe",
							zap.Error(err),
						)
						return
					}
					// Note: No newline needed - WebSocket messages are already delimited
				}
			}
		}
	}()

	return pr, nil
}

// QueryPipelineRunLogsHTTP retrieves historical logs for a completed PipelineRun via Loki HTTP API
// Uses /loki/api/v1/query_range for querying historical logs (not real-time streaming)
// startTime and endTime define the time range to query (use build's startedAt/finishedAt)
// limit specifies the maximum number of log entries to return
func (c *lokiClient) QueryPipelineRunLogsHTTP(ctx context.Context, pipelineRunName string, excludeTasks []string, startTime, endTime time.Time, limit int) (io.ReadCloser, error) {
	// Build LogQL query
	query := c.buildLogQLQuery(pipelineRunName, excludeTasks)

	// Build query_range URL
	queryURL := c.baseURL + "/loki/api/v1/query_range"

	// Add query parameters
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", fmt.Sprintf("%d", startTime.UnixNano()))
	params.Add("end", fmt.Sprintf("%d", endTime.UnixNano()))
	params.Add("limit", fmt.Sprintf("%d", limit)) // Use provided limit
	params.Add("direction", "forward")            // Chronological order

	fullURL := queryURL + "?" + params.Encode()

	c.logger.Info(ctx, "Querying Loki for historical logs",
		zap.String("url", queryURL),
		zap.String("query", query),
		zap.Time("start", startTime),
		zap.Time("end", endTime),
		zap.Int("limit", limit),
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add Basic Auth
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Add X-Scope-OrgID header
	if c.orgID != "" {
		req.Header.Add("X-Scope-OrgID", c.orgID)
	}

	// Execute HTTP request
	client := &http.Client{
		Timeout: 30 * time.Second, // Timeout for HTTP request
	}

	resp, err := client.Do(req)
	if err != nil {
		c.logger.Error(ctx, "Failed to query Loki HTTP API",
			zap.String("url", fullURL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to query Loki: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		defer func() {
			_ = resp.Body.Close()
		}()
		c.logger.Error(ctx, "Loki HTTP API returned non-200 status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", fullURL),
		)
		return nil, fmt.Errorf("loki returned status %d", resp.StatusCode)
	}

	c.logger.Info(ctx, "Successfully retrieved historical logs from Loki",
		zap.String("pipelineRunName", pipelineRunName),
	)

	// Return response body as ReadCloser (caller must close it)
	return resp.Body, nil
}

// buildLogQLQuery constructs a LogQL query for filtering PipelineRun logs
func (c *lokiClient) buildLogQLQuery(pipelineRunName string, excludeTasks []string) string {
	// Base query: match all pods in build-pipeline namespace with the PipelineRun name
	if len(excludeTasks) == 0 {
		return fmt.Sprintf(`{namespace="build-pipeline",pod=~"%s-.*-pod"}`, pipelineRunName)
	}

	// Add negative pod matcher to exclude specified tasks
	// Use regex OR pattern for multiple exclusions: (task1|task2|task3)
	excludePattern := strings.Join(excludeTasks, "|")
	return fmt.Sprintf(`{namespace="build-pipeline",pod=~"%s-.*-pod",pod!~".*-(%s)-pod"}`,
		pipelineRunName, excludePattern)
}
