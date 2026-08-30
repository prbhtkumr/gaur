package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"off", LogLevelOff},
		{"OFF", LogLevelOff},
		{"none", LogLevelOff},
		{"disabled", LogLevelOff},
		{"error", LogLevelError},
		{"ERROR", LogLevelError},
		{"err", LogLevelError},
		{"warn", LogLevelWarn},
		{"WARN", LogLevelWarn},
		{"warning", LogLevelWarn},
		{"info", LogLevelInfo},
		{"INFO", LogLevelInfo},
		{"debug", LogLevelDebug},
		{"DEBUG", LogLevelDebug},
		{"verbose", LogLevelVerbose},
		{"VERBOSE", LogLevelVerbose},
		{"trace", LogLevelVerbose},
		{"all", LogLevelVerbose},
		{"invalid", LogLevelInfo}, // Default
		{"", LogLevelInfo},        // Default
		{"  info  ", LogLevelInfo},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := LogLevelFromString(tc.input)
			if result != tc.expected {
				t.Errorf("LogLevelFromString(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelOff, "off"},
		{LogLevelError, "error"},
		{LogLevelWarn, "warn"},
		{LogLevelInfo, "info"},
		{LogLevelDebug, "debug"},
		{LogLevelVerbose, "verbose"},
		{LogLevel(999), "unknown"}, // Invalid level
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.level.String()
			if result != tc.expected {
				t.Errorf("LogLevel(%d).String() = %q, want %q", tc.level, result, tc.expected)
			}
		})
	}
}

func TestInitLogger(t *testing.T) {
	// Create a temporary directory for test logs
	tmpDir, err := os.MkdirTemp("", "gaur-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test initializing with info level
	err = InitLogger(LogLevelInfo, tmpDir)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}
	defer CloseLogger()

	// Check that the log file was created
	logPath := GetLogFilePath()
	if logPath == "" {
		t.Error("Log file path should not be empty")
	}

	if !strings.HasPrefix(logPath, tmpDir) {
		t.Errorf("Log file should be in temp dir, got %s", logPath)
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file should exist")
	}

	// Test that log level is set correctly
	if GetLogLevel() != LogLevelInfo {
		t.Errorf("Log level should be info, got %v", GetLogLevel())
	}
}

func TestInitLoggerOff(t *testing.T) {
	// Test initializing with off level
	err := InitLogger(LogLevelOff, "")
	if err != nil {
		t.Fatalf("InitLogger with off level failed: %v", err)
	}
	defer CloseLogger()

	// Log file path should be empty when logging is off
	if GetLogFilePath() != "" {
		t.Error("Log file path should be empty when logging is off")
	}

	if GetLogLevel() != LogLevelOff {
		t.Errorf("Log level should be off, got %v", GetLogLevel())
	}
}

func TestSetLogLevel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gaur-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = InitLogger(LogLevelInfo, tmpDir)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}
	defer CloseLogger()

	// Change log level
	SetLogLevel(LogLevelDebug)
	if GetLogLevel() != LogLevelDebug {
		t.Errorf("Log level should be debug, got %v", GetLogLevel())
	}

	SetLogLevel(LogLevelError)
	if GetLogLevel() != LogLevelError {
		t.Errorf("Log level should be error, got %v", GetLogLevel())
	}
}

func TestLogFunctions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gaur-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = InitLogger(LogLevelVerbose, tmpDir)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}
	defer CloseLogger()

	// Test all log functions - they should not panic
	LogError("TEST", "error message: %s", "test")
	LogWarn("TEST", "warning message: %s", "test")
	LogInfo("TEST", "info message: %s", "test")
	LogDebug("TEST", "debug message: %s", "test")
	LogVerbose("TEST", "verbose message: %s", "test")

	// Close logger and read log file
	logPath := GetLogFilePath()
	CloseLogger()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify log entries exist
	if !strings.Contains(logContent, "error message: test") {
		t.Error("Log should contain error message")
	}
	if !strings.Contains(logContent, "warning message: test") {
		t.Error("Log should contain warning message")
	}
	if !strings.Contains(logContent, "info message: test") {
		t.Error("Log should contain info message")
	}
	if !strings.Contains(logContent, "debug message: test") {
		t.Error("Log should contain debug message")
	}
	if !strings.Contains(logContent, "verbose message: test") {
		t.Error("Log should contain verbose message")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gaur-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize with error level only
	err = InitLogger(LogLevelError, tmpDir)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}

	// Log messages at different levels
	LogError("TEST", "error-only-visible")
	LogWarn("TEST", "warn-should-not-appear")
	LogInfo("TEST", "info-should-not-appear")
	LogDebug("TEST", "debug-should-not-appear")

	logPath := GetLogFilePath()
	CloseLogger()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Only error should be visible
	if !strings.Contains(logContent, "error-only-visible") {
		t.Error("Log should contain error message")
	}
	if strings.Contains(logContent, "warn-should-not-appear") {
		t.Error("Log should NOT contain warning message at error level")
	}
	if strings.Contains(logContent, "info-should-not-appear") {
		t.Error("Log should NOT contain info message at error level")
	}
	if strings.Contains(logContent, "debug-should-not-appear") {
		t.Error("Log should NOT contain debug message at error level")
	}
}

