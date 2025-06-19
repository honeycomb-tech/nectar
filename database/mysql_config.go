package database

import (
	"context"
	"database/sql/driver"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"nectar/errors"
)

// MySQLErrorLogger is a custom MySQL logger that routes errors to the unified error system
type MySQLErrorLogger struct {
	mu sync.Mutex
}

// Print implements the mysql.Logger interface
func (mel *MySQLErrorLogger) Print(v ...interface{}) {
	mel.mu.Lock()
	defer mel.mu.Unlock()

	// Convert arguments to string
	var msg string
	if len(v) > 0 {
		format, ok := v[0].(string)
		if ok && len(v) > 1 {
			msg = fmt.Sprintf(format, v[1:]...)
		} else {
			parts := make([]string, len(v))
			for i, arg := range v {
				parts[i] = fmt.Sprint(arg)
			}
			msg = strings.Join(parts, " ")
		}
	}

	// Route to unified error system
	mel.processMySQLLog(msg)
}

// processMySQLLog processes MySQL driver log messages
func (mel *MySQLErrorLogger) processMySQLLog(msg string) {
	// Filter out duplicate key errors immediately
	if strings.Contains(msg, "Error 1062") || strings.Contains(msg, "Duplicate entry") {
		return // Don't log duplicate key errors at all
	}

	// Determine error type and component
	component := "MySQL.Driver"
	operation := "Unknown"
	errorType := errors.ErrorTypeDatabase

	msgLower := strings.ToLower(msg)

	switch {
	case strings.Contains(msgLower, "packets.go"):
		component = "MySQL.Packets"
		operation = "PacketError"
		errorType = errors.ErrorTypeNetwork

	case strings.Contains(msgLower, "connection"):
		component = "MySQL.Connection"
		operation = "ConnectionError"
		errorType = errors.ErrorTypeNetwork

	case strings.Contains(msgLower, "timeout"):
		component = "MySQL.Network"
		operation = "Timeout"
		errorType = errors.ErrorTypeNetwork

	case strings.Contains(msgLower, "deadlock"):
		component = "MySQL.Transaction"
		operation = "Deadlock"
		errorType = errors.ErrorTypeDatabase

	case strings.Contains(msgLower, "syntax"):
		component = "MySQL.Query"
		operation = "SyntaxError"
		errorType = errors.ErrorTypeDatabase

	case strings.Contains(msgLower, "constraint"):
		component = "MySQL.Constraint"
		operation = "ConstraintViolation"
		errorType = errors.ErrorTypeConstraint
	}

	// Clean up the message
	msg = cleanupMySQLMessage(msg)

	// Route to unified error system
	errors.Get().LogError(errorType, component, operation, msg)
}

// cleanupMySQLMessage cleans up MySQL log messages
func cleanupMySQLMessage(msg string) string {
	// Remove file:line references
	if idx := strings.Index(msg, ".go:"); idx != -1 {
		if spaceIdx := strings.LastIndex(msg[:idx], " "); spaceIdx != -1 {
			msg = msg[:spaceIdx] + msg[idx+10:] // Skip past the line number
		}
	}

	// Remove common prefixes
	msg = strings.TrimPrefix(msg, "[mysql] ")
	msg = strings.TrimPrefix(msg, "mysql: ")

	return strings.TrimSpace(msg)
}

// ConfigureMySQLDriver sets up the MySQL driver with custom error logging
func ConfigureMySQLDriver() {
	// Set custom logger for MySQL driver
	mysql.SetLogger(&MySQLErrorLogger{})

	// Register custom driver with error handling
	mysql.RegisterDial("tcp", func(addr string) (net.Conn, error) {
		conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
		if err != nil {
			errors.Get().NetworkError("MySQL.Driver", "Dial", err)
			return nil, err
		}
		return conn, nil
	})

	// Enable debug logging if needed (controlled by environment variable)
	if os.Getenv("MYSQL_DEBUG") == "true" {
		log.Println("MySQL debug logging enabled")
		// The MySQL driver will now log all operations through our custom logger
	}
}

// MySQLConnector wraps the standard MySQL connector with error handling
type MySQLConnector struct {
	dsn string
}

// NewMySQLConnector creates a new MySQL connector with error handling
func NewMySQLConnector(dsn string) driver.Connector {
	return &MySQLConnector{dsn: dsn}
}

// Connect implements driver.Connector
func (mc *MySQLConnector) Connect(ctx context.Context) (driver.Conn, error) {
	cfg, err := mysql.ParseDSN(mc.dsn)
	if err != nil {
		errors.Get().DatabaseError("MySQL.Connector", "ParseDSN", err)
		return nil, err
	}

	// Set additional configuration for better error handling
	cfg.RejectReadOnly = true
	cfg.CheckConnLiveness = true
	cfg.InterpolateParams = true // Reduce round trips

	// Create connector with config
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		errors.Get().DatabaseError("MySQL.Connector", "NewConnector", err)
		return nil, err
	}

	// Connect with context
	conn, err := connector.Connect(ctx)
	if err != nil {
		errors.Get().NetworkError("MySQL.Connector", "Connect", err)
		return nil, err
	}

	return conn, nil
}

// Driver implements driver.Connector
func (mc *MySQLConnector) Driver() driver.Driver {
	return mysql.MySQLDriver{}
}

