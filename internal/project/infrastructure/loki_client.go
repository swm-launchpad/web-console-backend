package infrastructure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/config"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure"
)

// lokiClient implements the infrastructure.LokiClient interface for querying application logs
type lokiClient struct {
	baseURL    string
	username   string
	password   string
	orgID      string
	httpClient *http.Client
	kubeClient infrastructure.KubeClient
	logger     logger.Logger
}

// NewLokiClient creates a new Loki client instance for project application logs
func NewLokiClient(cfg *config.Config, kubeClient infrastructure.KubeClient, log logger.Logger) infrastructure.LokiClient {
	return &lokiClient{
		baseURL:    cfg.Loki.URL,
		username:   cfg.Loki.Username,
		password:   cfg.Loki.Password,
		orgID:      cfg.Loki.OrgID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		kubeClient: kubeClient,
		logger:     log,
	}
}

// StreamApplicationLogs streams real-time logs from application pods via WebSocket
func (c *lokiClient) StreamApplicationLogs(ctx context.Context, projectSlug string, since time.Time) (io.ReadCloser, error) {
	// Build LogQL query for application namespace
	// Match all revisions (e.g., projectSlug-00001, projectSlug-00002) and exclude sidecar containers
	query := fmt.Sprintf(`{namespace="application",app=~"%s-[0-9]+",container!~"queue-proxy|istio-proxy|nginx-proxy"}`, projectSlug)

	// Convert http(s):// to ws(s)://
	wsURL := strings.Replace(c.baseURL, "https://", "wss://", 1)
	wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
	wsURL = wsURL + "/loki/api/v1/tail"

	// Add query parameters
	params := url.Values{}
	params.Add("query", query)
	params.Add("start", fmt.Sprintf("%d", since.UnixNano()))
	params.Add("limit", "1000")

	fullURL := wsURL + "?" + params.Encode()

	// Prepare headers
	headers := http.Header{}
	if c.username != "" && c.password != "" {
		auth := c.username + ":" + c.password
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		headers.Add("Authorization", "Basic "+encodedAuth)
	}
	if c.orgID != "" {
		headers.Add("X-Scope-OrgID", c.orgID)
	}

	c.logger.Info(ctx, "Connecting to Loki WebSocket for application logs",
		zap.String("url", wsURL),
		zap.String("query", query),
		zap.String("project_slug", projectSlug),
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

	c.logger.Info(ctx, "Loki WebSocket connected for application logs",
		zap.String("project_slug", projectSlug),
	)

	// Create a pipe for streaming data
	pr, pw := io.Pipe()

	// Create cancellable context for coordinated shutdown
	wsCtx, wsCancel := context.WithCancel(ctx)

	// Start goroutine to read from WebSocket and write to pipe
	go func() {
		c.logger.Debug(wsCtx, "Application log outer goroutine STARTED",
			zap.String("project_slug", projectSlug),
		)
		defer func() {
			c.logger.Debug(wsCtx, "Application log outer goroutine EXITING (defer cleanup complete)",
				zap.String("project_slug", projectSlug),
			)
		}()
		defer func() {
			_ = pw.Close()
		}()
		defer func() {
			c.logger.Debug(wsCtx, "Defer: Starting cleanup sequence for application logs",
				zap.String("project_slug", projectSlug),
			)

			// Step 1: Force any blocked ReadMessage() to timeout immediately
			// SetReadDeadline is the ONLY way to interrupt a blocked ReadMessage() call
			c.logger.Debug(wsCtx, "Defer: Setting read deadline to NOW")
			_ = conn.SetReadDeadline(time.Now())

			// Step 2: Send proper WebSocket close frame to Loki server
			// This signals Loki to decrement its connection counter
			c.logger.Debug(wsCtx, "Defer: Sending close frame to Loki")
			closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			writeDeadline := time.Now().Add(time.Second)
			if err := conn.WriteControl(websocket.CloseMessage, closeMsg, writeDeadline); err != nil {
				c.logger.Debug(wsCtx, "Defer: Failed to send close frame to Loki", zap.Error(err))
			} else {
				c.logger.Debug(wsCtx, "Defer: Close frame sent successfully")
			}

			// Step 3: Close the connection
			c.logger.Debug(wsCtx, "Defer: Closing WebSocket connection")
			_ = conn.Close()
			c.logger.Debug(wsCtx, "Defer: WebSocket connection closed")
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
			c.logger.Debug(wsCtx, "Application log inner read goroutine STARTED",
				zap.String("project_slug", projectSlug),
			)
			defer func() {
				c.logger.Debug(wsCtx, "Application log inner read goroutine EXITING",
					zap.String("project_slug", projectSlug),
				)
			}()

			for {
				c.logger.Debug(wsCtx, "Inner: Calling ReadMessage() - WILL BLOCK until message or deadline")
				messageType, message, err := conn.ReadMessage()
				c.logger.Debug(wsCtx, "Inner: ReadMessage() returned",
					zap.Error(err),
					zap.Int("messageType", messageType),
					zap.Int("messageSize", len(message)),
				)

				c.logger.Debug(wsCtx, "Inner: Attempting to send to msgChan...")
				select {
				case msgChan <- wsMessage{messageType, message, err}:
					c.logger.Debug(wsCtx, "Inner: Successfully sent to msgChan")
					if err != nil {
						c.logger.Debug(wsCtx, "Inner: Error detected, returning from inner goroutine")
						return // Exit on error
					}
				case <-wsCtx.Done():
					c.logger.Debug(wsCtx, "Inner: ctx.Done() detected while trying to send to msgChan, returning")
					return // Exit on context cancellation
				}
			}
		}()

		// Monitor both context cancellation and WebSocket messages
		for {
			select {
			case <-wsCtx.Done():
				c.logger.Debug(wsCtx, "Outer: ctx.Done() received, initiating shutdown",
					zap.String("project_slug", projectSlug),
				)
				c.logger.Debug(wsCtx, "Outer: Returning to execute defers")
				return

			case msg := <-msgChan:
				if msg.err != nil {
					if websocket.IsCloseError(msg.err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						c.logger.Debug(wsCtx, "Loki WebSocket closed normally for application logs",
							zap.String("project_slug", projectSlug),
						)
					} else {
						c.logger.Warn(wsCtx, "Error reading from Loki WebSocket for application logs",
							zap.String("project_slug", projectSlug),
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
						c.logger.Warn(wsCtx, "Error writing to pipe for application logs",
							zap.String("project_slug", projectSlug),
							zap.Error(err),
						)
						return
					}
					// Note: No newline needed - WebSocket messages are already delimited by Loki
				}
			}
		}
	}()

	// Start pod monitoring goroutine
	go c.monitorPodsAndCancelOnDeath(wsCtx, wsCancel, projectSlug)

	return pr, nil
}

// monitorPodsAndCancelOnDeath monitors application pod status and cancels context when pods die
func (c *lokiClient) monitorPodsAndCancelOnDeath(ctx context.Context, cancel context.CancelFunc, projectSlug string) {
	c.logger.Info(ctx, "Started pod monitoring for application logs",
		zap.String("project_slug", projectSlug),
	)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - cleanup goroutine
			c.logger.Info(ctx, "Pod monitoring goroutine stopped",
				zap.String("project_slug", projectSlug),
			)
			return

		case <-ticker.C:
			// Check if pods are still running
			podsRunning, err := c.kubeClient.CheckApplicationPodsRunning(ctx, projectSlug)
			if err != nil {
				c.logger.Warn(ctx, "Failed to check pod status during monitoring",
					zap.String("project_slug", projectSlug),
					zap.Error(err),
				)
				// Don't cancel on error - could be temporary network issue
				continue
			}

			if !podsRunning {
				c.logger.Info(ctx, "Application pods died - cancelling WebSocket stream",
					zap.String("project_slug", projectSlug),
				)
				// Cancel context to trigger shutdown of WebSocket goroutines
				cancel()
				return
			}

			c.logger.Debug(ctx, "Pod status check: running",
				zap.String("project_slug", projectSlug),
			)
		}
	}
}

