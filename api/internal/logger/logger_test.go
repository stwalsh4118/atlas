package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestNew_DevelopmentMode(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	logger := New("development")

	// Restore stdout
	if err := w.Close(); err != nil {
		t.Errorf("Failed to close pipe writer: %v", err)
	}
	os.Stdout = old

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}
	if logger.GetZerolog() == nil {
		t.Error("Expected zerolog instance to be available")
	}

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Errorf("Failed to copy pipe output: %v", err)
	}
}

func TestNew_ProductionMode(t *testing.T) {
	logger := New("production")

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}
	if logger.GetZerolog() == nil {
		t.Error("Expected zerolog instance to be available")
	}
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	logger.Debug("debug message", map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	})

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("Expected log output to contain message")
	}
	if !strings.Contains(output, "value1") {
		t.Error("Expected log output to contain field value")
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	logger.Info("info message", map[string]interface{}{
		"user":   "testuser",
		"action": "login",
	})

	output := buf.String()
	if !strings.Contains(output, "info message") {
		t.Error("Expected log output to contain message")
	}
	if !strings.Contains(output, "testuser") {
		t.Error("Expected log output to contain user field")
	}
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	logger.Warn("warning message", map[string]interface{}{
		"warning_type": "rate_limit",
	})

	output := buf.String()
	if !strings.Contains(output, "warning message") {
		t.Error("Expected log output to contain message")
	}
	if !strings.Contains(output, "rate_limit") {
		t.Error("Expected log output to contain warning_type field")
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	testErr := errors.New("test error")
	logger.Error("error occurred", testErr, map[string]interface{}{
		"context": "database",
	})

	output := buf.String()
	if !strings.Contains(output, "error occurred") {
		t.Error("Expected log output to contain message")
	}
	if !strings.Contains(output, "test error") {
		t.Error("Expected log output to contain error message")
	}
	if !strings.Contains(output, "database") {
		t.Error("Expected log output to contain context field")
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	childLogger := logger.With(map[string]interface{}{
		"component": "api",
		"version":   "1.0",
	})

	childLogger.Info("test message", nil)

	output := buf.String()
	if !strings.Contains(output, "api") {
		t.Error("Expected log output to contain component field from context")
	}
	if !strings.Contains(output, "1.0") {
		t.Error("Expected log output to contain version field from context")
	}
}

func TestWithRequestID(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	requestID := "req-12345"
	childLogger := logger.WithRequestID(requestID)

	childLogger.Info("request received", nil)

	output := buf.String()
	if !strings.Contains(output, requestID) {
		t.Error("Expected log output to contain request ID")
	}
	if !strings.Contains(output, "request_id") {
		t.Error("Expected log output to have request_id field")
	}
}

func TestLogLevels_Production(t *testing.T) {
	var buf bytes.Buffer

	// Create production logger that writes to buffer
	zlog := zerolog.New(&buf).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	// Debug should not appear in production (info level)
	logger.Debug("debug message", nil)
	debugOutput := buf.String()

	buf.Reset()

	// Info should appear
	logger.Info("info message", nil)
	infoOutput := buf.String()

	if strings.Contains(debugOutput, "debug message") {
		t.Error("Debug message should not appear in production logging")
	}
	if !strings.Contains(infoOutput, "info message") {
		t.Error("Info message should appear in production logging")
	}
}

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	logger.Info("test json", map[string]interface{}{
		"key": "value",
	})

	output := buf.String()

	// Try to parse as JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Errorf("Expected valid JSON output, got error: %v", err)
	}

	if logEntry["message"] != "test json" {
		t.Error("Expected JSON to contain message field")
	}
}

func TestNilFields(t *testing.T) {
	var buf bytes.Buffer
	zlog := zerolog.New(&buf).With().Timestamp().Logger()
	logger := &Logger{zlog: zlog}

	// Should not panic with nil fields
	logger.Info("message with nil fields", nil)

	output := buf.String()
	if !strings.Contains(output, "message with nil fields") {
		t.Error("Expected message to be logged even with nil fields")
	}
}
