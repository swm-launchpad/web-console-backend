package logger

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LoggingMiddleware creates a Gin middleware for request logging
type LoggingMiddleware struct {
	logger Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Handler returns the Gin middleware handler
func (m *LoggingMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for existing request ID from upstream (e.g., load balancer, API gateway)
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate new request ID only if not provided
			requestID = uuid.New().String()
		}

		// Add request ID to context
		ctx := WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// Add request ID to response header for client tracking
		c.Header("X-Request-ID", requestID)

		// Start timer
		start := time.Now()

		// Log request start
		m.logger.Info(ctx, "request started",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		// Process request
		c.Next()

		// Get updated context after middleware chain (includes auth metadata)
		ctx = c.Request.Context()

		// Calculate duration
		duration := time.Since(start)

		// Determine log level based on status code
		statusCode := c.Writer.Status()
		logFunc := m.logger.Info
		if statusCode >= 500 {
			logFunc = m.logger.Error
		} else if statusCode >= 400 {
			logFunc = m.logger.Warn
		}

		// Log request completion
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.Int("response_size", c.Writer.Size()),
		}

		// Add error if present
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		logFunc(ctx, "request completed", fields...)
	}
}

// RecoveryMiddleware creates a Gin middleware for panic recovery with logging
func (m *LoggingMiddleware) RecoveryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx := c.Request.Context()

				// Log the panic
				m.logger.Error(ctx, "panic recovered",
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.Stack("stacktrace"),
				)

				// Abort with 500 status
				c.AbortWithStatus(500)
			}
		}()

		c.Next()
	}
}
