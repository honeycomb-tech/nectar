package logger

import (
	"fmt"
	"log"
	"nectar/errors"
)

// Logger provides standardized logging methods
type Logger struct {
	component string
}

// New creates a new logger for a component
func New(component string) *Logger {
	return &Logger{component: component}
}

// Info logs an informational message
func (l *Logger) Info(operation, message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	log.Printf("[INFO] %s.%s: %s", l.component, operation, msg)
}

// Warning logs a warning
func (l *Logger) Warning(operation, message string, args ...interface{}) {
	msg := fmt.Sprintf(message, args...)
	log.Printf("[WARN] %s.%s: %s", l.component, operation, msg)
	errors.Get().Warning(l.component, operation, msg)
}

// Error logs an error to both log and error system
func (l *Logger) Error(operation string, err error) {
	log.Printf("[ERROR] %s.%s: %v", l.component, operation, err)
	errors.Get().ProcessingError(l.component, operation, err)
}

// DatabaseError logs a database error
func (l *Logger) DatabaseError(operation string, err error) {
	log.Printf("[ERROR] %s.%s: Database error: %v", l.component, operation, err)
	errors.Get().DatabaseError(l.component, operation, err)
}

// NetworkError logs a network error
func (l *Logger) NetworkError(operation string, err error) {
	log.Printf("[ERROR] %s.%s: Network error: %v", l.component, operation, err)
	errors.Get().NetworkError(l.component, operation, err)
}

// Debug logs a debug message (only if DEBUG env var is set)
func (l *Logger) Debug(operation, message string, args ...interface{}) {
	// Debug messages are not sent to error system
	msg := fmt.Sprintf(message, args...)
	log.Printf("[DEBUG] %s.%s: %s", l.component, operation, msg)
}