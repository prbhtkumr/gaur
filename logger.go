package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the verbosity level of logging
type LogLevel int

const (
	// LogLevelOff disables all logging
	LogLevelOff LogLevel = iota
	// LogLevelError logs only errors
	LogLevelError
	// LogLevelWarn logs warnings and errors
	LogLevelWarn
	// LogLevelInfo logs info, warnings, and errors
	LogLevelInfo
	// LogLevelDebug logs debug info and above
	LogLevelDebug
	// LogLevelVerbose logs everything including detailed traces
	LogLevelVerbose
)

// LogLevelNames maps log levels to their string representation
var LogLevelNames = map[LogLevel]string{
	LogLevelOff:     "off",
	LogLevelError:   "error",
	LogLevelWarn:    "warn",
	LogLevelInfo:    "info",
	LogLevelDebug:   "debug",
	LogLevelVerbose: "verbose",
}

// LogLevelFromString converts a string to LogLevel
func LogLevelFromString(s string) LogLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off", "none", "disabled":
		return LogLevelOff
	case "error", "err":
		return LogLevelError
	case "warn", "warning":
		return LogLevelWarn
	case "info":
		return LogLevelInfo
	case "debug":
		return LogLevelDebug
	case "verbose", "trace", "all":
		return LogLevelVerbose
	default:
		return LogLevelInfo // Default to info
	}
}

// String returns the string representation of a LogLevel
func (l LogLevel) String() string {
	if name, ok := LogLevelNames[l]; ok {
		return name
	}
	return "unknown"
}

// Logger is the application-wide logger with support for levels and file output
type Logger struct {
	mu       sync.Mutex
	level    LogLevel
	file     *os.File
	filePath string
	logger   *log.Logger
	enabled  bool
}

// Global logger instance
var appLogger *Logger

// InitLogger initializes the global logger with the given configuration
func InitLogger(level LogLevel, logDir string) error {
	if appLogger != nil {
		appLogger.Close()
	}

	appLogger = &Logger{
		level:   level,
		enabled: level != LogLevelOff,
	}

	if !appLogger.enabled {
		appLogger.logger = log.New(io.Discard, "", 0)
		return nil
	}

	// Determine log directory (XDG compliant)
	if logDir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = filepath.Join(os.Getenv("HOME"), ".config")
		}
		logDir = filepath.Join(configDir, "gaur")
	}

	// Sanitize and validate logDir to prevent path traversal
	logDir = filepath.Clean(logDir)
	if !filepath.IsAbs(logDir) {
		return fmt.Errorf("log directory must be an absolute path: %s", logDir)
	}

	// Ensure directory exists with restrictive permissions
	if err := os.MkdirAll(logDir, 0700); err != nil { // #nosec G301,G703 - 0700 is intentionally restrictive, logDir is validated above
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with date suffix for rotation
	logFileName := fmt.Sprintf("gaur-%s.log", time.Now().Format("2006-01-02"))
	appLogger.filePath = filepath.Join(logDir, logFileName)

	file, err := os.OpenFile(appLogger.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	appLogger.file = file
	appLogger.logger = log.New(file, "", 0) // Custom formatting in write methods

	// Log startup
	appLogger.logStartup()

	return nil
}

// logStartup logs the application startup information
func (l *Logger) logStartup() {
	l.writeLog(LogLevelInfo, "APP", "═══════════════════════════════════════════════════════════════")
	l.writeLog(LogLevelInfo, "APP", "gaur started")
	l.writeLog(LogLevelInfo, "APP", "Log level: %s", l.level.String())
	l.writeLog(LogLevelInfo, "APP", "Log file: %s", l.filePath)
	l.writeLog(LogLevelInfo, "APP", "Go version: %s", runtime.Version())
	l.writeLog(LogLevelInfo, "APP", "OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	l.writeLog(LogLevelInfo, "APP", "═══════════════════════════════════════════════════════════════")
}

// Close closes the log file
func (l *Logger) Close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.writeLogLocked(LogLevelInfo, "APP", "gaur shutting down")
		if err := l.file.Close(); err != nil {
			// Can't log this error since we're closing, but we've handled it
			_ = err
		}
		l.file = nil
	}
}

