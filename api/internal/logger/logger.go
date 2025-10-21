package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger and provides structured logging capabilities.
type Logger struct {
	zlog zerolog.Logger
}

// New creates a new Logger instance configured for the given environment.
// In development mode, it outputs pretty-printed colored logs.
// In production mode, it outputs JSON formatted logs.
func New(env string) *Logger {
	var output io.Writer

	if env == "development" {
		// Pretty console output for development
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	} else {
		// JSON output for production
		output = os.Stdout
	}

	// Configure global settings
	zerolog.TimeFieldFormat = time.RFC3339

	// Set log level based on environment
	level := zerolog.InfoLevel
	if env == "development" {
		level = zerolog.DebugLevel
	}

	// Create logger
	zlog := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{zlog: zlog}
}

// Debug logs a debug message with optional fields.
func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	event := l.zlog.Debug()
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	event.Msg(msg)
}

// Info logs an info message with optional fields.
func (l *Logger) Info(msg string, fields map[string]interface{}) {
	event := l.zlog.Info()
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	event.Msg(msg)
}

// Warn logs a warning message with optional fields.
func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	event := l.zlog.Warn()
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	event.Msg(msg)
}

// Error logs an error message with an error and optional fields.
func (l *Logger) Error(msg string, err error, fields map[string]interface{}) {
	event := l.zlog.Error().Err(err)
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	event.Msg(msg)
}

// Fatal logs a fatal message and exits the application.
func (l *Logger) Fatal(msg string, err error, fields map[string]interface{}) {
	event := l.zlog.Fatal().Err(err)
	for key, value := range fields {
		event = event.Interface(key, value)
	}
	event.Msg(msg)
}

// With creates a child logger with additional context fields.
// This is useful for adding request IDs or other contextual information.
func (l *Logger) With(fields map[string]interface{}) *Logger {
	ctx := l.zlog.With()
	for key, value := range fields {
		ctx = ctx.Interface(key, value)
	}
	return &Logger{zlog: ctx.Logger()}
}

// WithRequestID creates a child logger with a request ID field.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		zlog: l.zlog.With().Str("request_id", requestID).Logger(),
	}
}

// GetZerolog returns the underlying zerolog.Logger for advanced usage.
func (l *Logger) GetZerolog() *zerolog.Logger {
	return &l.zlog
}
