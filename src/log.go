package main

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

// RotatingWriter wraps a file writer with rotation capabilities
type RotatingWriter struct {
	filePath string
	file     *os.File
	mutex    sync.Mutex
	maxSize  int64
}

// NewRotatingWriter creates a new rotating writer
func NewRotatingWriter(filePath string, maxSize int64) (*RotatingWriter, error) {
	rw := &RotatingWriter{
		filePath: filePath,
		maxSize:  maxSize,
	}

	// Check if rotation is needed before opening
	if err := rw.rotateIfNeeded(); err != nil {
		return nil, fmt.Errorf("failed to initialize rotating writer: %w", err)
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	rw.file = file
	return rw, nil
}

// Write implements io.Writer interface
func (rw *RotatingWriter) Write(p []byte) (n int, err error) {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()

	// Check if rotation is needed before writing
	if err := rw.rotateIfNeeded(); err != nil {
		// If rotation fails, still try to write to current file
		// to avoid losing log data
		if rw.file != nil {
			return rw.file.Write(p)
		}
		return 0, err
	}

	return rw.file.Write(p)
}

// rotateIfNeeded checks if rotation is needed and performs it
func (rw *RotatingWriter) rotateIfNeeded() error {
	if rw.file == nil {
		return nil
	}

	// Get current file size
	info, err := rw.file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	if info.Size() < rw.maxSize {
		return nil
	}

	// Close current file before rotation
	if err := rw.file.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Remove old log if it exists
	oldLogPath := rw.filePath + ".old"
	if _, err := os.Stat(oldLogPath); err == nil {
		if err := os.Remove(oldLogPath); err != nil {
			return fmt.Errorf("failed to remove old log file: %w", err)
		}
	}

	// Rename current log to old
	if err := os.Rename(rw.filePath, oldLogPath); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Create new log file
	file, err := os.OpenFile(rw.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file after rotation: %w", err)
	}

	rw.file = file
	return nil
}

// Close closes the rotating writer
func (rw *RotatingWriter) Close() error {
	rw.mutex.Lock()
	defer rw.mutex.Unlock()

	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// setupLogging configures logrus to write logs to both console and file with rotation
func setupLogging() {
	const maxLogSize = 10 * 1024 * 1024 // 10MB

	logFilePath := realPath(logPath)
	rotatingWriter, err := NewRotatingWriter(logFilePath, maxLogSize)
	if err != nil {
		logrus.Fatal("Failed to create rotating log writer: ", err)
	}

	// Set logrus to log to both stdout and the rotating log file
	multiWriter := io.MultiWriter(os.Stdout, rotatingWriter)
	logrus.SetOutput(multiWriter)
	logrus.SetLevel(logLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   false, // Disable colors for file output
	})

	// Set custom report caller to show file and line info
	logrus.SetReportCaller(true)
}

// logError is a convenience function for error logging
func logError(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

// logWarn is a convenience function for warning logging
func logWarn(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

// logInfo is a convenience function for info logging
func logInfo(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

// logDebug is a convenience function for debug logging
func logDebug(format string, args ...interface{}) {
	if SysConfig.DebugLogEnabled {
		logrus.Debugf(format, args...)
	}
}

// logFatal is a convenience function for fatal logging
func logFatal(format string, args ...interface{}) {
	logrus.Fatalf(format, args...)
}

// logPanic is a convenience function for panic logging
func logPanic(format string, args ...interface{}) {
	logrus.Panicf(format, args...)
}

// WithFields creates a log entry with structured fields
func WithFields(fields map[string]interface{}) *logrus.Entry {
	return logrus.WithFields(logrus.Fields(fields))
}
