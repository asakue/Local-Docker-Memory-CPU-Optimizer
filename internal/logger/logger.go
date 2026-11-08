package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logMutex sync.Mutex
	logFile  *os.File
	logPath  string
)

// Init initializes the logger file (defaulting to dockertrim.log in current dir).
func Init(customPath string) error {
	logMutex.Lock()
	defer logMutex.Unlock()

	if customPath == "" {
		customPath = "dockertrim.log"
	}
	logPath = customPath

	// Check file size, truncate if > 5MB to avoid disk bloat
	if fi, err := os.Stat(logPath); err == nil && fi.Size() > 5*1024*1024 {
		_ = os.Rename(logPath, logPath+".old")
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	logFile = f

	write("INFO", fmt.Sprintf("=== dockertrim session started at %s ===", time.Now().Format(time.RFC3339)))
	return nil
}

func write(level, msg string) {
	if logFile == nil {
		// Try lazy init
		if err := Init(""); err != nil {
			return
		}
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, msg)
	_, _ = logFile.WriteString(line)
	_ = logFile.Sync()
}

// Info logs an informational message.
func Info(format string, args ...interface{}) {
	logMutex.Lock()
	defer logMutex.Unlock()
	write("INFO", fmt.Sprintf(format, args...))
}

// Warn logs a warning message.
func Warn(format string, args ...interface{}) {
	logMutex.Lock()
	defer logMutex.Unlock()
	write("WARN", fmt.Sprintf(format, args...))
}

// Error logs an error message.
func Error(format string, args ...interface{}) {
	logMutex.Lock()
	defer logMutex.Unlock()
	write("ERROR", fmt.Sprintf(format, args...))
}

// GetLogPath returns the absolute path of the current log file.
func GetLogPath() string {
	abs, err := filepath.Abs(logPath)
	if err != nil {
		return logPath
	}
	return abs
}

// Close closes the logger file.
func Close() {
	logMutex.Lock()
	defer logMutex.Unlock()
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
}

