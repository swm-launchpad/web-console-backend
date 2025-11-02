package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/container/application"
	containerservice "github.com/swm-launchpad/web-console-backend/internal/container/domain/service"
)

// WebSocketError represents a structured error message for WebSocket
type WebSocketError struct {
	Type      string `json:"type"`      // "error"
	Code      string `json:"code"`      // Error code (e.g., "BUILD_HISTORY_NOT_FOUND")
	Message   string `json:"message"`   // Human-readable error message
	Retryable bool   `json:"retryable"` // Whether the client should retry
}

// BuildLogHandler handles build log related HTTP requests
type BuildLogHandler struct {
	createBuildLogTokenUC *application.CreateBuildLogTokenUseCase
	streamBuildLogsUC     *application.StreamBuildLogsUseCase
	getBuildLogHistoryUC  *application.GetBuildLogHistoryUseCase
	containerService      containerservice.ContainerService
	jwtUtil               *jwt.JWTUtil
	logger                logger.Logger
	upgrader              websocket.Upgrader
}

// NewBuildLogHandler creates a new BuildLogHandler instance
func NewBuildLogHandler(
	createBuildLogTokenUC *application.CreateBuildLogTokenUseCase,
	streamBuildLogsUC *application.StreamBuildLogsUseCase,
	getBuildLogHistoryUC *application.GetBuildLogHistoryUseCase,
	containerService containerservice.ContainerService,
	jwtUtil *jwt.JWTUtil,
	log logger.Logger,
) *BuildLogHandler {
	return &BuildLogHandler{
		createBuildLogTokenUC: createBuildLogTokenUC,
		streamBuildLogsUC:     streamBuildLogsUC,
		getBuildLogHistoryUC:  getBuildLogHistoryUC,
		containerService:      containerService,
		jwtUtil:               jwtUtil,
		logger:                log,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking based on CORS config
				return true
			},
		},
	}
}

// CreateBuildLogTokenRequest represents the request to create a build log token
// No body is needed - containerID comes from URL slug
type CreateBuildLogTokenRequest struct{}

// CreateBuildLogTokenResponse represents the response containing the build log token
type CreateBuildLogTokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"` // ISO 8601 format
}

