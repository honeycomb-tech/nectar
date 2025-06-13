package processors

import (
	"fmt"
	"log"
	unifiederrors "nectar/errors"
	"os"
	"sync"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "Info"
	case SeverityWarning:
		return "Warning"
	case SeverityError:
		return "Error"
	case SeverityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// ErrorCategory represents the category of an error
type ErrorCategory int

const (
	CategoryDatabase ErrorCategory = iota
	CategoryNetwork
	CategoryProcessing
	CategoryValidation
	CategorySystem
)

func (c ErrorCategory) String() string {
	switch c {
	case CategoryDatabase:
		return "Database"
	case CategoryNetwork:
		return "Network"
	case CategoryProcessing:
		return "Processing"
	case CategoryValidation:
		return "Validation"
	case CategorySystem:
		return "System"
	default:
		return "Unknown"
	}
}

// CollectedError represents a single error occurrence
type CollectedError struct {
	Timestamp   time.Time
	Severity    ErrorSeverity
	Category    ErrorCategory
	Component   string
	Operation   string
	Message     string
	Context     string
	Count       int64
	LastSeen    time.Time
	FirstSeen   time.Time
}

// ErrorCollector collects and manages errors during processing
type ErrorCollector struct {
	errors      map[string]*CollectedError
	mutex       sync.RWMutex
	maxErrors   int
	logFile     *os.File
	lastFlush   time.Time
	flushPeriod time.Duration
}

var (
	globalErrorCollector *ErrorCollector
	once                sync.Once
)

// GetGlobalErrorCollector returns the singleton error collector
func GetGlobalErrorCollector() *ErrorCollector {
	once.Do(func() {
		globalErrorCollector = NewErrorCollector(1000, 5*time.Minute)
	})
	return globalErrorCollector
}

// NewErrorCollector creates a new error collector
func NewErrorCollector(maxErrors int, flushPeriod time.Duration) *ErrorCollector {
	ec := &ErrorCollector{
		errors:      make(map[string]*CollectedError),
		maxErrors:   maxErrors,
		lastFlush:   time.Now(),
		flushPeriod: flushPeriod,
	}

	// Open or create error log file
	logFile, err := os.OpenFile("errors.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Failed to open error log file: %v", err)
	} else {
		ec.logFile = logFile
	}

	// Start periodic flush
	go ec.periodicFlush()

	return ec
}

// periodicFlush periodically writes errors to file
func (ec *ErrorCollector) periodicFlush() {
	ticker := time.NewTicker(ec.flushPeriod)
	defer ticker.Stop()

	for range ticker.C {
		ec.Flush()
	}
}

// CollectError collects an error with full context
func (ec *ErrorCollector) CollectError(
	severity ErrorSeverity,
	category ErrorCategory,
	component string,
	operation string,
	message string,
	context string,
) *CollectedError {
	// Create error signature for deduplication
	signature := fmt.Sprintf("%s:%s:%s:%s:%s", 
		severity.String(), category.String(), component, operation, message)
	
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	
	now := time.Now()
	
	// Check if this error already exists
	if existing, exists := ec.errors[signature]; exists {
		existing.Count++
		existing.LastSeen = now
		return existing
	}
	
	// Create new error entry
	err := &CollectedError{
		Timestamp: now,
		Severity:  severity,
		Category:  category,
		Component: component,
		Operation: operation,
		Message:   message,
		Context:   context,
		Count:     1,
		FirstSeen: now,
		LastSeen:  now,
	}
	
	// Check if we've reached max errors
	if len(ec.errors) >= ec.maxErrors {
		// Remove oldest error
		var oldestKey string
		var oldestTime time.Time
		for key, e := range ec.errors {
			if oldestTime.IsZero() || e.LastSeen.Before(oldestTime) {
				oldestKey = key
				oldestTime = e.LastSeen
			}
		}
		delete(ec.errors, oldestKey)
	}
	
	ec.errors[signature] = err
	
	// Check if we need to flush
	if time.Since(ec.lastFlush) > ec.flushPeriod {
		go ec.Flush()
	}
	
	return err
}

// ProcessingError is a convenience method for processing errors
func (ec *ErrorCollector) ProcessingError(component, operation, message, context string) {
	ec.CollectError(SeverityError, CategoryProcessing, component, operation, message, context)
	// Forward to unified system
	unifiederrors.Get().LogError(unifiederrors.ErrorTypeProcessing, component, operation, message, context)
}

// ProcessingWarning is a convenience method for processing warnings
func (ec *ErrorCollector) ProcessingWarning(component, operation, message, context string) {
	ec.CollectError(SeverityWarning, CategoryProcessing, component, operation, message, context)
	// Forward to unified system
	unifiederrors.Get().Warning(component, operation, message)
}