// QueryApplicationLogsHistory queries historical logs with pagination using HTTP API
func (c *lokiClient) QueryApplicationLogsHistory(ctx context.Context, projectSlug string, before time.Time, limit int) ([]infrastructure.ApplicationLogEntry, error) {
	// Build LogQL query
	// Match all revisions (e.g., projectSlug-00001, projectSlug-00002) and exclude sidecar containers
	query := fmt.Sprintf(`{namespace="application",app=~"%s-[0-9]+",container!~"queue-proxy|istio-proxy|nginx-proxy"}`, projectSlug)

	// Determine time range
	end := before
	if end.IsZero() {
		end = time.Now()
	}

	// 24-hour limit
	start := end.Add(-24 * time.Hour)

	// Build URL
	params := url.Values{}
	params.Add("query", query)
	params.Add("direction", "backward") // Latest first
	params.Add("limit", strconv.Itoa(limit))
	params.Add("start", fmt.Sprintf("%d", start.UnixNano()))
	params.Add("end", fmt.Sprintf("%d", end.UnixNano()))

	fullURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", c.baseURL, params.Encode())

	c.logger.Info(ctx, "Querying Loki for application log history",
		zap.String("project_slug", projectSlug),
		zap.Time("start", start),
		zap.Time("end", end),
		zap.Int("limit", limit),
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	if c.orgID != "" {
		req.Header.Set("X-Scope-OrgID", c.orgID)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(ctx, "Failed to query Loki",
			zap.String("url", fullURL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to query Loki: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error(ctx, "Loki returned non-OK status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("loki returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var lokiResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Stream map[string]string `json:"stream"`
				Values [][]string        `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		c.logger.Error(ctx, "Failed to parse Loki response", zap.Error(err))
		return nil, fmt.Errorf("failed to parse Loki response: %w", err)
	}

	if lokiResp.Status != "success" {
		return nil, fmt.Errorf("loki query failed with status: %s", lokiResp.Status)
	}

	c.logger.Info(ctx, "Loki query successful",
		zap.String("project_slug", projectSlug),
		zap.Int("result_count", len(lokiResp.Data.Result)),
	)

	// Convert to ApplicationLogEntry
	var entries []infrastructure.ApplicationLogEntry
	for _, result := range lokiResp.Data.Result {
		containerName := result.Stream["container"]
		if containerName == "" {
			containerName = "unknown"
		}

		for _, value := range result.Values {
			// value[0] = timestamp (nanoseconds as string)
			// value[1] = log line
			timestampNano, err := strconv.ParseInt(value[0], 10, 64)
			if err != nil {
				c.logger.Warn(ctx, "Failed to parse timestamp", zap.String("timestamp", value[0]))
				continue
			}

			timestamp := time.Unix(0, timestampNano)
			logLine := value[1]

			entries = append(entries, infrastructure.ApplicationLogEntry{
				Timestamp:     timestamp,
				TimestampNs:   value[0], // Preserve nanosecond timestamp from Loki
				ContainerName: containerName,
				LogLine:       logLine,
			})
		}
	}

	// Sort by nanosecond timestamp (oldest first) for precise ordering
	// Loki returns in backward direction (newest first), so we reverse it
	// Use TimestampNs (string) instead of Timestamp to preserve nanosecond precision
	sort.Slice(entries, func(i, j int) bool {
		// Parse nanosecond timestamps as int64 for comparison
		iNs, _ := strconv.ParseInt(entries[i].TimestampNs, 10, 64)
		jNs, _ := strconv.ParseInt(entries[j].TimestampNs, 10, 64)
		return iNs < jNs
	})

	c.logger.Info(ctx, "Application log history query completed",
		zap.String("project_slug", projectSlug),
		zap.Int("total_entries", len(entries)),
	)

	return entries, nil
}

// QueryApplicationLogsAfter queries logs after a specific timestamp for forward pagination
func (c *lokiClient) QueryApplicationLogsAfter(ctx context.Context, projectSlug string, after time.Time, limit int) ([]infrastructure.ApplicationLogEntry, error) {
	// Build LogQL query
	// Match all revisions (e.g., projectSlug-00001, projectSlug-00002) and exclude sidecar containers
	query := fmt.Sprintf(`{namespace="application",app=~"%s-[0-9]+",container!~"queue-proxy|istio-proxy|nginx-proxy"}`, projectSlug)

	// Determine time range
	start := after
	end := time.Now()

	// Build URL
	params := url.Values{}
	params.Add("query", query)
	params.Add("direction", "forward") // Forward direction for newer logs
	params.Add("limit", strconv.Itoa(limit))
	params.Add("start", fmt.Sprintf("%d", start.UnixNano()))
	params.Add("end", fmt.Sprintf("%d", end.UnixNano()))

	fullURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", c.baseURL, params.Encode())

	c.logger.Info(ctx, "Querying Loki for newer application logs (forward)",
		zap.String("project_slug", projectSlug),
		zap.Time("after", after),
		zap.Time("end", end),
		zap.Int("limit", limit),
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	if c.orgID != "" {
		req.Header.Set("X-Scope-OrgID", c.orgID)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(ctx, "Failed to query Loki",
			zap.String("url", fullURL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to query Loki: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error(ctx, "Loki returned non-OK status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("loki returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var lokiResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Stream map[string]string `json:"stream"`
				Values [][]string        `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		c.logger.Error(ctx, "Failed to parse Loki response", zap.Error(err))
		return nil, fmt.Errorf("failed to parse Loki response: %w", err)
	}

	if lokiResp.Status != "success" {
		return nil, fmt.Errorf("loki query failed with status: %s", lokiResp.Status)
	}

	c.logger.Info(ctx, "Loki forward query successful",
		zap.String("project_slug", projectSlug),
		zap.Int("result_count", len(lokiResp.Data.Result)),
	)

	// Convert to ApplicationLogEntry
	var entries []infrastructure.ApplicationLogEntry
	for _, result := range lokiResp.Data.Result {
		containerName := result.Stream["container"]
		if containerName == "" {
			containerName = "unknown"
		}

		for _, value := range result.Values {
			// value[0] = timestamp (nanoseconds as string)
			// value[1] = log line
			timestampNano, err := strconv.ParseInt(value[0], 10, 64)
			if err != nil {
				c.logger.Warn(ctx, "Failed to parse timestamp", zap.String("timestamp", value[0]))
				continue
			}

			timestamp := time.Unix(0, timestampNano)
			logLine := value[1]

			entries = append(entries, infrastructure.ApplicationLogEntry{
				Timestamp:     timestamp,
				TimestampNs:   value[0], // Preserve nanosecond timestamp from Loki
				ContainerName: containerName,
				LogLine:       logLine,
			})
		}
	}

	// Sort by nanosecond timestamp (oldest first) for precise ordering
	// Loki returns in forward direction (oldest first)
	// Use TimestampNs (string) instead of Timestamp to preserve nanosecond precision
	sort.Slice(entries, func(i, j int) bool {
		// Parse nanosecond timestamps as int64 for comparison
		iNs, _ := strconv.ParseInt(entries[i].TimestampNs, 10, 64)
		jNs, _ := strconv.ParseInt(entries[j].TimestampNs, 10, 64)
		return iNs < jNs
	})

	c.logger.Info(ctx, "Application log forward query completed",
		zap.String("project_slug", projectSlug),
		zap.Int("total_entries", len(entries)),
	)

	return entries, nil
}

// QueryApplicationLogsHistoryRaw queries historical logs as raw Loki JSON stream (backward direction)
func (c *lokiClient) QueryApplicationLogsHistoryRaw(ctx context.Context, projectSlug string, before time.Time, limit int) (io.ReadCloser, error) {
	// Build LogQL query
	// Match all revisions (e.g., projectSlug-00001, projectSlug-00002) and exclude sidecar containers
	query := fmt.Sprintf(`{namespace="application",app=~"%s-[0-9]+",container!~"queue-proxy|istio-proxy|nginx-proxy"}`, projectSlug)

	// Determine time range
	end := before
	if end.IsZero() {
		end = time.Now()
	}

	// 24-hour limit
	start := end.Add(-24 * time.Hour)

	// Build URL
	params := url.Values{}
	params.Add("query", query)
	params.Add("direction", "backward") // Latest first
	params.Add("limit", strconv.Itoa(limit))
	params.Add("start", fmt.Sprintf("%d", start.UnixNano()))
	params.Add("end", fmt.Sprintf("%d", end.UnixNano()))

	fullURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", c.baseURL, params.Encode())

	c.logger.Info(ctx, "Querying Loki for raw application log history",
		zap.String("project_slug", projectSlug),
		zap.Time("start", start),
		zap.Time("end", end),
		zap.Int("limit", limit),
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	if c.orgID != "" {
		req.Header.Set("X-Scope-OrgID", c.orgID)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(ctx, "Failed to query Loki",
			zap.String("url", fullURL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to query Loki: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error(ctx, "Loki returned non-OK status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("loki returned status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info(ctx, "Successfully retrieved raw application log history from Loki",
		zap.String("project_slug", projectSlug),
	)

	// Return response body as ReadCloser (caller must close it)
	return resp.Body, nil
}

// QueryApplicationLogsAfterRaw queries logs as raw Loki JSON stream (forward direction)
func (c *lokiClient) QueryApplicationLogsAfterRaw(ctx context.Context, projectSlug string, after time.Time, limit int) (io.ReadCloser, error) {
	// Build LogQL query
	// Match all revisions (e.g., projectSlug-00001, projectSlug-00002) and exclude sidecar containers
	query := fmt.Sprintf(`{namespace="application",app=~"%s-[0-9]+",container!~"queue-proxy|istio-proxy|nginx-proxy"}`, projectSlug)

	// Determine time range
	start := after
	end := time.Now()

	// Build URL
	params := url.Values{}
	params.Add("query", query)
	params.Add("direction", "forward") // Forward direction for newer logs
	params.Add("limit", strconv.Itoa(limit))
	params.Add("start", fmt.Sprintf("%d", start.UnixNano()))
	params.Add("end", fmt.Sprintf("%d", end.UnixNano()))

	fullURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", c.baseURL, params.Encode())

	c.logger.Info(ctx, "Querying Loki for raw newer application logs (forward)",
		zap.String("project_slug", projectSlug),
		zap.Time("after", after),
		zap.Time("end", end),
		zap.Int("limit", limit),
	)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	if c.orgID != "" {
		req.Header.Set("X-Scope-OrgID", c.orgID)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error(ctx, "Failed to query Loki",
			zap.String("url", fullURL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to query Loki: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error(ctx, "Loki returned non-OK status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("loki returned status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info(ctx, "Successfully retrieved raw application logs (forward) from Loki",
		zap.String("project_slug", projectSlug),
	)

	// Return response body as ReadCloser (caller must close it)
	return resp.Body, nil
}
