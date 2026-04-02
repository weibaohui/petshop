package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Module    string                 `json:"module"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Logger provides structured logging
type Logger struct {
	mu     sync.Mutex
	module string
}

var (
	globalLogger *Logger
	once         sync.Once
	logFile      *os.File
)

// Init initializes the global logger
func Init(logDir string) error {
	var err error
	once.Do(func() {
		if logDir != "" {
			os.MkdirAll(logDir, 0755)
			logFile, err = os.OpenFile(
				filepath.Join(logDir, fmt.Sprintf("app_%s.log", time.Now().Format("20060102"))),
				os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
			}
		}
		globalLogger = &Logger{module: "app"}
	})
	return err
}

// New creates a new logger for a specific module
func New(module string) *Logger {
	return &Logger{module: module}
}

// Close closes the log file
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// ResetForTesting resets the logger state for testing
// This should only be used in tests
func ResetForTesting() {
	once = sync.Once{}
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	globalLogger = nil
}

func (l *Logger) log(level LogLevel, msg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format("2006-01-02 15:04:05.000"),
		Level:     level.String(),
		Module:    l.module,
		Message:   msg,
		Fields:    fields,
	}

	output := fmt.Sprintf("[%s] [%s] [%s] %s %v\n",
		entry.Timestamp, entry.Level, entry.Module, entry.Message, entry.Fields)

	fmt.Print(output)

	if logFile != nil {
		l.mu.Lock()
		logFile.WriteString(output)
		l.mu.Unlock()
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	l.log(DEBUG, msg, fields)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log(INFO, msg, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.log(WARN, msg, fields)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.log(ERROR, msg, fields)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields map[string]interface{}) {
	l.log(FATAL, msg, fields)
	os.Exit(1)
}

// Package-level convenience functions
func Debug(msg string, fields map[string]interface{}) {
	globalLogger.Debug(msg, fields)
}

func Info(msg string, fields map[string]interface{}) {
	globalLogger.Info(msg, fields)
}

func Warn(msg string, fields map[string]interface{}) {
	globalLogger.Warn(msg, fields)
}

func Error(msg string, fields map[string]interface{}) {
	globalLogger.Error(msg, fields)
}

func Access(msg string, fields map[string]interface{}) {
	globalLogger.Info(msg, fields)
}