// DatabaseError is a convenience method for database errors
func (ec *ErrorCollector) DatabaseError(component, operation, message, context string) {
	ec.CollectError(SeverityError, CategoryDatabase, component, operation, message, context)
	// Forward to unified system
	unifiederrors.Get().LogError(unifiederrors.ErrorTypeDatabase, component, operation, message, context)
}

// NetworkError is a convenience method for network errors
func (ec *ErrorCollector) NetworkError(component, operation, message, context string) {
	ec.CollectError(SeverityError, CategoryNetwork, component, operation, message, context)
	// Forward to unified system
	unifiederrors.Get().LogError(unifiederrors.ErrorTypeNetwork, component, operation, message, context)
}

// GetErrors returns a copy of current errors
func (ec *ErrorCollector) GetErrors() []CollectedError {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	
	errors := make([]CollectedError, 0, len(ec.errors))
	for _, err := range ec.errors {
		errors = append(errors, *err)
	}
	
	return errors
}

// GetErrorCount returns the total number of unique errors
func (ec *ErrorCollector) GetErrorCount() int {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	return len(ec.errors)
}

// GetTotalErrorCount returns the total count of all error occurrences
func (ec *ErrorCollector) GetTotalErrorCount() int64 {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	
	var total int64
	for _, err := range ec.errors {
		total += err.Count
	}
	
	return total
}

// Flush writes all errors to the log file
func (ec *ErrorCollector) Flush() error {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	
	if ec.logFile == nil {
		return nil
	}
	
	// Clear file and write header
	ec.logFile.Truncate(0)
	ec.logFile.Seek(0, 0)
	
	fmt.Fprintf(ec.logFile, "NECTAR INDEXER ERROR LOG\n")
	fmt.Fprintf(ec.logFile, "========================\n")
	fmt.Fprintf(ec.logFile, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(ec.logFile, "Total Errors: %d\n\n", len(ec.errors))
	
	// Group errors by severity
	errorsBySeverity := make(map[ErrorSeverity][]*CollectedError)
	for _, err := range ec.errors {
		errorsBySeverity[err.Severity] = append(errorsBySeverity[err.Severity], err)
	}
	
	// Write errors sorted by severity
	errorNum := 1
	for _, severity := range []ErrorSeverity{SeverityCritical, SeverityError, SeverityWarning, SeverityInfo} {
		errors := errorsBySeverity[severity]
		for _, err := range errors {
			fmt.Fprintf(ec.logFile, "[%d] %s - %s (Count: %d)\n",
				errorNum,
				err.FirstSeen.Format("2006-01-02 15:04:05"),
				err.Severity.String(),
				err.Count)
			fmt.Fprintf(ec.logFile, "    [WARNING] %s\n", err.Message)
			if err.Context != "" {
				fmt.Fprintf(ec.logFile, "    Context: %s\n", err.Context)
			}
			fmt.Fprintf(ec.logFile, "\n")
			errorNum++
		}
	}
	
	fmt.Fprintf(ec.logFile, "END OF ERROR LOG\n")
	ec.logFile.Sync()
	
	ec.lastFlush = time.Now()
	return nil
}

// Clear removes all collected errors
func (ec *ErrorCollector) Clear() {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.errors = make(map[string]*CollectedError)
}

// Close closes the error collector and flushes remaining errors
func (ec *ErrorCollector) Close() error {
	ec.Flush()
	if ec.logFile != nil {
		return ec.logFile.Close()
	}
	return nil
}

// GetRecentErrors returns errors from the last N minutes
func (ec *ErrorCollector) GetRecentErrors(minutes int) []CollectedError {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	recent := []CollectedError{}
	
	for _, err := range ec.errors {
		if err.LastSeen.After(cutoff) {
			recent = append(recent, *err)
		}
	}
	
	return recent
}

// GetErrorsByCategory returns all errors of a specific category
func (ec *ErrorCollector) GetErrorsByCategory(category ErrorCategory) []CollectedError {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	
	errors := []CollectedError{}
	for _, err := range ec.errors {
		if err.Category == category {
			errors = append(errors, *err)
		}
	}
	
	return errors
}

// GetErrorsBySeverity returns all errors of a specific severity
func (ec *ErrorCollector) GetErrorsBySeverity(severity ErrorSeverity) []CollectedError {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	
	errors := []CollectedError{}
	for _, err := range ec.errors {
		if err.Severity == severity {
			errors = append(errors, *err)
		}
	}
	
	return errors
}