package errors

import (
	"fmt"
	"log"
	"strings"
)

// MigrateFromOldSystems migrates from the old error collection systems to the unified one
func MigrateFromOldSystems() {
	log.Println(" Migrating to Unified Error System...")

	// Hook into standard log output to capture legacy log.Printf errors
	installLogInterceptor()

	log.Println("[OK] Error system migration complete")
}

// installLogInterceptor intercepts log.Printf calls that contain error patterns
func installLogInterceptor() {
	// This is called by the custom log writer in main.go
	// We'll handle the interception there
}

// ParseLogMessage extracts error information from log messages
func ParseLogMessage(logMsg string) (errorType ErrorType, component, operation, message string, isError bool) {
	// Remove timestamp if present
	cleanMsg := logMsg
	if idx := strings.Index(logMsg, "] "); idx != -1 {
		cleanMsg = logMsg[idx+2:]
	}

	// Detect error patterns
	if strings.Contains(logMsg, "Error 1452") || strings.Contains(logMsg, "foreign key constraint") {
		return ErrorTypeConstraint, "Database", "ForeignKey", cleanMsg, true
	}

	if strings.Contains(logMsg, "Error 1062") || strings.Contains(logMsg, "Duplicate entry") {
		// Duplicate key errors are not real errors - they happen during normal operation
		// when multiple workers try to insert the same block
		return "", "", "", "", false
	}

	if strings.Contains(logMsg, "SLOW SQL") || strings.Contains(logMsg, "slow query") {
		return ErrorTypeSlow, "Database", "SlowQuery", cleanMsg, true
	}

	if strings.Contains(logMsg, "failed to process") {
		return ErrorTypeProcessing, "Processor", "ProcessBlock", cleanMsg, true
	}

	if strings.Contains(logMsg, "Warning:") || strings.Contains(logMsg, "[WARNING]") {
		// Extract component and operation from warning
		if strings.Contains(logMsg, "failed to find existing transaction") {
			return ErrorTypeWarning, "Transaction", "Lookup", cleanMsg, true
		}
		if strings.Contains(logMsg, "Historical reference") {
			return ErrorTypeWarning, "Transaction", "HistoricalRef", cleanMsg, true
		}
		return ErrorTypeWarning, "General", "Warning", cleanMsg, true
	}

	if strings.Contains(logMsg, "Error:") || strings.Contains(logMsg, "[ERROR]") || strings.Contains(logMsg, "failed") {
		// Extract component from error context
		component = "General"
		operation = "Error"

		if strings.Contains(logMsg, "BlockFetch") {
			component = "BlockFetch"
			operation = "Fetch"
		} else if strings.Contains(logMsg, "ChainSync") {
			component = "ChainSync"
			operation = "Sync"
		} else if strings.Contains(logMsg, "connection") {
			component = "Network"
			operation = "Connection"
		} else if strings.Contains(logMsg, "transaction") {
			component = "Transaction"
			operation = "Process"
		} else if strings.Contains(logMsg, "block") {
			component = "Block"
			operation = "Process"
		}

		return ErrorTypeProcessing, component, operation, cleanMsg, true
	}

	if strings.Contains(logMsg, "constraint") {
		return ErrorTypeConstraint, "Database", "Constraint", cleanMsg, true
	}

	if strings.Contains(logMsg, "deadlock") {
		return ErrorTypeDatabase, "Database", "Deadlock", cleanMsg, true
	}

	if strings.Contains(logMsg, "connection refused") || strings.Contains(logMsg, "connection reset") {
		return ErrorTypeNetwork, "Network", "Connection", cleanMsg, true
	}

	if strings.Contains(logMsg, "timeout") {
		return ErrorTypeNetwork, "Network", "Timeout", cleanMsg, true
	}

	return "", "", "", "", false
}

// ConvertProcessorError converts processor-specific errors to unified format
func ConvertProcessorError(component, operation string, err error) {
	if err == nil {
		return
	}

	ues := Get()
	errStr := err.Error()

	// Categorize based on error content
	switch {
	case strings.Contains(errStr, "foreign key"):
		ues.ConstraintError(component, operation, err, "")
	case strings.Contains(errStr, "duplicate"):
		ues.ConstraintError(component, operation, err, "")
	case strings.Contains(errStr, "constraint"):
		ues.ConstraintError(component, operation, err, "")
	case strings.Contains(errStr, "connection"):
		ues.NetworkError(component, operation, err)
	case strings.Contains(errStr, "timeout"):
		ues.NetworkError(component, operation, err)
	case strings.Contains(errStr, "validation"):
		ues.ValidationError(component, operation, err)
	default:
		ues.ProcessingError(component, operation, err)
	}
}

// WrapError wraps an error with component context and logs it
func WrapError(component, operation string, err error) error {
	if err == nil {
		return nil
	}

	// Log to unified system
	ConvertProcessorError(component, operation, err)

	// Return wrapped error
	return fmt.Errorf("%s.%s: %w", component, operation, err)
}
