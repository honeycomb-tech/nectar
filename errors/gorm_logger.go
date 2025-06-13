package errors

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// UnifiedGormLogger implements GORM's logger interface and forwards to unified error system
type UnifiedGormLogger struct {
	SlowThreshold         time.Duration
	IgnoreRecordNotFound  bool
	LogLevel              gormlogger.LogLevel
}

// NewGormLogger creates a new GORM logger that uses the unified error system
func NewGormLogger() gormlogger.Interface {
	return &UnifiedGormLogger{
		SlowThreshold:        time.Second,
		IgnoreRecordNotFound: true,
		LogLevel:            gormlogger.Warn,
	}
}

// LogMode sets the log level
func (l *UnifiedGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs info level messages
func (l *UnifiedGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		Get().LogError(ErrorTypeSystem, "GORM", "Info", fmt.Sprintf(msg, data...))
	}
}

// Warn logs warning level messages
func (l *UnifiedGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		Get().Warning("GORM", "Warning", fmt.Sprintf(msg, data...))
	}
}

// Error logs error level messages
func (l *UnifiedGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		Get().LogError(ErrorTypeDatabase, "GORM", "Error", fmt.Sprintf(msg, data...))
	}
}

// Trace logs SQL queries and detects slow queries and errors
func (l *UnifiedGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, _ := fc()
	
	// Extract operation type from SQL
	operation := "Query"
	sqlLower := strings.ToLower(sql)
	if strings.HasPrefix(sqlLower, "insert") {
		operation = "Insert"
	} else if strings.HasPrefix(sqlLower, "update") {
		operation = "Update"
	} else if strings.HasPrefix(sqlLower, "delete") {
		operation = "Delete"
	} else if strings.HasPrefix(sqlLower, "select") {
		operation = "Select"
	} else if strings.HasPrefix(sqlLower, "create") {
		operation = "Create"
	} else if strings.HasPrefix(sqlLower, "drop") {
		operation = "Drop"
	} else if strings.HasPrefix(sqlLower, "alter") {
		operation = "Alter"
	}

	// Extract table name if possible
	table := extractTableName(sql)
	component := "GORM"
	if table != "" {
		component = fmt.Sprintf("GORM.%s", table)
	}

	switch {
	case err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.IgnoreRecordNotFound):
		// Handle different types of database errors
		errStr := err.Error()
		
		// Check for specific MySQL/TiDB errors
		if strings.Contains(errStr, "Error 1452") || strings.Contains(errStr, "foreign key constraint fails") {
			// Foreign key constraint error
			Get().ConstraintError(component, operation, err, fmt.Sprintf("SQL: %s", truncateSQL(sql)))
		} else if strings.Contains(errStr, "Error 1062") || strings.Contains(errStr, "Duplicate entry") {
			// Duplicate key error
			Get().ConstraintError(component, operation, err, fmt.Sprintf("SQL: %s", truncateSQL(sql)))
		} else if strings.Contains(errStr, "Error 1213") || strings.Contains(errStr, "Deadlock found") {
			// Deadlock error
			Get().DatabaseError(component, operation, fmt.Errorf("deadlock detected: %w", err))
		} else if strings.Contains(errStr, "connection") || strings.Contains(errStr, "Can't connect") {
			// Connection error
			Get().NetworkError(component, operation, err)
		} else {
			// General database error
			Get().DatabaseError(component, operation, err)
		}
		
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		// Slow query
		Get().SlowQuery(component, operation, elapsed, truncateSQL(sql))
		
	case l.LogLevel == gormlogger.Info:
		// Normal query logging (only in info mode)
		// Don't log every query to avoid noise
	}
}

// extractTableName attempts to extract the table name from SQL
func extractTableName(sql string) string {
	sqlLower := strings.ToLower(sql)
	
	// Common patterns to extract table name
	patterns := []struct {
		prefix string
		suffix string
	}{
		{"from `", "`"},
		{"from ", " "},
		{"into `", "`"},
		{"into ", " "},
		{"update `", "`"},
		{"update ", " "},
		{"table `", "`"},
		{"table ", " "},
	}
	
	for _, p := range patterns {
		if idx := strings.Index(sqlLower, p.prefix); idx != -1 {
			start := idx + len(p.prefix)
			if end := strings.Index(sqlLower[start:], p.suffix); end != -1 {
				return sql[start : start+end]
			}
		}
	}
	
	return ""
}

// truncateSQL truncates long SQL queries for logging
func truncateSQL(sql string) string {
	const maxLength = 500
	if len(sql) <= maxLength {
		return sql
	}
	return sql[:maxLength] + "..."
}