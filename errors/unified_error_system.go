package errors

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorType represents the type of error for categorization
type ErrorType string

const (
	// Error types for dashboard display
	ErrorTypeDatabase    ErrorType = "Database"
	ErrorTypeNetwork     ErrorType = "Network"
	ErrorTypeProcessing  ErrorType = "Processing"
	ErrorTypeValidation  ErrorType = "Validation"
	ErrorTypeConstraint  ErrorType = "Constraint"
	ErrorTypeSlow        ErrorType = "Slow Query"
	ErrorTypeWarning     ErrorType = "Warning"
	ErrorTypeBlockFetch  ErrorType = "BlockFetch"
	ErrorTypeTransaction ErrorType = "Transaction"
	ErrorTypeConnection  ErrorType = "Connection"
	ErrorTypeSystem      ErrorType = "System"
)

// UnifiedError represents a single error with full context
type UnifiedError struct {
	ID          string
	Timestamp   time.Time
	Type        ErrorType
	Component   string
	Operation   string
	Message     string
	Details     string
	StackTrace  string
	Count       int64
	FirstSeen   time.Time
	LastSeen    time.Time
}

// UnifiedErrorSystem is the single source of truth for all errors
type UnifiedErrorSystem struct {
	// Core storage
	errors      map[string]*UnifiedError // Deduplicated by signature
	errorList   []*UnifiedError          // Ordered list for dashboard
	mutex       sync.RWMutex
	
	// Statistics
	totalErrors      atomic.Int64
	errorsByType     map[ErrorType]*atomic.Int64
	recentErrorRate  atomic.Int64 // Errors in last minute
	
	// Dashboard callback
	dashboardFunc    func(errorType, message string)
	dashboardEnabled atomic.Bool
	
	// Persistence
	logFile         *os.File
	persistencePath string
	
	// Configuration
	maxErrors       int
	maxListSize     int
	flushInterval   time.Duration
	
	// Metrics
	lastFlush       time.Time
	lastRateCalc    time.Time
}

var (
	globalUnifiedSystem *UnifiedErrorSystem
	systemOnce          sync.Once
)

// Initialize creates and returns the global unified error system
func Initialize(dashboardFunc func(errorType, message string)) *UnifiedErrorSystem {
	systemOnce.Do(func() {
		globalUnifiedSystem = &UnifiedErrorSystem{
			errors:          make(map[string]*UnifiedError),
			errorList:       make([]*UnifiedError, 0, 1000),
			errorsByType:    make(map[ErrorType]*atomic.Int64),
			maxErrors:       10000,      // Keep up to 10k unique errors
			maxListSize:     1000,       // Dashboard shows last 1k errors
			flushInterval:   5 * time.Minute,
			persistencePath: "unified_errors.log",
			dashboardFunc:   dashboardFunc,
		}
		
		// Initialize error type counters
		for _, errType := range []ErrorType{
			ErrorTypeDatabase, ErrorTypeNetwork, ErrorTypeProcessing,
			ErrorTypeValidation, ErrorTypeConstraint, ErrorTypeSlow,
			ErrorTypeWarning, ErrorTypeBlockFetch, ErrorTypeTransaction,
			ErrorTypeConnection, ErrorTypeSystem,
		} {
			globalUnifiedSystem.errorsByType[errType] = &atomic.Int64{}
		}
		
		globalUnifiedSystem.dashboardEnabled.Store(true)
		globalUnifiedSystem.initializePersistence()
		
		// Start background tasks
		go globalUnifiedSystem.backgroundMaintenance()
		
		log.Println("[OK] Unified Error System initialized")
	})
	
	return globalUnifiedSystem
}

// Get returns the global unified error system
func Get() *UnifiedErrorSystem {
	if globalUnifiedSystem == nil {
		panic("UnifiedErrorSystem not initialized. Call Initialize() first")
	}
	return globalUnifiedSystem
}

// SetDashboardCallback sets or updates the dashboard callback function
func (ues *UnifiedErrorSystem) SetDashboardCallback(dashboardFunc func(errorType, message string)) {
	ues.mutex.Lock()
	defer ues.mutex.Unlock()
	ues.dashboardFunc = dashboardFunc
	ues.dashboardEnabled.Store(dashboardFunc != nil)
}

