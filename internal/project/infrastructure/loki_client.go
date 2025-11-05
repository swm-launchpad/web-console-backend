package infrastructure

import (
	"bytes"
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
	"sync"
	"sync/atomic"
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
	conn, resp, err := dialer.DialContext(ctx, fullURL, headers)
	if err != nil {
		c.logger.Error(ctx, "Failed to connect to Loki WebSocket",
			zap.String("url", fullURL),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to Loki: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	c.logger.Info(ctx, "Loki WebSocket connected for application logs",
		zap.String("project_slug", projectSlug),
	)

	// Wrap with monitored reader that checks pod status every 30 seconds
	baseReader := newApplicationLogReader(conn, c.logger, ctx)
	return newMonitoredApplicationLogReader(baseReader, c.kubeClient, projectSlug, c.logger, ctx), nil
}

// applicationLogReader wraps WebSocket connection as io.ReadCloser
// It reads Loki tail messages and formats them with [container_name] prefix
type applicationLogReader struct {
	conn   *websocket.Conn
	buffer *bytes.Buffer
	logger logger.Logger
	ctx    context.Context
}

func newApplicationLogReader(conn *websocket.Conn, logger logger.Logger, ctx context.Context) *applicationLogReader {
	return &applicationLogReader{
		conn:   conn,
		buffer: &bytes.Buffer{},
		logger: logger,
		ctx:    ctx,
	}
}

func (r *applicationLogReader) Read(p []byte) (int, error) {
	// If buffer is empty, read next WebSocket message
	for r.buffer.Len() == 0 {
		_, message, err := r.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return 0, io.EOF
			}
			return 0, fmt.Errorf("failed to read from websocket: %w", err)
		}

		// Parse Loki tail response
		var tailResp struct {
			Streams []struct {
				Stream map[string]string `json:"stream"`
				Values [][]string        `json:"values"`
			} `json:"streams"`
		}

		if err := json.Unmarshal(message, &tailResp); err != nil {
			r.logger.Warn(r.ctx, "Failed to parse Loki tail response", zap.Error(err))
			continue
		}

		// Format each log line with [container_name] prefix
		for _, stream := range tailResp.Streams {
			containerName := stream.Stream["container"]
			if containerName == "" {
				containerName = "unknown"
			}

			for _, value := range stream.Values {
				// value[0] = timestamp (nanoseconds)
				// value[1] = log line
				timestampNano, _ := strconv.ParseInt(value[0], 10, 64)
				timestamp := time.Unix(0, timestampNano).Format(time.RFC3339)
				logLine := value[1]

				// Format: [container_name] timestamp log_line\n
				formatted := fmt.Sprintf("[%s] %s %s\n", containerName, timestamp, logLine)
				r.buffer.WriteString(formatted)
			}
		}
	}

	// Read from buffer
	return r.buffer.Read(p)
}

func (r *applicationLogReader) Close() error {
	r.logger.Info(r.ctx, "Closing application log WebSocket reader")
	return r.conn.Close()
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
				ContainerName: containerName,
				LogLine:       logLine,
			})
		}
	}

	// Sort by timestamp (oldest first)
	// Loki returns in backward direction (newest first), so we reverse it
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	c.logger.Info(ctx, "Application log history query completed",
		zap.String("project_slug", projectSlug),
		zap.Int("total_entries", len(entries)),
	)

	return entries, nil
}

// monitoredApplicationLogReader wraps applicationLogReader and monitors pod status
// Closes the connection automatically if pods die
type monitoredApplicationLogReader struct {
	reader      io.ReadCloser
	kubeClient  infrastructure.KubeClient
	projectSlug string
	logger      logger.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	closeOnce   sync.Once
	closed      atomic.Bool
}

func newMonitoredApplicationLogReader(
	reader io.ReadCloser,
	kubeClient infrastructure.KubeClient,
	projectSlug string,
	logger logger.Logger,
	parentCtx context.Context,
) *monitoredApplicationLogReader {
	// Create cancellable context for goroutine cleanup
	ctx, cancel := context.WithCancel(parentCtx)

	m := &monitoredApplicationLogReader{
		reader:      reader,
		kubeClient:  kubeClient,
		projectSlug: projectSlug,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start background goroutine to monitor pod status
	go m.monitorPodStatus()

	m.logger.Info(ctx, "Started pod monitoring for application logs",
		zap.String("project_slug", projectSlug),
	)

	return m
}

// monitorPodStatus checks pod status every 30 seconds and closes connection if pods die
func (m *monitoredApplicationLogReader) monitorPodStatus() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			// Context cancelled - cleanup goroutine
			m.logger.Info(m.ctx, "Pod monitoring goroutine stopped",
				zap.String("project_slug", m.projectSlug),
			)
			return

		case <-ticker.C:
			// Check if pods are still running
			podsRunning, err := m.kubeClient.CheckApplicationPodsRunning(m.ctx, m.projectSlug)
			if err != nil {
				m.logger.Warn(m.ctx, "Failed to check pod status during monitoring",
					zap.String("project_slug", m.projectSlug),
					zap.Error(err),
				)
				// Don't close on error - could be temporary network issue
				continue
			}

			if !podsRunning {
				m.logger.Info(m.ctx, "Application pods died - closing WebSocket",
					zap.String("project_slug", m.projectSlug),
				)
				// Close the connection gracefully
				_ = m.Close()
				return
			}

			m.logger.Debug(m.ctx, "Pod status check: running",
				zap.String("project_slug", m.projectSlug),
			)
		}
	}
}

func (m *monitoredApplicationLogReader) Read(p []byte) (int, error) {
	if m.closed.Load() {
		return 0, io.EOF
	}
	return m.reader.Read(p)
}

func (m *monitoredApplicationLogReader) Close() error {
	var err error
	m.closeOnce.Do(func() {
		m.closed.Store(true)
		m.logger.Info(m.ctx, "Closing monitored application log reader",
			zap.String("project_slug", m.projectSlug),
		)

		// Cancel context to stop monitoring goroutine
		m.cancel()

		// Close underlying reader
		err = m.reader.Close()
	})
	return err
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
				ContainerName: containerName,
				LogLine:       logLine,
			})
		}
	}

	// Loki returns in forward direction (oldest first)
	// Already in correct order, no need to reverse
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	c.logger.Info(ctx, "Application log forward query completed",
		zap.String("project_slug", projectSlug),
		zap.Int("total_entries", len(entries)),
	)

	return entries, nil
}
