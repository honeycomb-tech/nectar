package errors

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

// StderrInterceptor captures all stderr output and routes MySQL-related errors to the unified error system
type StderrInterceptor struct {
	originalStderr *os.File
	reader         *os.File
	writer         *os.File
	errorSystem    *UnifiedErrorSystem
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// NewStderrInterceptor creates a new stderr interceptor
func NewStderrInterceptor(errorSystem *UnifiedErrorSystem) (*StderrInterceptor, error) {
	// Create a pipe to intercept stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	interceptor := &StderrInterceptor{
		originalStderr: os.Stderr,
		reader:         reader,
		writer:         writer,
		errorSystem:    errorSystem,
		stopChan:       make(chan struct{}),
	}

	// Redirect stderr to our pipe
	os.Stderr = writer

	// Start the interceptor goroutine
	interceptor.wg.Add(1)
	go interceptor.intercept()

	return interceptor, nil
}

// intercept reads from the pipe and processes MySQL-related errors
func (si *StderrInterceptor) intercept() {
	defer si.wg.Done()

	scanner := bufio.NewScanner(si.reader)
	for scanner.Scan() {
		select {
		case <-si.stopChan:
			return
		default:
			line := scanner.Text()
			
			// Process the line
			si.processLine(line)
			
			// Also write to original stderr for debugging
			si.originalStderr.WriteString(line + "\n")
		}
	}
}

// processLine processes a single line from stderr
func (si *StderrInterceptor) processLine(line string) {
	// Check if this is a MySQL-related error
	if si.isMySQLError(line) {
		si.processMySQLError(line)
	}
}