// LogError logs an error with full context - main entry point for all errors
func (ues *UnifiedErrorSystem) LogError(errType ErrorType, component, operation, message string, details ...string) {
	// Create error signature for deduplication
	signature := fmt.Sprintf("%s:%s:%s:%s", errType, component, operation, message)
	
	// Capture stack trace for new errors
	stackTrace := ""
	if len(details) == 0 || !strings.Contains(details[0], "stack:") {
		stackTrace = ues.captureStackTrace(3) // Skip LogError, captureStackTrace, and caller
	}
	
	detailStr := strings.Join(details, " ")
	
	ues.mutex.Lock()
	defer ues.mutex.Unlock()
	
	now := time.Now()
	
	// Check if error already exists
	if existing, exists := ues.errors[signature]; exists {
		existing.Count++
		existing.LastSeen = now
		
		// Update details if new information provided
		if detailStr != "" && existing.Details != detailStr {
			existing.Details = detailStr
		}
	} else {
		// Create new error
		err := &UnifiedError{
			ID:         fmt.Sprintf("%d", now.UnixNano()),
			Timestamp:  now,
			Type:       errType,
			Component:  component,
			Operation:  operation,
			Message:    message,
			Details:    detailStr,
			StackTrace: stackTrace,
			Count:      1,
			FirstSeen:  now,
			LastSeen:   now,
		}
		
		// Add to map
		ues.errors[signature] = err
		
		// Add to list for dashboard (maintain max size)
		ues.errorList = append(ues.errorList, err)
		if len(ues.errorList) > ues.maxListSize {
			ues.errorList = ues.errorList[1:]
		}
		
		// Clean up if too many unique errors
		if len(ues.errors) > ues.maxErrors {
			ues.cleanupOldErrors()
		}
	}
	
	// Update statistics
	ues.totalErrors.Add(1)
	if counter, ok := ues.errorsByType[errType]; ok {
		counter.Add(1)
	}
	
	// Forward to dashboard if enabled
	if ues.dashboardEnabled.Load() && ues.dashboardFunc != nil {
		// Format message for dashboard
		dashboardMsg := fmt.Sprintf("%s.%s: %s", component, operation, message)
		if detailStr != "" {
			dashboardMsg += fmt.Sprintf(" (%s)", detailStr)
		}
		ues.dashboardFunc(string(errType), dashboardMsg)
	}
	
	// Log to file for analysis
	ues.logToFile(errType, component, operation, message, detailStr)
}

// Convenience methods for common error types
func (ues *UnifiedErrorSystem) DatabaseError(component, operation string, err error) {
	ues.LogError(ErrorTypeDatabase, component, operation, err.Error())
}

func (ues *UnifiedErrorSystem) NetworkError(component, operation string, err error) {
	ues.LogError(ErrorTypeNetwork, component, operation, err.Error())
}

func (ues *UnifiedErrorSystem) ProcessingError(component, operation string, err error) {
	ues.LogError(ErrorTypeProcessing, component, operation, err.Error())
}

func (ues *UnifiedErrorSystem) ValidationError(component, operation string, err error) {
	ues.LogError(ErrorTypeValidation, component, operation, err.Error())
}

func (ues *UnifiedErrorSystem) ConstraintError(component, operation string, err error, details string) {
	ues.LogError(ErrorTypeConstraint, component, operation, err.Error(), details)
}

func (ues *UnifiedErrorSystem) SlowQuery(component, operation string, duration time.Duration, query string) {
	msg := fmt.Sprintf("Query took %v", duration)
	ues.LogError(ErrorTypeSlow, component, operation, msg, query)
}

func (ues *UnifiedErrorSystem) Warning(component, operation, message string) {
	ues.LogError(ErrorTypeWarning, component, operation, message)
}

// captureStackTrace captures the current stack trace
func (ues *UnifiedErrorSystem) captureStackTrace(skip int) string {
	const maxDepth = 10
	var builder strings.Builder
	
	for i := skip; i < skip+maxDepth; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		// Skip runtime and system functions
		fnName := fn.Name()
		if strings.Contains(fnName, "runtime.") {
			continue
		}
		
		// Format: function (file:line)
		builder.WriteString(fmt.Sprintf("  %s (%s:%d)\n", fnName, file, line))
	}
	
	return builder.String()
}

// GetRecentErrors returns errors from the last N minutes
func (ues *UnifiedErrorSystem) GetRecentErrors(minutes int) []*UnifiedError {
	ues.mutex.RLock()
	defer ues.mutex.RUnlock()
	
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	recent := make([]*UnifiedError, 0)
	
	// Return from the ordered list (most recent errors)
	for i := len(ues.errorList) - 1; i >= 0; i-- {
		if ues.errorList[i].LastSeen.After(cutoff) {
			recent = append(recent, ues.errorList[i])
		}
	}
	
	return recent
}