// CreateBuildLogToken handles POST /api/v1/containers/:slug/build-log-token
// @Summary Create build log access token
// @Description Creates a short-lived token (30 minutes) for accessing build logs via WebSocket
// @Tags containers
// @Accept json
// @Produce json
// @Param slug path string true "Container slug"
// @Success 200 {object} CreateBuildLogTokenResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/containers/{slug}/build-log-token [post]
func (h *BuildLogHandler) CreateBuildLogToken(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated user ID
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container by slug
	slug := c.Param("slug")
	if slug == "" {
		h.logger.Warn(ctx, "Container slug is missing")
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.containerService.GetContainerBySlug(ctx, slug)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get container by slug",
			zap.String("slug", slug),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Execute use case
	input := application.CreateBuildLogTokenInput{
		UserID:      userID.(uint),
		ContainerID: container.ContainerID(),
	}

	output, err := h.createBuildLogTokenUC.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "Failed to create build log token",
			zap.Uint("user_id", input.UserID),
			zap.Uint("container_id", input.ContainerID),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Return response
	resp := CreateBuildLogTokenResponse{
		Token:     output.Token,
		ExpiresAt: output.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"), // ISO 8601
	}

	h.logger.Info(ctx, "Build log token created successfully",
		zap.Uint("user_id", input.UserID),
		zap.Uint("container_id", input.ContainerID),
		zap.String("expires_at", resp.ExpiresAt),
	)

	response.OK(c, resp)
}

// StreamBuildLogs handles GET /api/v1/containers/:slug/build-logs/stream
// @Summary Stream build logs via WebSocket
// @Description Streams build logs from Loki via WebSocket. Requires a valid build log token as query parameter.
// @Tags containers
// @Param slug path string true "Container slug"
// @Param token query string true "Build log token (obtained from /build-log-token endpoint)"
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/containers/{slug}/build-logs/stream [get]
func (h *BuildLogHandler) StreamBuildLogs(c *gin.Context) {
	ctx := c.Request.Context()

	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		h.logger.Warn(ctx, "Build log token is missing")
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Validate build log token
	claims, err := h.jwtUtil.ValidateBuildLogToken(ctx, token)
	if err != nil {
		h.logger.Warn(ctx, "Invalid build log token", zap.Error(err))
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container by slug
	slug := c.Param("slug")
	if slug == "" {
		h.logger.Warn(ctx, "Container slug is missing")
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.containerService.GetContainerBySlug(ctx, slug)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get container by slug",
			zap.String("slug", slug),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Verify token is for this container
	if claims.ContainerID != container.ContainerID() {
		h.logger.Warn(ctx, "Token container ID does not match requested container",
			zap.Uint("token_container_id", claims.ContainerID),
			zap.Uint("request_container_id", container.ContainerID()),
		)
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Upgrade to WebSocket
	wsConn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error(ctx, "Failed to upgrade to WebSocket",
			zap.Uint("container_id", container.ContainerID()),
			zap.Error(err),
		)
		return
	}
	defer func() {
		_ = wsConn.Close()
	}()

	h.logger.Info(ctx, "WebSocket connection established for build logs",
		zap.Uint("user_id", claims.UserID),
		zap.Uint("container_id", container.ContainerID()),
	)

	// Create context for managing goroutines
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Stream build logs from Loki
	input := application.StreamBuildLogsInput{
		ContainerID: container.ContainerID(),
	}

	lokiStream, err := h.streamBuildLogsUC.Execute(streamCtx, input)
	if err != nil {
		h.logger.Error(ctx, "Failed to stream build logs",
			zap.Uint("container_id", container.ContainerID()),
			zap.Error(err),
		)

		// Send structured error message via WebSocket
		errorMsg := WebSocketError{
			Type: "error",
		}

		// Determine error code and message based on error type
		if err.Error() == "build history not found" {
			errorMsg.Code = "BUILD_HISTORY_NOT_FOUND"
			errorMsg.Message = "빌드 파이프라인이 생성되는 중입니다"
			errorMsg.Retryable = true
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

	h.logger.Info(ctx, "Started streaming build logs from Loki",
		zap.Uint("container_id", container.ContainerID()),
	)

	// Use WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Goroutine 1: Read from Loki and write to WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel() // Cancel context when this goroutine exits

		// Read from Loki stream and write to WebSocket
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			select {
			case <-streamCtx.Done():
				h.logger.Info(streamCtx, "Loki to WebSocket stream cancelled")
				return
			default:
			}

			n, err := lokiStream.Read(buf)
			if n > 0 {
				// Write to WebSocket
				if err := wsConn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
					h.logger.Warn(streamCtx, "Failed to write to WebSocket",
						zap.Uint("container_id", container.ContainerID()),
						zap.Error(err),
					)
					return
				}
			}

			if err != nil {
				if err == io.EOF {
					h.logger.Info(streamCtx, "Loki stream ended",
						zap.Uint("container_id", container.ContainerID()),
					)
				} else {
					h.logger.Error(streamCtx, "Error reading from Loki stream",
						zap.Uint("container_id", container.ContainerID()),
						zap.Error(err),
					)

					// Send error message to Frontend
					errorMsg := WebSocketError{
						Type:      "error",
						Code:      "LOKI_STREAM_ERROR",
						Message:   "로그 스트림 중 오류가 발생했습니다",
						Retryable: true,
					}

					// Special handling for Loki concurrent limit error
					if strings.Contains(err.Error(), "max concurrent tail requests") {
						errorMsg.Code = "LOKI_CONCURRENT_LIMIT_EXCEEDED"
						errorMsg.Message = "서버가 혼잡합니다. 잠시 후 다시 시도해주세요."
					}

					// Send error as JSON to frontend
					if errorJSON, marshalErr := json.Marshal(errorMsg); marshalErr == nil {
						_ = wsConn.WriteMessage(websocket.TextMessage, errorJSON)
					}
				}
				return
			}
		}
	}()

	// Goroutine 2: Monitor WebSocket connection for client disconnection
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel() // Cancel context when this goroutine exits

		for {
			select {
			case <-streamCtx.Done():
				return
			default:
			}

			// Read from WebSocket to detect disconnection
			// We don't expect any messages from client (read-only stream)
			_, _, err := wsConn.ReadMessage()
			if err != nil {
				// Close code 1005 (No Status Received)도 정상 종료로 간주
				// React 컴포넌트 빠른 언마운트 시 close handshake 미완료는 예상된 동작
				if websocket.IsCloseError(err,
					websocket.CloseNormalClosure,    // 1000
					websocket.CloseGoingAway,        // 1001
					websocket.CloseNoStatusReceived, // 1005
				) {
					h.logger.Info(streamCtx, "WebSocket connection closed by client",
						zap.Uint("container_id", container.ContainerID()),
					)
				} else {
					// 진짜 비정상 종료만 경고
					h.logger.Warn(streamCtx, "WebSocket connection closed unexpectedly",
						zap.Uint("container_id", container.ContainerID()),
						zap.Error(err),
					)
				}
				return
			}
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	h.logger.Info(ctx, "Build log streaming completed",
		zap.Uint("container_id", container.ContainerID()),
	)
}

// GetBuildLogHistory handles GET /api/v1/containers/:slug/build-logs/history
// @Summary Get historical build logs via HTTP
// @Description Retrieves historical build logs for the latest completed build from Loki via HTTP. Requires standard authentication.
// @Tags containers
// @Param slug path string true "Container slug"
// @Success 200 {object} object "Loki query_range response (JSON)"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /api/v1/containers/{slug}/build-logs/history [get]
func (h *BuildLogHandler) GetBuildLogHistory(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated user ID from context (set by RequireAuth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "User ID not found in context")
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	// Get container by slug
	slug := c.Param("slug")
	if slug == "" {
		h.logger.Warn(ctx, "Container slug is missing")
		response.Error(c, auth.ErrUnauthorized, mapContainerError)
		return
	}

	container, err := h.containerService.GetContainerBySlug(ctx, slug)
	if err != nil {
		h.logger.Warn(ctx, "Failed to get container by slug",
			zap.String("slug", slug),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}

	// Get historical build logs with permission check
	input := application.GetBuildLogHistoryInput{
		UserID:      userID.(uint),
		ContainerID: container.ContainerID(),
	}

	lokiResponse, err := h.getBuildLogHistoryUC.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "Failed to get build log history",
			zap.Uint("container_id", container.ContainerID()),
			zap.Error(err),
		)
		response.Error(c, err, mapContainerError)
		return
	}
	defer func() {
		_ = lokiResponse.Close()
	}()

	h.logger.Info(ctx, "Streaming historical build logs to client",
		zap.Uint("user_id", userID.(uint)),
		zap.Uint("container_id", container.ContainerID()),
	)

	// Set response headers for JSON streaming
	c.Header("Content-Type", "application/json")
	c.Header("Transfer-Encoding", "chunked")

	// Stream Loki response to client
	_, err = io.Copy(c.Writer, lokiResponse)
	if err != nil {
		h.logger.Error(ctx, "Failed to stream historical logs to client",
			zap.Uint("container_id", container.ContainerID()),
			zap.Error(err),
		)
		return
	}

	h.logger.Info(ctx, "Historical build logs sent successfully",
		zap.Uint("container_id", container.ContainerID()),
	)
}
