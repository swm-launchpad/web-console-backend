package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/project/application"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/infrastructure/repository"
)

// WebSocketError represents a structured error message for WebSocket
type WebSocketError struct {
	Type      string `json:"type"`      // "error"
	Code      string `json:"code"`      // Error code
	Message   string `json:"message"`   // Human-readable error message
	Retryable bool   `json:"retryable"` // Whether the client should retry
}

// ProjectLogHandler handles project application log related HTTP requests
type ProjectLogHandler struct {
	createTokenUC *application.CreateProjectLogTokenUseCase
	streamLogsUC  *application.StreamProjectLogsUseCase
	historyUC     *application.GetProjectLogHistoryUseCase
	projectRepo   repository.ProjectRepository
	jwtSecret     string
	logger        logger.Logger
	upgrader      websocket.Upgrader
}

// NewProjectLogHandler creates a new ProjectLogHandler instance
func NewProjectLogHandler(
	createTokenUC *application.CreateProjectLogTokenUseCase,
	streamLogsUC *application.StreamProjectLogsUseCase,
	historyUC *application.GetProjectLogHistoryUseCase,
	projectRepo repository.ProjectRepository,
	jwtSecret string,
	log logger.Logger,
) *ProjectLogHandler {
	return &ProjectLogHandler{
		createTokenUC: createTokenUC,
		streamLogsUC:  streamLogsUC,
		historyUC:     historyUC,
		projectRepo:   projectRepo,
		jwtSecret:     jwtSecret,
		logger:        log,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		},
	}
}

