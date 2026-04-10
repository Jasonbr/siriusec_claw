package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level constants
const (
	LevelFatal = 0
	LevelError = 1
	LevelWarn  = 2
	LevelInfo  = 3
	LevelDebug = 4
	LevelTrace = 5
)

var (
	globalLevel    = LevelInfo
	globalLogDir   string
	globalMu       sync.Mutex
	globalFile     *os.File
	globalFileDate string
)

// Init initializes global logging with a directory and level.
func Init(logDir string, level int) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogDir = logDir
	globalLevel = level
	if logDir != "" {
		_ = os.MkdirAll(logDir, 0755)
	}
}

// Sub returns a sub-logger with a name prefix.
func Sub(name string) *Logger {
	return &Logger{prefix: name}
}

// Logger is a named sub-logger.
type Logger struct {
	prefix string
}

// Info logs at info level.
func (l *Logger) Info(format string, args ...interface{}) {
	if globalLevel >= LevelInfo {
		writeLog(l.prefix, "INFO", format, args...)
	}
}

// Warn logs at warn level.
func (l *Logger) Warn(format string, args ...interface{}) {
	if globalLevel >= LevelWarn {
		writeLog(l.prefix, "WARN", format, args...)
	}
}

// Error logs at error level.
func (l *Logger) Error(format string, args ...interface{}) {
	if globalLevel >= LevelError {
		writeLog(l.prefix, "ERROR", format, args...)
	}
}

// Debug logs at debug level.
func (l *Logger) Debug(format string, args ...interface{}) {
	if globalLevel >= LevelDebug {
		writeLog(l.prefix, "DEBUG", format, args...)
	}
}

// Global convenience functions

// Info logs at info level.
func Info(format string, args ...interface{}) {
	if globalLevel >= LevelInfo {
		writeLog("", "INFO", format, args...)
	}
}

// Warn logs at warn level.
func Warn(format string, args ...interface{}) {
	if globalLevel >= LevelWarn {
		writeLog("", "WARN", format, args...)
	}
}

// Error(format string, args ...interface{}) logs at error level.
func Error(format string, args ...interface{}) {
	if globalLevel >= LevelError {
		writeLog("", "ERROR", format, args...)
	}
}

// Debug logs at debug level.
func Debug(format string, args ...interface{}) {
	if globalLevel >= LevelDebug {
		writeLog("", "DEBUG", format, args...)
	}
}

func writeLog(prefix, level, format string, args ...interface{}) {
	now := time.Now()
	msg := fmt.Sprintf(format, args...)
	var tag string
	if prefix != "" {
		tag = fmt.Sprintf("[%s] ", prefix)
	}
	line := fmt.Sprintf("%s %s %s%s\n", now.Format("15:04:05.000"), level, tag, msg)

	// Write to stderr
	_, _ = io.WriteString(os.Stderr, line)

	// Write to log file
	if globalLogDir != "" {
		writeToFile(now, line)
	}
}

func writeToFile(now time.Time, line string) {
	globalMu.Lock()
	defer globalMu.Unlock()

	dateStr := now.Format("2006-01-02")
	if globalFile == nil || globalFileDate != dateStr {
		if globalFile != nil {
			_ = globalFile.Close()
		}
		path := filepath.Join(globalLogDir, fmt.Sprintf("siriusec_claw-%s.log", dateStr))
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		globalFile = f
		globalFileDate = dateStr
	}
	_, _ = globalFile.WriteString(line)
}

func init() {
	// Use slog default handler for stdlib compatibility
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
}
