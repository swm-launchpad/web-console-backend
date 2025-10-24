package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface for application logging
type Logger interface {
	Debug(ctx context.Context, msg string, fields ...zap.Field)
	Info(ctx context.Context, msg string, fields ...zap.Field)
	Warn(ctx context.Context, msg string, fields ...zap.Field)
	Error(ctx context.Context, msg string, fields ...zap.Field)
	Fatal(ctx context.Context, msg string, fields ...zap.Field)
	With(fields ...zap.Field) Logger
	Sync() error
}

// zapLogger is the Zap implementation of Logger
type zapLogger struct {
	logger *zap.Logger
}

// New creates a new logger instance based on the configuration
func New(cfg Config) (Logger, error) {
	var zapCfg zap.Config

	// Set log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Configure based on environment
	if cfg.Format == "json" {
		// Production configuration with JSON output
		zapCfg = zap.NewProductionConfig()
		zapCfg.Level = zap.NewAtomicLevelAt(level)
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapCfg.EncoderConfig.CallerKey = "caller"
		zapCfg.EncoderConfig.MessageKey = "message"
		zapCfg.EncoderConfig.LevelKey = "level"
		zapCfg.EncoderConfig.StacktraceKey = "stacktrace"
	} else {
		// Development configuration with console output
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.Level = zap.NewAtomicLevelAt(level)
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Configure output paths
	zapCfg.OutputPaths = []string{"stdout"}
	zapCfg.ErrorOutputPaths = []string{"stderr"}

	// Add file output if specified
	if cfg.FilePath != "" {
		zapCfg.OutputPaths = append(zapCfg.OutputPaths, cfg.FilePath)
	}

	// Build the logger
	l, err := zapCfg.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1), // Skip the wrapper methods
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	return &zapLogger{logger: l}, nil
}

// NewForTest creates a no-op logger for testing
func NewForTest() Logger {
	return &zapLogger{logger: zap.NewNop()}
}

// Debug logs a debug message
func (l *zapLogger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.extractContextFields(ctx, fields)
	l.logger.Debug(msg, fields...)
}

// Info logs an info message
func (l *zapLogger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.extractContextFields(ctx, fields)
	l.logger.Info(msg, fields...)
}

// Warn logs a warning message
func (l *zapLogger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.extractContextFields(ctx, fields)
	l.logger.Warn(msg, fields...)
}

// Error logs an error message
func (l *zapLogger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.extractContextFields(ctx, fields)
	l.logger.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *zapLogger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	fields = l.extractContextFields(ctx, fields)
	l.logger.Fatal(msg, fields...)
}

// With creates a child logger with the given fields
func (l *zapLogger) With(fields ...zap.Field) Logger {
	return &zapLogger{logger: l.logger.With(fields...)}
}

// Sync flushes any buffered log entries
func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

// extractContextFields extracts logger fields from context
func (l *zapLogger) extractContextFields(ctx context.Context, fields []zap.Field) []zap.Field {
	// Extract request ID from context if available
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		fields = append([]zap.Field{zap.String("request_id", requestID)}, fields...)
	}

	// Extract user ID from context if available
	if userID := UserIDFromContext(ctx); userID != 0 {
		fields = append([]zap.Field{zap.Uint("user_id", userID)}, fields...)
	}

	return fields
}

// Global logger instance (initialized on startup)
var globalLogger Logger

// InitGlobal initializes the global logger
func InitGlobal(cfg Config) error {
	l, err := New(cfg)
	if err != nil {
		return err
	}
	globalLogger = l
	return nil
}

// GetGlobal returns the global logger instance
func GetGlobal() Logger {
	if globalLogger == nil {
		// Fallback to a basic logger if not initialized
		l, _ := New(Config{
			Level:  "info",
			Format: "console",
		})
		globalLogger = l
	}
	return globalLogger
}

// CloseGlobal closes the global logger
func CloseGlobal() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// Helper functions for backward compatibility with standard log package

// Debug logs a debug message using the global logger
func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	GetGlobal().Debug(ctx, msg, fields...)
}

// Info logs an info message using the global logger
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	GetGlobal().Info(ctx, msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	GetGlobal().Warn(ctx, msg, fields...)
}

// Error logs an error message using the global logger
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	GetGlobal().Error(ctx, msg, fields...)
}

// Fatal logs a fatal message and exits using the global logger
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	GetGlobal().Fatal(ctx, msg, fields...)
	os.Exit(1)
}
