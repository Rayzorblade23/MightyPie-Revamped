package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// Log levels ordered by severity
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var levelNames = map[LogLevel]string{
	LevelDebug: "DBG",
	LevelInfo:  "INF",
	LevelWarn:  "WRN",
	LevelError: "ERR",
	LevelFatal: "FTL",
}

// Logger represents a structured logger
type Logger struct {
	component  string
	level      LogLevel
	output     io.Writer
	timeFormat string
	mu         sync.Mutex
}

var (
	// Global default logger
	defaultLogger *Logger
	// Global log level
	globalLevel LogLevel = LevelInfo
)

// init initializes the default logger
func init() {
	// Parse RUST_LOG environment variable
	parseRustLogEnv()
	
	// Initialize default logger
	defaultLogger = &Logger{
		component:  "main",
		level:      globalLevel,
		output:     os.Stdout,
		timeFormat: "2006/01/02 15:04:05",
	}
}

// parseRustLogEnv parses the RUST_LOG environment variable to set the global log level
func parseRustLogEnv() {
	rustLog := strings.ToUpper(os.Getenv("RUST_LOG"))
	
	switch rustLog {
	case "TRACE", "DEBUG":
		globalLevel = LevelDebug
	case "INFO":
		globalLevel = LevelInfo
	case "WARN", "WARNING":
		globalLevel = LevelWarn
	case "ERROR":
		globalLevel = LevelError
	case "FATAL":
		globalLevel = LevelFatal
	default:
		// Default to INFO if not specified or invalid
		globalLevel = LevelInfo
	}
}

// New creates a new logger with the specified component name
func New(component string) *Logger {
	return &Logger{
		component:  component,
		level:      globalLevel,
		output:     os.Stdout,
		timeFormat: "2006/01/02 15:04:05",
	}
}

// SetOutput sets the output writer for the logger
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}

// SetLevel sets the minimum log level for this logger
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log logs a message at the specified level
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Format the log message
	timestamp := time.Now().Format(l.timeFormat)
	levelName := levelNames[level]
	
	var msg string
	if len(args) > 0 {
		// Use the format string with args
		msg = fmt.Sprintf(format, args...)
	} else {
		// No args, treat format as a literal message
		msg = format
	}

	// Build the log entry without any color codes
	logEntry := fmt.Sprintf("%s [%s] [%s] %s\n", 
		timestamp, levelName, l.component, msg)

	// Write to output
	_, _ = fmt.Fprint(l.output, logEntry)
	
	// If fatal, exit the program
	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug is the lowest log level available

// Debug logs a message at debug level
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs a message at info level
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a message at warn level
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs a message at error level
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal logs a message at fatal level and exits the program
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
}

// Global convenience functions

// Debug is the lowest log level available globally

// Debug logs a message at debug level using the default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs a message at info level using the default logger
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs a message at warn level using the default logger
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs a message at error level using the default logger
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs a message at fatal level and exits the program
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// ReplaceStdLog replaces the standard library's log package with our structured logger
func ReplaceStdLog(component string) {
	logger := New(component)
	log.SetOutput(&stdLogAdapter{logger: logger})
	log.SetFlags(0) // Remove timestamp as we'll add our own
}

// stdLogAdapter adapts our logger to be used as an io.Writer for the standard log package
type stdLogAdapter struct {
	logger *Logger
}

// Write implements io.Writer for the standard log package
func (a *stdLogAdapter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSuffix(string(p), "\n")
	a.logger.Info("%s", msg)
	return len(p), nil
}
