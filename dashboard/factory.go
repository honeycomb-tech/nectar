package dashboard

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

// DashboardType represents the type of dashboard to create
type DashboardType string

const (
	DashboardTypeTerminal DashboardType = "terminal"
	DashboardTypeWeb      DashboardType = "web"
	DashboardTypeBoth     DashboardType = "both"
	DashboardTypeNone     DashboardType = "none"
)

// DashboardConfig holds configuration for creating dashboards
type DashboardConfig struct {
	Type        DashboardType
	WebPort     int
	DetailedLog bool
}

// GetDashboardConfig creates configuration from environment variables
func GetDashboardConfig() *DashboardConfig {
	config := &DashboardConfig{
		Type:        DashboardTypeBoth, // Default to both
		WebPort:     8080,              // Default port
		DetailedLog: false,
	}

	log.Printf("[Dashboard] Getting config - DASHBOARD_TYPE=%s, WEB_PORT=%s", 
		os.Getenv("DASHBOARD_TYPE"), os.Getenv("WEB_PORT"))

	// Check DASHBOARD_TYPE env var
	if dashType := os.Getenv("DASHBOARD_TYPE"); dashType != "" {
		switch dashType {
		case "terminal":
			config.Type = DashboardTypeTerminal
		case "web":
			config.Type = DashboardTypeWeb
		case "both":
			config.Type = DashboardTypeBoth
		case "none":
			config.Type = DashboardTypeNone
		default:
			log.Printf("[Dashboard] Unknown dashboard type '%s', using both", dashType)
		}
	}

	// Check WEB_PORT env var
	if portStr := os.Getenv("WEB_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			config.WebPort = port
		} else {
			log.Printf("[Dashboard] Invalid WEB_PORT '%s', using default 8080", portStr)
		}
	}

	// Check DETAILED_LOG env var
	if detailed := os.Getenv("DETAILED_LOG"); detailed == "true" {
		config.DetailedLog = true
	}

	return config
}

// CreateDashboard creates a dashboard based on the configuration
func CreateDashboard(config *DashboardConfig) (Dashboard, error) {
	switch config.Type {
	case DashboardTypeTerminal:
		log.Println("[Dashboard] Creating terminal-only dashboard")
		return NewTerminalDashboardAdapter(config.DetailedLog), nil

	case DashboardTypeWeb:
		log.Printf("[Dashboard] Creating web-only dashboard on port %d", config.WebPort)
		return NewWebDashboardAdapter(config.WebPort)

	case DashboardTypeBoth:
		log.Printf("[Dashboard] Creating both terminal and web dashboards (web port: %d)", config.WebPort)
		
		// Create both dashboards
		terminal := NewTerminalDashboardAdapter(config.DetailedLog)
		web, err := NewWebDashboardAdapter(config.WebPort)
		if err != nil {
			// If web fails, fall back to terminal only
			log.Printf("[Dashboard] Failed to create web dashboard: %v. Using terminal only.", err)
			return terminal, nil
		}
		
		// Return multi-dashboard
		return NewMultiDashboard(terminal, web), nil

	case DashboardTypeNone:
		log.Println("[Dashboard] No dashboard requested")
		return &NullDashboard{}, nil

	default:
		return nil, fmt.Errorf("unknown dashboard type: %s", config.Type)
	}
}

// CreateDefaultDashboard creates a dashboard with default settings
func CreateDefaultDashboard() (Dashboard, error) {
	config := GetDashboardConfig()
	return CreateDashboard(config)
}

// NullDashboard is a no-op dashboard implementation
type NullDashboard struct{}

func (n *NullDashboard) Start() error { return nil }
func (n *NullDashboard) Stop() {}
func (n *NullDashboard) UpdateEraProgress(era string, progress float64) {}
func (n *NullDashboard) UpdatePerformance(blocksPerSec string, currentBlocksPerSec, peakBlocksPerSec float64,
	totalBlocks int64, currentSlot, tipSlot uint64, memoryUsage, cpuUsage string,
	runtime time.Duration, currentEra string) {}
func (n *NullDashboard) AddActivity(actType, message string) {}
func (n *NullDashboard) UpdateErrors(totalErrors int, categorizedErrors []string) {}
func (n *NullDashboard) AddError(errorType, component, message string) {}