package errors

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// MySQLLogInterceptor captures MySQL driver logs and routes them to the unified error system
type MySQLLogInterceptor struct {
	originalWriter io.Writer
	errorSystem    *UnifiedErrorSystem
	mu             sync.Mutex
}

// NewMySQLLogInterceptor creates a new MySQL log interceptor
func NewMySQLLogInterceptor(errorSystem *UnifiedErrorSystem) *MySQLLogInterceptor {
	return &MySQLLogInterceptor{
		originalWriter: os.Stderr, // MySQL driver writes to stderr by default
		errorSystem:    errorSystem,
	}
}

// Write implements io.Writer interface to intercept MySQL driver logs
func (mli *MySQLLogInterceptor) Write(p []byte) (n int, err error) {
	mli.mu.Lock()
	defer mli.mu.Unlock()

	// Convert to string for analysis
	logMsg := string(p)

	// Parse MySQL driver log messages
	if mli.isMySQLDriverLog(logMsg) {
		mli.processMySQLLog(logMsg)
	}

	// Always return success to prevent MySQL driver from retrying
	return len(p), nil
}

// isMySQLDriverLog checks if the log message is from MySQL driver
func (mli *MySQLLogInterceptor) isMySQLDriverLog(msg string) bool {
	// MySQL driver logs typically contain these patterns
	mysqlPatterns := []string{
		"packets.go:",
		"connection.go:",
		"mysql:",
		"MySQL",
		"unexpected EOF",
		"busy buffer",
		"unexpected seq nr",
		"bad connection",
		"broken pipe",
		"invalid connection",
	}

	msgLower := strings.ToLower(msg)
	for _, pattern := range mysqlPatterns {
		if strings.Contains(msgLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// processMySQLLog processes MySQL driver log messages
func (mli *MySQLLogInterceptor) processMySQLLog(msg string) {
	// Extract key information from the log message
	component := "MySQL"
	operation := "Connection"
	errorType := ErrorTypeDatabase

	// Determine specific error type and details
	msgLower := strings.ToLower(msg)

	switch {
	case strings.Contains(msgLower, "packets.go"):
		component = "MySQL.Packets"
		if strings.Contains(msgLower, "unexpected seq nr") {
			operation = "SequenceError"
			errorType = ErrorTypeNetwork
			// Extract sequence numbers if possible
			if seqInfo := extractSequenceInfo(msg); seqInfo != "" {
				msg = fmt.Sprintf("Packet sequence error: %s", seqInfo)
			}
		} else if strings.Contains(msgLower, "unexpected eof") {
			operation = "EOFError"
			errorType = ErrorTypeNetwork
			msg = "Unexpected EOF while reading packet"
		}

	case strings.Contains(msgLower, "connection.go"):
		component = "MySQL.Connection"
		if strings.Contains(msgLower, "busy buffer") {
			operation = "BusyBuffer"
			errorType = ErrorTypeNetwork
			msg = "Connection buffer is busy - possible protocol sync issue"
		} else if strings.Contains(msgLower, "bad connection") {
			operation = "BadConnection"
			errorType = ErrorTypeNetwork
			msg = "Connection is in bad state"
		}

	case strings.Contains(msgLower, "broken pipe"):
		component = "MySQL.Network"
		operation = "BrokenPipe"
		errorType = ErrorTypeNetwork
		msg = "Connection broken (broken pipe)"

	case strings.Contains(msgLower, "invalid connection"):
		component = "MySQL.Connection"
		operation = "InvalidConnection"
		errorType = ErrorTypeDatabase
		msg = "Invalid database connection"

	case strings.Contains(msgLower, "timeout"):
		component = "MySQL.Network"
		operation = "Timeout"
		errorType = ErrorTypeNetwork
		msg = extractTimeoutInfo(msg)

	case strings.Contains(msgLower, "deadlock"):
		component = "MySQL.Transaction"
		operation = "Deadlock"
		errorType = ErrorTypeDatabase
		msg = "Database deadlock detected"

	case strings.Contains(msgLower, "too many connections"):
		component = "MySQL.Connection"
		operation = "ConnectionLimit"
		errorType = ErrorTypeDatabase
		msg = "Too many connections to database"
	}

	// Clean up the message
	msg = cleanMySQLLogMessage(msg)

	// Route to unified error system
	if mli.errorSystem != nil {
		mli.errorSystem.LogError(errorType, component, operation, msg)
	}
}

// extractSequenceInfo extracts sequence number information from the log
func extractSequenceInfo(msg string) string {
	// Look for patterns like "got 123 want 124"
	parts := strings.Fields(msg)
	for i, part := range parts {
		if part == "got" && i+3 < len(parts) && parts[i+2] == "want" {
			return fmt.Sprintf("got %s, expected %s", parts[i+1], parts[i+3])
		}
	}
	return ""
}

// extractTimeoutInfo extracts timeout information from the log
func extractTimeoutInfo(msg string) string {
	if strings.Contains(msg, "i/o timeout") {
		return "Network I/O timeout"
	} else if strings.Contains(msg, "read timeout") {
		return "Read timeout exceeded"
	} else if strings.Contains(msg, "write timeout") {
		return "Write timeout exceeded"
	}
	return "Connection timeout"
}

// cleanMySQLLogMessage removes unnecessary parts from MySQL log messages
func cleanMySQLLogMessage(msg string) string {
	// Remove timestamp if present
	if idx := strings.Index(msg, "]"); idx != -1 && idx < 30 {
		msg = strings.TrimSpace(msg[idx+1:])
	}

	// Remove file:line information
	if idx := strings.Index(msg, ".go:"); idx != -1 {
		if spaceIdx := strings.LastIndex(msg[:idx], " "); spaceIdx != -1 {
			prefix := msg[:spaceIdx]
			// Find the end of file:line pattern
			endIdx := idx + 3
			for endIdx < len(msg) && msg[endIdx] >= '0' && msg[endIdx] <= '9' {
				endIdx++
			}
			if endIdx < len(msg) {
				msg = prefix + msg[endIdx:]
			}
		}
	}

	// Remove MySQL prefix if present
	msg = strings.TrimPrefix(msg, "[mysql] ")
	msg = strings.TrimPrefix(msg, "mysql: ")

	// Clean up whitespace
	msg = strings.TrimSpace(msg)

	return msg
}

// InstallMySQLLogger installs the MySQL log interceptor globally
func InstallMySQLLogger() {
	// Get the unified error system instance
	// errorSystem := Get()

	// Create the interceptor
	// interceptor := NewMySQLLogInterceptor(errorSystem)

	// REMOVED: log.SetOutput(interceptor) - This was redirecting ALL logs
	// MySQL driver errors come through stderr, which we handle separately
	// The stderr interceptor will catch MySQL-specific errors without
	// interfering with normal application logging
}

// MySQLLogWriter is a specialized writer that captures MySQL-specific patterns
type MySQLLogWriter struct {
	interceptor *MySQLLogInterceptor
}

// Write implements io.Writer
func (mlw *MySQLLogWriter) Write(p []byte) (n int, err error) {
	return mlw.interceptor.Write(p)
}

// SetupMySQLLogging configures MySQL driver logging to use our unified error system
func SetupMySQLLogging() {
	// Skip MySQL logging setup for now - it was interfering with normal operation
	// The unified error system will still capture database errors through GORM
	// TODO: Implement a more targeted approach that doesn't redirect all logs
	
	// Log that MySQL logging setup is currently disabled
	log.Println("[INFO] MySQL driver logging interception temporarily disabled")
}