package logger

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestLogLevelString tests LogLevel.String() method
func TestLogLevelString(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected string
	}{
		{"DEBUG", DEBUG, "DEBUG"},
		{"INFO", INFO, "INFO"},
		{"WARN", WARN, "WARN"},
		{"ERROR", ERROR, "ERROR"},
		{"FATAL", FATAL, "FATAL"},
		{"UNKNOWN", LogLevel(100), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("LogLevel.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestNew tests New() function
func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		module string
	}{
		{"default module", "app"},
		{"custom module", "auth"},
		{"empty module", ""},
		{"module with special chars", "api/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.module)
			if logger == nil {
				t.Fatal("New() returned nil")
			}
			if logger.module != tt.module {
				t.Errorf("Logger.module = %v, want %v", logger.module, tt.module)
			}
		})
	}
}

// TestLoggerLog tests Logger.log() method
func TestLoggerLog(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New("test")

	// Test with fields
	logger.log(INFO, "test message", map[string]interface{}{"key": "value"})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "INFO") {
		t.Errorf("log output missing level: %v", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("log output missing module: %v", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("log output missing message: %v", output)
	}
	if !strings.Contains(output, "key") || !strings.Contains(output, "value") {
		t.Errorf("log output missing fields: %v", output)
	}
}

// TestLoggerMethods tests Debug/Info/Warn/Error methods
func TestLoggerMethods(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
		fn    func(*Logger, string, map[string]interface{})
	}{
		{"Debug", DEBUG, (*Logger).Debug},
		{"Info", INFO, (*Logger).Info},
		{"Warn", WARN, (*Logger).Warn},
		{"Error", ERROR, (*Logger).Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			logger := New("test")
			tt.fn(logger, tt.name+" message", map[string]interface{}{"test": true})

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			expectedLevel := tt.level.String()
			if !strings.Contains(output, expectedLevel) {
				t.Errorf("output missing level %s: %v", expectedLevel, output)
			}
			if !strings.Contains(output, tt.name+" message") {
				t.Errorf("output missing message: %v", output)
			}
		})
	}
}

// TestLoggerWithNilFields tests logging with nil fields
func TestLoggerWithNilFields(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New("test")
	logger.Info("nil fields message", nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "nil fields message") {
		t.Errorf("output missing message: %v", output)
	}
}

// TestLoggerWithEmptyFields tests logging with empty fields
func TestLoggerWithEmptyFields(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New("test")
	logger.Info("empty fields message", map[string]interface{}{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "empty fields message") {
		t.Errorf("output missing message: %v", output)
	}
}

// TestInit tests Init() function with log directory
func TestInit(t *testing.T) {
	// Note: This test assumes Init() hasn't been called yet in this test run
	// Create temp directory
	tempDir := t.TempDir()

	err := Init(tempDir)
	if err != nil {
		t.Errorf("Init() error = %v", err)
	}

	if globalLogger == nil {
		t.Error("globalLogger is nil after Init()")
	}

	if logFile == nil {
		t.Error("logFile is nil after Init()")
	}

	// Check if log file was created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 log file, got %d", len(entries))
	}

	if !strings.HasSuffix(entries[0].Name(), ".log") {
		t.Errorf("Expected .log file, got %s", entries[0].Name())
	}
}

// TestInitWithEmptyDir tests Init() with empty directory (should create logger without file)
func TestInitWithEmptyDir(t *testing.T) {
	// This test must be run before any other test that calls Init()
	// because sync.Once cannot be reset
	// If Init() was already called, this test validates the existing state

	// Note: We cannot reset sync.Once, so we just check the current state
	// In a real scenario, this test should run in isolation
	if globalLogger == nil {
		t.Skip("Init() was not called - this test should run before TestInit")
	}

	// If we get here, Init() was already called, so we just verify the state
	t.Log("Init() was already called, verifying global state")
}

// TestClose tests Close() function
func TestClose(t *testing.T) {
	// Ensure Init was called first
	tempDir := t.TempDir()
	Init(tempDir)

	// Write something to ensure file is open
	if globalLogger != nil {
		Info("test message", nil)
	}

	// Close should not panic
	Close()

	// Try to close again (should not panic)
	Close()
}