func TestSpecializedLoggers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gaur-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = InitLogger(LogLevelVerbose, tmpDir)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}
	defer CloseLogger()

	// Test specialized loggers - they should not panic
	LogModeChange(modeDashboard, modeInstall)
	LogCommand("install", []string{"pkg1", "pkg2"})
	LogCommand("install", []string{})
	LogCommand("install", []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7"})
	LogCommandResult("install", true, nil)
	LogPackageOperation("install", []string{"pkg1"}, true)
	LogPackageOperation("install", []string{"pkg1"}, false)
	LogSettingChange("theme", "old", "new")
	LogCacheOperation("keep3", "100MB")
	LogRefresh("user-request")
}

func TestModeString(t *testing.T) {
	tests := []struct {
		mode     viewMode
		expected string
	}{
		{modeDashboard, "dashboard"},
		{modeInstall, "install"},
		{modeRemove, "remove"},
		{modeUpdate, "update"},
		{modeUpdateSelective, "update-selective"},
		{modeCacheMenu, "cache-menu"},
		{modeCacheSelective, "cache-selective"},
		{modeSettings, "settings"},
		{viewMode(999), "unknown(999)"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := modeString(tc.mode)
			if result != tc.expected {
				t.Errorf("modeString(%d) = %q, want %q", tc.mode, result, tc.expected)
			}
		})
	}
}

func TestLogDirectoryCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gaur-logger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with nested directory that doesn't exist
	nestedDir := filepath.Join(tmpDir, "nested", "logs")

	err = InitLogger(LogLevelInfo, nestedDir)
	if err != nil {
		t.Fatalf("InitLogger should create nested directories: %v", err)
	}
	defer CloseLogger()

	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("Nested log directory should be created")
	}
}

func TestNilLoggerSafety(t *testing.T) {
	// Reset global logger
	if appLogger != nil {
		CloseLogger()
	}

	// These should not panic even without initialization
	LogError("TEST", "message")
	LogWarn("TEST", "message")
	LogInfo("TEST", "message")
	LogDebug("TEST", "message")
	LogVerbose("TEST", "message")
	SetLogLevel(LogLevelDebug)

	if GetLogLevel() != LogLevelOff {
		t.Error("GetLogLevel should return Off when logger is nil")
	}

	if GetLogFilePath() != "" {
		t.Error("GetLogFilePath should return empty string when logger is nil")
	}
}

func TestLogDirectoryValidation(t *testing.T) {
	// Test that relative paths are rejected
	relativePaths := []string{
		"logs",
		"./logs",
		"../logs",
		"../../etc/cron.d",
	}
	
	for _, p := range relativePaths {
		t.Run("RelativePath_"+p, func(t *testing.T) {
			err := InitLogger(LogLevelInfo, p)
			if err == nil {
				t.Errorf("InitLogger should reject relative path: %s", p)
			}
			if appLogger != nil {
				CloseLogger()
			}
		})
	}

	// Test that absolute paths with traversal components are properly cleaned
	tmpDir, err := os.MkdirTemp("", "gaur-logger-test-traversal-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	traversalPath := filepath.Join(tmpDir, "nested", "..", "safe_logs")
	err = InitLogger(LogLevelInfo, traversalPath)
	if err != nil {
		t.Fatalf("InitLogger should accept absolute path with traversal: %v", err)
	}
	defer CloseLogger()

	logPath := GetLogFilePath()
	expectedDir := filepath.Join(tmpDir, "safe_logs")
	
	if !strings.HasPrefix(logPath, expectedDir) {
		t.Errorf("Log path was not properly cleaned. Expected prefix %s, got %s", expectedDir, logPath)
	}
}