// CreateProjectLogToken handles POST /api/v1/projects/:slug/log-token
// Creates a JWT token for accessing project logs via WebSocket
func (h *ProjectLogHandler) CreateProjectLogToken(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")
	userID := c.GetUint("user_id")

	h.logger.Info(ctx, "Creating project log token",
		zap.String("slug", slug),
		zap.Uint("user_id", userID),
	)

	// Find project by slug (permission check)
	project, err := h.projectRepo.FindBySlug(ctx, slug)
	if err != nil {
		h.logger.Error(ctx, "Project not found for log token",
			zap.String("slug", slug),
			zap.Error(err),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	// Execute use case
	input := application.CreateProjectLogTokenInput{
		ProjectID: project.ProjectID(),
		UserID:    userID,
	}

	output, err := h.createTokenUC.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "Failed to create project log token",
			zap.String("slug", slug),
			zap.Error(err),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "Project log token created successfully",
		zap.String("slug", slug),
		zap.Uint("project_id", project.ProjectID()),
	)

	response.OK(c, output)
}

// StreamProjectLogs handles GET /api/v1/projects/:slug/logs/stream (WebSocket)
// Streams real-time logs from running application pods
func (h *ProjectLogHandler) StreamProjectLogs(c *gin.Context) {
	ctx := c.Request.Context()

	h.logger.Info(ctx, "Project log stream request received")

	// 1. Validate JWT token from query parameter
	tokenString := c.Query("token")
	if tokenString == "" {
		h.logger.Error(ctx, "Missing token in query parameter")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		h.logger.Error(ctx, "Invalid token", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		h.logger.Error(ctx, "Invalid token claims")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		return
	}

	projectID := uint(claims["project_id"].(float64))
	userID := uint(claims["user_id"].(float64))

	h.logger.Info(ctx, "Token validated for project log streaming",
		zap.Uint("project_id", projectID),
		zap.Uint("user_id", userID),
	)

	// 2. Upgrade to WebSocket
	wsConn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error(ctx, "Failed to upgrade to WebSocket", zap.Error(err))
		return
	}

	// Initial cleanup (in case of early exit before main defer)
	defer func() {
		_ = wsConn.Close()
	}()

	h.logger.Info(ctx, "WebSocket connection established for project logs",
		zap.Uint("project_id", projectID),
	)

	// 3. Create context for managing goroutines
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 4. Execute use case to start streaming
	input := application.StreamProjectLogsInput{
		ProjectID: projectID,
	}

	lokiStream, err := h.streamLogsUC.Execute(streamCtx, input)
	if err != nil {
		h.logger.Error(ctx, "Failed to stream project logs",
			zap.Uint("project_id", projectID),
			zap.Error(err),
		)

		// Send structured error message via WebSocket
		errorMsg := WebSocketError{
			Type: "error",
		}

		// Determine error code and message based on error type
		if err.Error() == "no pods running" {
			errorMsg.Code = "NO_PODS_RUNNING"
			errorMsg.Message = "실행 중인 Pod가 없습니다"
			errorMsg.Retryable = false
		} else {
			errorMsg.Code = "STREAM_ERROR"
			errorMsg.Message = err.Error()
			errorMsg.Retryable = false
		}

		// Marshal error to JSON
		errorJSON, marshalErr := json.Marshal(errorMsg)
		if marshalErr != nil {
			h.logger.Error(ctx, "Failed to marshal error message", zap.Error(marshalErr))
			_ = wsConn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
			return
		}

		_ = wsConn.WriteMessage(websocket.TextMessage, errorJSON)
		return
	}
	defer func() {
		_ = lokiStream.Close()
	}()

	h.logger.Info(ctx, "Started streaming project logs from Loki",
		zap.Uint("project_id", projectID),
	)

	// ⚠️ CRITICAL: Defer cleanup sequence (LIFO order)
	// This pattern follows build_log_handler.go (lines 228-434) to prevent goroutine leaks
	// 1. Force any blocked ReadMessage() to timeout immediately
	// 2. Close Loki stream (interrupts any blocked Read())
	// 3. Send proper WebSocket close frame
	// 4. Close WebSocket connection
	defer func() {
		// Step 1: Force WebSocket reads to timeout
		_ = wsConn.SetReadDeadline(time.Now())

		// Step 2: Close Loki stream
		_ = lokiStream.Close()

		// Step 3: Send proper close frame (best effort)
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		writeDeadline := time.Now().Add(time.Second)
		_ = wsConn.WriteControl(websocket.CloseMessage, closeMsg, writeDeadline)

		// Step 4: Close WebSocket connection
		_ = wsConn.Close()

		h.logger.Info(streamCtx, "Project log streaming completed",
			zap.Uint("project_id", projectID),
		)
	}()

	// Message types for channel communication
	type lokiMessage struct {
		data []byte
		err  error
	}
	type wsMessage struct {
		err error
	}

	// Buffered channels prevent goroutine blocking on send
	lokiChan := make(chan lokiMessage, 1)
	wsChan := make(chan wsMessage, 1)

	// Inner goroutine 1: Read from Loki stream
	go func() {
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := lokiStream.Read(buf)

			// Copy data to avoid buffer reuse issues
			data := make([]byte, n)
			copy(data, buf[:n])

			// Send result to channel (non-blocking with context check)
			select {
			case lokiChan <- lokiMessage{data, err}:
				if err != nil {
					return // Exit on error (including EOF)
				}
			case <-streamCtx.Done():
				return // Exit on context cancellation
			}
		}
	}()

	// Inner goroutine 2: Monitor WebSocket for client disconnection
	go func() {
		for {
			// ReadMessage blocks until message received or connection closed
			_, _, err := wsConn.ReadMessage()

			// Send result to channel (non-blocking with context check)
			select {
			case wsChan <- wsMessage{err}:
				return // Always exit after first message/error
			case <-streamCtx.Done():
				return // Exit on context cancellation
			}
		}
	}()

	// Main coordination loop (blocking select - no default case)
	// This blocks the handler until streaming completes, preventing premature return
	for {
		select {
		case <-streamCtx.Done():
			// Context cancelled - cleanup via defers
			h.logger.Info(streamCtx, "Stream context cancelled")
			return

		case lokiMsg := <-lokiChan:
			// Message from Loki stream
			if lokiMsg.err != nil {
				if lokiMsg.err == io.EOF {
					h.logger.Info(streamCtx, "Loki stream ended",
						zap.Uint("project_id", projectID),
					)
				} else {
					h.logger.Error(streamCtx, "Error reading from Loki stream",
						zap.Uint("project_id", projectID),
						zap.Error(lokiMsg.err),
					)

					// Send error message to frontend
					errorMsg := WebSocketError{
						Type:      "error",
						Code:      "LOKI_STREAM_ERROR",
						Message:   "로그 스트림 중 오류가 발생했습니다",
						Retryable: true,
					}

					// Special handling for Loki concurrent limit error
					if strings.Contains(lokiMsg.err.Error(), "max concurrent tail requests") {
						errorMsg.Code = "LOKI_CONCURRENT_LIMIT_EXCEEDED"
						errorMsg.Message = "서버가 혼잡합니다. 잠시 후 다시 시도해주세요."
					}

					// Send error as JSON to frontend (best effort)
					if errorJSON, marshalErr := json.Marshal(errorMsg); marshalErr == nil {
						_ = wsConn.WriteMessage(websocket.TextMessage, errorJSON)
					}
				}
				return // Cleanup via defers
			}

			// Write data to WebSocket
			if len(lokiMsg.data) > 0 {
				if err := wsConn.WriteMessage(websocket.TextMessage, lokiMsg.data); err != nil {
					h.logger.Warn(streamCtx, "Failed to write to WebSocket",
						zap.Uint("project_id", projectID),
						zap.Error(err),
					)
					return // Cleanup via defers
				}
			}

		case wsMsg := <-wsChan:
			// Message from WebSocket (client disconnection)
			if wsMsg.err != nil {
				// Close code 1005 (No Status Received) is also considered normal
				// Quick React component unmount may result in incomplete close handshake
				if websocket.IsCloseError(wsMsg.err,
					websocket.CloseNormalClosure,    // 1000
					websocket.CloseGoingAway,        // 1001
					websocket.CloseNoStatusReceived, // 1005
				) {
					h.logger.Info(streamCtx, "WebSocket connection closed by client",
						zap.Uint("project_id", projectID),
					)
				} else {
					// Only warn on truly abnormal closures
					h.logger.Warn(streamCtx, "WebSocket connection closed unexpectedly",
						zap.Uint("project_id", projectID),
						zap.Error(wsMsg.err),
					)
				}
			}
			return // Cleanup via defers
		}
	}
}