// SetLevel changes the log level at runtime
func (l *Logger) SetLevel(level LogLevel) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	oldLevel := l.level
	l.level = level
	l.enabled = level != LogLevelOff

	if l.enabled && l.file != nil {
		l.writeLogLocked(LogLevelInfo, "CONFIG", "Log level changed: %s -> %s", oldLevel.String(), level.String())
	}
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	if l == nil {
		return LogLevelOff
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// writeLog writes a log entry with the given level and category
func (l *Logger) writeLog(level LogLevel, category string, format string, args ...interface{}) {
	if l == nil || !l.enabled || level > l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writeLogLocked(level, category, format, args...)
}

func center(s string, width int) string {
	if len(s) >= width {
		return s
	}
	padding := width - len(s)
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// writeLogLocked writes a log entry (must hold lock)
func (l *Logger) writeLogLocked(level LogLevel, category string, format string, args ...interface{}) {
	if l.logger == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := strings.ToUpper(level.String())
	message := fmt.Sprintf(format, args...)

	// Format: [TIMESTAMP] [LEVEL] [CATEGORY] message
	logLine := fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, center(levelStr, 7), center(category, 9), message)
	l.logger.Println(logLine)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Public logging functions - use these throughout the application
// ═══════════════════════════════════════════════════════════════════════════════

// LogError logs an error message
func LogError(category string, format string, args ...interface{}) {
	if appLogger != nil {
		appLogger.writeLog(LogLevelError, category, format, args...)
	}
}

// LogWarn logs a warning message
func LogWarn(category string, format string, args ...interface{}) {
	if appLogger != nil {
		appLogger.writeLog(LogLevelWarn, category, format, args...)
	}
}

// LogInfo logs an informational message
func LogInfo(category string, format string, args ...interface{}) {
	if appLogger != nil {
		appLogger.writeLog(LogLevelInfo, category, format, args...)
	}
}

// LogDebug logs a debug message
func LogDebug(category string, format string, args ...interface{}) {
	if appLogger != nil {
		appLogger.writeLog(LogLevelDebug, category, format, args...)
	}
}

// LogVerbose logs a verbose/trace message
func LogVerbose(category string, format string, args ...interface{}) {
	if appLogger != nil {
		appLogger.writeLog(LogLevelVerbose, category, format, args...)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Specialized logging helpers
// ═══════════════════════════════════════════════════════════════════════════════

// LogModeChange logs a mode transition
func LogModeChange(from, to viewMode) {
	LogInfo("MODE", "Mode changed: %s -> %s", modeString(from), modeString(to))
}

// LogCommand logs a command execution
func LogCommand(action string, packages []string) {
	if len(packages) == 0 {
		LogInfo("CMD", "%s: (no packages)", action)
		return
	}
	if len(packages) <= 5 {
		LogInfo("CMD", "%s: %s", action, strings.Join(packages, ", "))
	} else {
		LogInfo("CMD", "%s: %s... (+%d more)", action, strings.Join(packages[:5], ", "), len(packages)-5)
	}
	LogDebug("CMD", "%s full list: %v", action, packages)
}

// LogCommandResult logs the result of a command execution
func LogCommandResult(action string, success bool, err error) {
	if success {
		LogInfo("CMD", "%s completed successfully", action)
	} else if err != nil {
		LogError("CMD", "%s failed: %v", action, err)
	} else {
		LogWarn("CMD", "%s completed with warnings", action)
	}
}

// LogSearch logs a search operation
func LogSearch(source, query string, resultCount int, duration time.Duration) {
	LogDebug("SEARCH", "%s search for '%s': %d results in %v", source, query, resultCount, duration)
}

// LogPackageOperation logs a package operation (install/remove/update)
func LogPackageOperation(operation string, packages []string, confirmed bool) {
	if confirmed {
		LogInfo("PKG", "%s confirmed for %d package(s)", operation, len(packages))
		LogDebug("PKG", "%s packages: %v", operation, packages)
	} else {
		LogDebug("PKG", "%s cancelled for %d package(s)", operation, len(packages))
	}
}

// LogSettingChange logs a configuration setting change
func LogSettingChange(setting, oldValue, newValue string) {
	LogInfo("CONFIG", "Setting changed: %s = %s (was: %s)", setting, newValue, oldValue)
}

// LogDataLoad logs data loading operations
func LogDataLoad(dataType string, count int, duration time.Duration, err error) {
	if err != nil {
		LogError("DATA", "Failed to load %s: %v", dataType, err)
	} else {
		LogDebug("DATA", "Loaded %s: %d items in %v", dataType, count, duration)
	}
}

// LogKeyPress logs key press events (verbose level only)
func LogKeyPress(key string, mode viewMode, focused bool) {
	focusStr := ""
	if focused {
		focusStr = " (input focused)"
	}
	LogVerbose("INPUT", "Key: %s in %s%s", key, modeString(mode), focusStr)
}

// LogMouseEvent logs mouse events (verbose level only)
func LogMouseEvent(action string, x, y int) {
	LogVerbose("INPUT", "Mouse: %s at (%d, %d)", action, x, y)
}

// LogCacheOperation logs cache cleaning operations
func LogCacheOperation(strategy string, estimatedSize string) {
	LogInfo("CACHE", "Cache clean: %s (estimated: %s)", strategy, estimatedSize)
}

// LogRefresh logs refresh operations
func LogRefresh(trigger string) {
	LogInfo("REFRESH", "Full refresh triggered by: %s", trigger)
}

// modeString converts a viewMode to its string representation
func modeString(m viewMode) string {
	switch m {
	case modeDashboard:
		return "dashboard"
	case modeInstall:
		return "install"
	case modeRemove:
		return "remove"
	case modeUpdate:
		return "update"
	case modeUpdateSelective:
		return "update-selective"
	case modeCacheMenu:
		return "cache-menu"
	case modeCacheSelective:
		return "cache-selective"
	case modeSettings:
		return "settings"
	default:
		return fmt.Sprintf("unknown(%d)", m)
	}
}

// GetLogFilePath returns the current log file path
func GetLogFilePath() string {
	if appLogger != nil && appLogger.filePath != "" {
		return appLogger.filePath
	}
	return ""
}

// SetLogLevel sets the global log level
func SetLogLevel(level LogLevel) {
	if appLogger != nil {
		appLogger.SetLevel(level)
	}
}

// GetLogLevel returns the current global log level
func GetLogLevel() LogLevel {
	if appLogger != nil {
		return appLogger.GetLevel()
	}
	return LogLevelOff
}

// CloseLogger closes the global logger
func CloseLogger() {
	if appLogger != nil {
		appLogger.Close()
		appLogger = nil
	}
}