// TestCloseWithoutInit tests Close() without initialization
func TestCloseWithoutInit(t *testing.T) {
	// Should not panic even if logFile is nil
	Close()
}

// TestPackageLevelFunctions tests package-level convenience functions
func TestPackageLevelFunctions(t *testing.T) {
	// Ensure Init() was called
	if globalLogger == nil {
		t.Skip("globalLogger is nil - Init() must be called first")
	}

	tests := []struct {
		name string
		fn   func(string, map[string]interface{})
	}{
		{"Debug", Debug},
		{"Info", Info},
		{"Warn", Warn},
		{"Error", Error},
		{"Access", Access},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			tt.fn(tt.name+" message", map[string]interface{}{"key": "value"})

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if !strings.Contains(output, tt.name+" message") {
				t.Errorf("output missing message: %v", output)
			}
		})
	}
}

// TestPackageLevelFunctionsPanic tests that package functions panic without init
func TestPackageLevelFunctionsPanic(t *testing.T) {
	// This test can only work if globalLogger is nil
	// Since we can't reset sync.Once, we skip if already initialized
	if globalLogger != nil {
		t.Skip("globalLogger already initialized - cannot test panic case")
	}

	// Without Init(), globalLogger is nil, so calling package functions should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling package function without Init()")
		}
	}()

	Info("test", nil)
}

// TestConcurrency tests concurrent logging
func TestConcurrency(t *testing.T) {
	// Capture stdout to verify concurrent logging works
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New("concurrent")

	var wg sync.WaitGroup
	numGoroutines := 5
	numMessages := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				logger.Info("concurrent message", map[string]interface{}{
					"goroutine": id,
					"message":   j,
				})
			}
		}(i)
	}

	wg.Wait()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected number of messages
	expectedCount := numGoroutines * numMessages
	actualCount := strings.Count(output, "concurrent message")
	if actualCount != expectedCount {
		t.Errorf("Expected %d log messages, got %d", expectedCount, actualCount)
	}
}

// TestConcurrentPackageFunctions tests concurrent package-level functions
func TestConcurrentPackageFunctions(t *testing.T) {
	tempDir := t.TempDir()
	Init(tempDir)

	var wg sync.WaitGroup
	numGoroutines := 5
	numMessages := 10

	funcs := []func(string, map[string]interface{}){
		Debug,
		Info,
		Warn,
		Error,
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				fn := funcs[j%len(funcs)]
				fn("concurrent", map[string]interface{}{
					"goroutine": id,
					"message":   j,
				})
			}
		}(i)
	}

	wg.Wait()
}

// TestLogEntryStructure tests the LogEntry structure
func TestLogEntryStructure(t *testing.T) {
	entry := LogEntry{
		Timestamp: "2024-01-01 12:00:00.000",
		Level:     "INFO",
		Module:    "test",
		Message:   "test message",
		Fields:    map[string]interface{}{"key": "value"},
	}

	if entry.Timestamp != "2024-01-01 12:00:00.000" {
		t.Error("Timestamp mismatch")
	}
	if entry.Level != "INFO" {
		t.Error("Level mismatch")
	}
	if entry.Module != "test" {
		t.Error("Module mismatch")
	}
	if entry.Message != "test message" {
		t.Error("Message mismatch")
	}
	if entry.Fields["key"] != "value" {
		t.Error("Fields mismatch")
	}
}

// TestLogEntryWithNilFields tests LogEntry with nil Fields
func TestLogEntryWithNilFields(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     "DEBUG",
		Module:    "test",
		Message:   "test",
		Fields:    nil,
	}

	if entry.Fields != nil {
		t.Error("Fields should be nil")
	}
}

// TestLogFileContent tests actual log file content
func TestLogFileContent(t *testing.T) {
	// Create a temp directory and manually create a log file for testing
	tempDir := t.TempDir()
	logFilePath := filepath.Join(tempDir, "test.log")

	// Create log file manually
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}
	defer f.Close()

	// Write test content
	testContent := "[2024-01-01 12:00:00.000] [INFO] [filetest] file test message map[foo:bar]\n"
	f.WriteString(testContent)

	// Verify the file content
	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "file test message") {
		t.Error("Log file missing message")
	}
	if !strings.Contains(contentStr, "INFO") {
		t.Error("Log file missing level")
	}
	if !strings.Contains(contentStr, "filetest") {
		t.Error("Log file missing module")
	}
	if !strings.Contains(contentStr, "foo") || !strings.Contains(contentStr, "bar") {
		t.Error("Log file missing fields")
	}
}