// GetProjectLogHistory handles GET /api/v1/projects/:slug/logs/history
// Returns historical logs with pagination support
func (h *ProjectLogHandler) GetProjectLogHistory(c *gin.Context) {
	ctx := c.Request.Context()
	slug := c.Param("slug")

	h.logger.Info(ctx, "Project log history request received",
		zap.String("slug", slug),
	)

	// Query parameters
	beforeStr := c.Query("before")
	afterStr := c.Query("after")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		h.logger.Error(ctx, "Invalid limit parameter",
			zap.String("limit", limitStr),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	// Find project by slug (permission check)
	project, err := h.projectRepo.FindBySlug(ctx, slug)
	if err != nil {
		h.logger.Error(ctx, "Project not found for log history",
			zap.String("slug", slug),
			zap.Error(err),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	// Parse before timestamp
	var before time.Time
	if beforeStr != "" {
		before, err = time.Parse(time.RFC3339Nano, beforeStr)
		if err != nil {
			h.logger.Error(ctx, "Invalid before timestamp",
				zap.String("before", beforeStr),
				zap.Error(err),
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid before timestamp"})
			return
		}
	}

	// Parse after timestamp (for forward pagination)
	var after time.Time
	if afterStr != "" {
		after, err = time.Parse(time.RFC3339Nano, afterStr)
		if err != nil {
			h.logger.Error(ctx, "Invalid after timestamp",
				zap.String("after", afterStr),
				zap.Error(err),
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid after timestamp"})
			return
		}
	}

	h.logger.Info(ctx, "Executing project log history query",
		zap.String("slug", slug),
		zap.Uint("project_id", project.ProjectID()),
		zap.Time("before", before),
		zap.Time("after", after),
		zap.Int("limit", limit),
	)

	// Execute use case
	input := application.GetProjectLogHistoryInput{
		ProjectID: project.ProjectID(),
		Before:    before,
		After:     after,
		Limit:     limit,
	}

	output, err := h.historyUC.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "Failed to get project log history",
			zap.String("slug", slug),
			zap.Uint("project_id", project.ProjectID()),
			zap.Error(err),
		)
		response.Error(c, err, mapProjectError)
		return
	}

	h.logger.Info(ctx, "Project log history query completed",
		zap.String("slug", slug),
		zap.Uint("project_id", project.ProjectID()),
		zap.Int("log_count", len(output.Logs)),
	)

	response.OK(c, output)
}