// GetErrorsByType returns all errors of a specific type
func (ues *UnifiedErrorSystem) GetErrorsByType(errType ErrorType) []*UnifiedError {
	ues.mutex.RLock()
	defer ues.mutex.RUnlock()
	
	errors := make([]*UnifiedError, 0)
	for _, err := range ues.errorList {
		if err.Type == errType {
			errors = append(errors, err)
		}
	}
	
	return errors
}

// GetStatistics returns current error statistics
func (ues *UnifiedErrorSystem) GetStatistics() map[string]interface{} {
	ues.mutex.RLock()
	defer ues.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_errors"] = ues.totalErrors.Load()
	stats["unique_errors"] = len(ues.errors)
	stats["recent_errors"] = len(ues.errorList)
	
	// Error counts by type
	typeCounts := make(map[string]int64)
	for errType, counter := range ues.errorsByType {
		typeCounts[string(errType)] = counter.Load()
	}
	stats["by_type"] = typeCounts
	
	// Calculate error rate (errors per minute)
	now := time.Now()
	if ues.lastRateCalc.IsZero() {
		ues.lastRateCalc = now
	}
	
	timeDiff := now.Sub(ues.lastRateCalc).Minutes()
	if timeDiff > 0 {
		recentCount := ues.recentErrorRate.Load()
		stats["error_rate"] = float64(recentCount) / timeDiff
		ues.recentErrorRate.Store(0)
		ues.lastRateCalc = now
	}
	
	return stats
}

// cleanupOldErrors removes oldest errors when limit exceeded
func (ues *UnifiedErrorSystem) cleanupOldErrors() {
	// Find oldest 10% of errors
	toRemove := len(ues.errors) / 10
	if toRemove < 1 {
		toRemove = 1
	}
	
	// Sort by last seen and remove oldest
	type errorAge struct {
		signature string
		lastSeen  time.Time
	}
	
	ages := make([]errorAge, 0, len(ues.errors))
	for sig, err := range ues.errors {
		ages = append(ages, errorAge{sig, err.LastSeen})
	}
	
	// Simple selection of oldest errors
	for i := 0; i < toRemove && i < len(ages); i++ {
		var oldestIdx int
		oldestTime := ages[0].lastSeen
		
		for j := 1; j < len(ages); j++ {
			if ages[j].lastSeen.Before(oldestTime) {
				oldestIdx = j
				oldestTime = ages[j].lastSeen
			}
		}
		
		// Remove from map
		delete(ues.errors, ages[oldestIdx].signature)
		
		// Remove from slice
		ages[oldestIdx] = ages[len(ages)-1]
		ages = ages[:len(ages)-1]
	}
}

// initializePersistence sets up the log file
func (ues *UnifiedErrorSystem) initializePersistence() {
	var err error
	ues.logFile, err = os.OpenFile(ues.persistencePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open unified error log: %v", err)
	}
}

// logToFile writes error to persistent log
func (ues *UnifiedErrorSystem) logToFile(errType ErrorType, component, operation, message, details string) {
	if ues.logFile == nil {
		return
	}
	
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logEntry := fmt.Sprintf("[%s] %s | %s.%s | %s", timestamp, errType, component, operation, message)
	if details != "" {
		logEntry += " | " + details
	}
	logEntry += "\n"
	
	ues.logFile.WriteString(logEntry)
}

// backgroundMaintenance performs periodic tasks
func (ues *UnifiedErrorSystem) backgroundMaintenance() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		// Update error rate
		ues.recentErrorRate.Add(1)
		
		// Flush log file
		if ues.logFile != nil {
			ues.logFile.Sync()
		}
		
		// Clean up old errors if needed
		ues.mutex.RLock()
		errorCount := len(ues.errors)
		ues.mutex.RUnlock()
		
		if errorCount > ues.maxErrors*9/10 {
			ues.mutex.Lock()
			ues.cleanupOldErrors()
			ues.mutex.Unlock()
		}
	}
}

// Close cleanly shuts down the error system
func (ues *UnifiedErrorSystem) Close() error {
	if ues.logFile != nil {
		ues.logFile.Sync()
		return ues.logFile.Close()
	}
	return nil
}