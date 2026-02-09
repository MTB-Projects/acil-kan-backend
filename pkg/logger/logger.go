package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// InitLogger initializes the global logger
func InitLogger(env string) error {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Set output paths
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	logger, err := config.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return err
	}

	Log = logger
	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

// WithContext adds context fields to logger
func WithContext(ctx context.Context) *zap.Logger {
	return Log.With(zap.String("trace_id", getTraceID(ctx)))
}

// WithFields creates a logger with additional fields
func WithFields(fields ...zap.Field) *zap.Logger {
	return Log.With(fields...)
}

// HTTPMiddleware returns a Fiber middleware for logging HTTP requests
func HTTPMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		
		// Generate trace ID for this request
		traceID := generateTraceID()
		c.Locals("trace_id", traceID)

		// Log incoming request
		Log.Info("Incoming request",
			zap.String("trace_id", traceID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.String("ip", c.IP()),
			zap.String("user_agent", c.Get("User-Agent")),
		)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log response
		if err != nil {
			Log.Error("Request failed",
				zap.String("trace_id", traceID),
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", c.Response().StatusCode()),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
			return err
		}

		Log.Info("Request completed",
			zap.String("trace_id", traceID),
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("duration", duration),
		)

		return nil
	}
}

// LogUserAction logs user-specific actions
func LogUserAction(userUID, action string, fields ...zap.Field) {
	allFields := append([]zap.Field{
		zap.String("user_uid", userUID),
		zap.String("action", action),
		zap.Time("timestamp", time.Now()),
	}, fields...)
	
	Log.Info("User action", allFields...)
}

// LogDatabaseOperation logs database operations
func LogDatabaseOperation(operation, collection string, duration time.Duration, err error) {
	fields := []zap.Field{
		zap.String("operation", operation),
		zap.String("collection", collection),
		zap.Duration("duration", duration),
	}

	if err != nil {
		Log.Error("Database operation failed", append(fields, zap.Error(err))...)
	} else {
		Log.Debug("Database operation completed", fields...)
	}
}

// LogNotification logs notification sending
func LogNotification(notificationType, recipient string, success bool, err error) {
	fields := []zap.Field{
		zap.String("type", notificationType),
		zap.String("recipient", recipient),
		zap.Bool("success", success),
	}

	if err != nil {
		Log.Error("Notification failed", append(fields, zap.Error(err))...)
	} else {
		Log.Info("Notification sent", fields...)
	}
}

func getTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}
	return ""
}

func generateTraceID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}