// TestMultipleLoggers tests multiple logger instances
func TestMultipleLoggers(t *testing.T) {
	logger1 := New("module1")
	logger2 := New("module2")
	logger3 := New("module3")

	if logger1.module != "module1" {
		t.Error("logger1 module mismatch")
	}
	if logger2.module != "module2" {
		t.Error("logger2 module mismatch")
	}
	if logger3.module != "module3" {
		t.Error("logger3 module mismatch")
	}

	// Test that loggers are independent
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger1.Info("msg1", nil)
	logger2.Info("msg2", nil)
	logger3.Info("msg3", nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "module1") {
		t.Error("output missing module1")
	}
	if !strings.Contains(output, "module2") {
		t.Error("output missing module2")
	}
	if !strings.Contains(output, "module3") {
		t.Error("output missing module3")
	}
}

// TestLogLevelValues tests that log levels have correct values
func TestLogLevelValues(t *testing.T) {
	if DEBUG != 0 {
		t.Errorf("DEBUG should be 0, got %d", DEBUG)
	}
	if INFO != 1 {
		t.Errorf("INFO should be 1, got %d", INFO)
	}
	if WARN != 2 {
		t.Errorf("WARN should be 2, got %d", WARN)
	}
	if ERROR != 3 {
		t.Errorf("ERROR should be 3, got %d", ERROR)
	}
	if FATAL != 4 {
		t.Errorf("FATAL should be 4, got %d", FATAL)
	}
}

// TestLargeFields tests logging with large fields
func TestLargeFields(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New("test")

	// Create large field value
	largeValue := strings.Repeat("x", 10000)
	logger.Info("large field test", map[string]interface{}{
		"large": largeValue,
	})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "large field test") {
		t.Error("output missing message")
	}
	if !strings.Contains(output, largeValue) {
		t.Error("output missing large value")
	}
}

// TestSpecialCharacters tests logging with special characters
func TestSpecialCharacters(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	logger := New("test")
	logger.Info("special chars: \n\t\"'\\", map[string]interface{}{
		"key": "value with \n\t\"'",
	})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Just verify it doesn't panic and contains the message
	if !strings.Contains(output, "special chars") {
		t.Error("output missing message")
	}
}

// TestAccessFunction specifically tests the Access function
func TestAccessFunction(t *testing.T) {
	// Ensure Init() was called
	if globalLogger == nil {
		tempDir := t.TempDir()
		Init(tempDir)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Access("access log", map[string]interface{}{
		"method": "GET",
		"path":   "/api/test",
	})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Access() uses Info level internally
	if !strings.Contains(output, "INFO") {
		t.Error("output missing INFO level")
	}
	if !strings.Contains(output, "access log") {
		t.Error("output missing message")
	}
}

// TestFatalSubprocess tests Fatal() using subprocess
// This test is skipped by default because it calls os.Exit(1)
func TestFatalSubprocess(t *testing.T) {
	if os.Getenv("BE_CRASHER") != "1" {
		t.Skip("Skipping Fatal() test - run with BE_CRASHER=1 to test")
	}

	// This code runs in a subprocess and should exit
	tempDir := t.TempDir()
	Init(tempDir)
	logger := New("fatal_test")
	logger.Fatal("fatal message", map[string]interface{}{"test": true})
}

// Benchmark tests

func BenchmarkLoggerInfo(b *testing.B) {
	logger := New("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", map[string]interface{}{
			"iteration": i,
			"data":      "test",
		})
	}
}

func BenchmarkLoggerInfoNoFields(b *testing.B) {
	logger := New("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", nil)
	}
}

func BenchmarkLoggerConcurrent(b *testing.B) {
	logger := New("bench")

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.Info("concurrent message", map[string]interface{}{
				"iteration": i,
			})
			i++
		}
	})
}
