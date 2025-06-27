package dashboard

import (
	"fmt"
	"log"
	"time"
	
	"nectar/web"
)

// WebDashboardAdapter adapts the web server to the Dashboard interface
type WebDashboardAdapter struct {
	server *web.Server
}

// NewWebDashboardAdapter creates a new web dashboard adapter
func NewWebDashboardAdapter(port int) (*WebDashboardAdapter, error) {
	server, err := web.NewServer(port)
	if err != nil {
		return nil, fmt.Errorf("failed to create web server: %v", err)
	}
	
	return &WebDashboardAdapter{
		server: server,
	}, nil
}

// SetAuthConfig sets the authentication configuration
func (w *WebDashboardAdapter) SetAuthConfig(enabled bool, username, password, secret string) {
	if w.server != nil {
		w.server.SetAuthConfig(enabled, username, password, secret)
	}
}

// Start starts the web server
func (w *WebDashboardAdapter) Start() error {
	log.Println("[Dashboard] Starting web dashboard server...")
	err := w.server.Start()
	if err != nil {
		log.Printf("[Dashboard] Failed to start web server: %v", err)
		return err
	}
	log.Println("[Dashboard] Web dashboard server started")
	return nil
}

// Stop stops the web server
func (w *WebDashboardAdapter) Stop() {
	w.server.Stop()
}

// UpdateEraProgress updates era progress
func (w *WebDashboardAdapter) UpdateEraProgress(era string, progress float64) {
	w.server.UpdateEraProgress(era, progress)
}

// UpdatePerformance updates performance metrics
func (w *WebDashboardAdapter) UpdatePerformance(blocksPerSec string, currentBlocksPerSec, peakBlocksPerSec float64,
	totalBlocks int64, currentSlot, tipSlot uint64, memoryUsage, cpuUsage string,
	runtime time.Duration, currentEra string) {
	
	// Update web server with performance data
	// The web server expects: slot, tip uint64, blocksPerSec float64, totalBlocks int64, mem, cpu string
	w.server.UpdatePerformance(currentSlot, tipSlot, currentBlocksPerSec, totalBlocks, memoryUsage, cpuUsage)
}

// AddActivity adds an activity to the dashboard
func (w *WebDashboardAdapter) AddActivity(actType, message string) {
	// The web server expects a data map as the third parameter
	w.server.AddActivity(actType, message, nil)
}

// UpdateErrors updates error statistics
func (w *WebDashboardAdapter) UpdateErrors(totalErrors int, categorizedErrors []string) {
	// The web server doesn't have a direct UpdateErrors method
	// Errors are added individually via AddError
	// This is just for compatibility with the Dashboard interface
}

// AddError adds a new error to the dashboard
func (w *WebDashboardAdapter) AddError(errorType, component, message string) {
	w.server.AddError(errorType, component, message)
}