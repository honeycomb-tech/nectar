package dashboard

import (
	"log"
	"time"
)

// Dashboard defines the common interface for all dashboard implementations
type Dashboard interface {
	// Start initializes and starts the dashboard
	Start() error
	
	// Stop gracefully shuts down the dashboard
	Stop()
	
	// Update methods
	UpdateEraProgress(era string, progress float64)
	UpdatePerformance(blocksPerSec string, currentBlocksPerSec, peakBlocksPerSec float64,
		totalBlocks int64, currentSlot, tipSlot uint64, memoryUsage, cpuUsage string,
		runtime time.Duration, currentEra string)
	AddActivity(actType, message string)
	UpdateErrors(totalErrors int, categorizedErrors []string)
	AddError(errorType, component, message string)
}

// MultiDashboard allows running multiple dashboards simultaneously
type MultiDashboard struct {
	dashboards []Dashboard
}

// NewMultiDashboard creates a new multi-dashboard wrapper
func NewMultiDashboard(dashboards ...Dashboard) *MultiDashboard {
	return &MultiDashboard{
		dashboards: dashboards,
	}
}

// Start starts all dashboards
func (m *MultiDashboard) Start() error {
	for _, d := range m.dashboards {
		if err := d.Start(); err != nil {
			// Log error but continue with other dashboards
			log.Printf("[Dashboard] Failed to start dashboard: %v", err)
		}
	}
	return nil
}

// Stop stops all dashboards
func (m *MultiDashboard) Stop() {
	for _, d := range m.dashboards {
		d.Stop()
	}
}

// UpdateEraProgress updates all dashboards
func (m *MultiDashboard) UpdateEraProgress(era string, progress float64) {
	for _, d := range m.dashboards {
		d.UpdateEraProgress(era, progress)
	}
}

// UpdatePerformance updates all dashboards
func (m *MultiDashboard) UpdatePerformance(blocksPerSec string, currentBlocksPerSec, peakBlocksPerSec float64,
	totalBlocks int64, currentSlot, tipSlot uint64, memoryUsage, cpuUsage string,
	runtime time.Duration, currentEra string) {
	for _, d := range m.dashboards {
		d.UpdatePerformance(blocksPerSec, currentBlocksPerSec, peakBlocksPerSec,
			totalBlocks, currentSlot, tipSlot, memoryUsage, cpuUsage, runtime, currentEra)
	}
}

// AddActivity adds activity to all dashboards
func (m *MultiDashboard) AddActivity(actType, message string) {
	for _, d := range m.dashboards {
		d.AddActivity(actType, message)
	}
}

// UpdateErrors updates errors in all dashboards
func (m *MultiDashboard) UpdateErrors(totalErrors int, categorizedErrors []string) {
	for _, d := range m.dashboards {
		d.UpdateErrors(totalErrors, categorizedErrors)
	}
}

// AddError adds a new error to all dashboards
func (m *MultiDashboard) AddError(errorType, component, message string) {
	for _, d := range m.dashboards {
		d.AddError(errorType, component, message)
	}
}