// isMySQLError checks if the line contains MySQL-related error patterns
func (si *StderrInterceptor) isMySQLError(line string) bool {
	// Skip duplicate key errors - they're not real errors
	if strings.Contains(line, "Error 1062") || strings.Contains(line, "Duplicate entry") {
		return false
	}
	
	mysqlPatterns := []string{
		// MySQL driver patterns
		"packets.go:",
		"connection.go:",
		"driver.go:",
		"connector.go:",
		"mysql:",
		"[mysql]",
		
		// Common MySQL error patterns
		"unexpected EOF",
		"busy buffer",
		"unexpected seq nr",
		"bad connection",
		"broken pipe",
		"invalid connection",
		"commands out of sync",
		"Packets out of order",
		"connection refused",
		"connection reset",
		
		// MySQL error codes (except 1062 which we filter above)
		"Error 1452",  // Foreign key constraint
		"Error 1451",  // Cannot delete or update a parent row
		"Error 1054",  // Unknown column
		"Error 1146",  // Table doesn't exist
		
		// TiDB specific patterns
		"tidb:",
		"[tidb]",
		"TiDB",
	}

	lineLower := strings.ToLower(line)
	for _, pattern := range mysqlPatterns {
		if strings.Contains(lineLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// processMySQLError processes MySQL-related errors from stderr
func (si *StderrInterceptor) processMySQLError(line string) {
	component := "MySQL"
	operation := "Unknown"
	errorType := ErrorTypeDatabase
	message := line

	lineLower := strings.ToLower(line)

	// Categorize the error
	switch {
	case strings.Contains(lineLower, "packets.go"):
		component = "MySQL.Packets"
		if strings.Contains(lineLower, "unexpected seq nr") {
			operation = "SequenceError"
			errorType = ErrorTypeNetwork
			// Extract sequence numbers
			if seqInfo := extractSequenceNumbers(line); seqInfo != "" {
				message = "Packet sequence mismatch: " + seqInfo
			} else {
				message = "Packet sequence number mismatch"
			}
		} else if strings.Contains(lineLower, "unexpected eof") {
			operation = "EOFError"
			errorType = ErrorTypeNetwork
			message = "Unexpected EOF while reading packet"
		} else if strings.Contains(lineLower, "packets out of order") {
			operation = "PacketOrder"
			errorType = ErrorTypeNetwork
			message = "Packets received out of order"
		}

	case strings.Contains(lineLower, "connection.go"):
		component = "MySQL.Connection"
		if strings.Contains(lineLower, "busy buffer") {
			operation = "BusyBuffer"
			errorType = ErrorTypeNetwork
			message = "Connection buffer busy - possible protocol desync"
		} else if strings.Contains(lineLower, "bad connection") {
			operation = "BadConnection"
			errorType = ErrorTypeNetwork
			message = "Connection in bad state"
		} else if strings.Contains(lineLower, "connection reset") {
			operation = "ConnectionReset"
			errorType = ErrorTypeNetwork
			message = "Connection reset by peer"
		}

	case strings.Contains(lineLower, "commands out of sync"):
		component = "MySQL.Protocol"
		operation = "CommandSync"
		errorType = ErrorTypeDatabase
		message = "Commands out of sync - protocol error"

	case strings.Contains(lineLower, "broken pipe"):
		component = "MySQL.Network"
		operation = "BrokenPipe"
		errorType = ErrorTypeNetwork
		message = "Connection lost (broken pipe)"

	case strings.Contains(lineLower, "connection refused"):
		component = "MySQL.Network"
		operation = "ConnectionRefused"
		errorType = ErrorTypeNetwork
		message = "Connection refused by MySQL server"

	case strings.Contains(lineLower, "timeout"):
		component = "MySQL.Network"
		operation = "Timeout"
		errorType = ErrorTypeNetwork
		if strings.Contains(lineLower, "i/o timeout") {
			message = "Network I/O timeout"
		} else if strings.Contains(lineLower, "read timeout") {
			message = "Read timeout exceeded"
		} else if strings.Contains(lineLower, "write timeout") {
			message = "Write timeout exceeded"
		} else {
			message = "Connection timeout"
		}

	case strings.Contains(lineLower, "too many connections"):
		component = "MySQL.Connection"
		operation = "ConnectionLimit"
		errorType = ErrorTypeDatabase
		message = "Maximum connection limit reached"

	case strings.Contains(lineLower, "deadlock"):
		component = "MySQL.Transaction"
		operation = "Deadlock"
		errorType = ErrorTypeDatabase
		message = "Transaction deadlock detected"
	}

	// Clean up the message
	message = cleanupErrorMessage(message)

	// Route to unified error system
	if si.errorSystem != nil {
		si.errorSystem.LogError(errorType, component, operation, message)
	}
}

// extractSequenceNumbers extracts sequence number information from error messages
func extractSequenceNumbers(line string) string {
	// Look for patterns like "got 123 want 124" or "expected 123, got 124"
	words := strings.Fields(line)
	
	for i := 0; i < len(words)-3; i++ {
		if words[i] == "got" && words[i+2] == "want" {
			return "got " + words[i+1] + ", expected " + words[i+3]
		}
		if words[i] == "expected" && words[i+2] == "got" {
			return "expected " + words[i+1] + ", got " + words[i+3]
		}
	}
	
	// Look for number patterns after "seq"
	for i := 0; i < len(words)-1; i++ {
		if strings.Contains(words[i], "seq") {
			if i+1 < len(words) {
				return "sequence " + words[i+1]
			}
		}
	}
	
	return ""
}

// cleanupErrorMessage removes file paths and line numbers from error messages
func cleanupErrorMessage(msg string) string {
	// Remove file:line references
	parts := strings.Split(msg, " ")
	var cleaned []string
	
	for _, part := range parts {
		// Skip parts that look like file:line references
		if strings.Contains(part, ".go:") && containsNumber(part) {
			continue
		}
		// Skip timestamp patterns
		if len(part) > 8 && part[2] == ':' && part[5] == ':' {
			continue
		}
		cleaned = append(cleaned, part)
	}
	
	result := strings.Join(cleaned, " ")
	
	// Remove common prefixes
	result = strings.TrimPrefix(result, "[mysql] ")
	result = strings.TrimPrefix(result, "mysql: ")
	result = strings.TrimPrefix(result, "[MySQL] ")
	
	// Trim whitespace
	result = strings.TrimSpace(result)
	
	return result
}

// containsNumber checks if a string contains numeric characters
func containsNumber(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// Stop stops the stderr interceptor
func (si *StderrInterceptor) Stop() {
	close(si.stopChan)
	
	// Restore original stderr
	os.Stderr = si.originalStderr
	
	// Close the writer to signal EOF to the reader
	si.writer.Close()
	
	// Wait for the interceptor goroutine to finish
	si.wg.Wait()
	
	// Close the reader
	si.reader.Close()
}

// SetupStderrInterceptor sets up stderr interception for MySQL errors
func SetupStderrInterceptor() (*StderrInterceptor, error) {
	errorSystem := Get()
	return NewStderrInterceptor(errorSystem)
}