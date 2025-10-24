package logger

import "context"

// Context keys for logger metadata
type contextKey string

const (
	requestIDKey contextKey = "logger_request_id"
	userIDKey    contextKey = "logger_user_id"
)

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from context
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext extracts the user ID from context
func UserIDFromContext(ctx context.Context) uint {
	if ctx == nil {
		return 0
	}
	if userID, ok := ctx.Value(userIDKey).(uint); ok {
		return userID
	}
	return 0
}

// DetachContext creates a new context that preserves logger metadata (request_id, user_id)
// but is detached from the parent context's cancellation and deadline.
// This is useful for background goroutines that should continue even after the original request completes.
func DetachContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	// Preserve request_id if present
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		newCtx = WithRequestID(newCtx, requestID)
	}

	// Preserve user_id if present
	if userID := UserIDFromContext(ctx); userID != 0 {
		newCtx = WithUserID(newCtx, userID)
	}

	return newCtx
}
