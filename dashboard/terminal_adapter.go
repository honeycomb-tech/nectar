package dashboard

import (
	"log"
	"time"
	
	"nectar/terminal"
)

// TerminalDashboardAdapter adapts the terminal progress indicator to the Dashboard interface
type TerminalDashboardAdapter struct {
	progress *terminal.ProgressIndicator
}

// NewTerminalDashboardAdapter creates a new terminal dashboard adapter
func NewTerminalDashboardAdapter(detailed bool) *TerminalDashboardAdapter {
	return &TerminalDashboardAdapter{
		progress: terminal.NewProgressIndicator(detailed),
	}
}

// Start starts the terminal progress display
func (t *TerminalDashboardAdapter) Start() error {
	log.Println("[Dashboard] Starting terminal progress indicator...")
	err := t.progress.Start()
	if err != nil {
		log.Printf("[Dashboard] Failed to start terminal progress: %v", err)
		return err
	}
	log.Println("[Dashboard] Terminal progress indicator started")
	return nil
}

// Stop stops the terminal progress display
func (t *TerminalDashboardAdapter) Stop() {
	t.progress.Stop()
}

// UpdateEraProgress updates era progress
func (t *TerminalDashboardAdapter) UpdateEraProgress(era string, progress float64) {
	t.progress.UpdateEraProgress(era, progress)
}

// UpdatePerformance updates performance metrics
func (t *TerminalDashboardAdapter) UpdatePerformance(blocksPerSec string, currentBlocksPerSec, peakBlocksPerSec float64,
	totalBlocks int64, currentSlot, tipSlot uint64, memoryUsage, cpuUsage string,
	runtime time.Duration, currentEra string) {
	
	// Update terminal with current values
	t.progress.UpdateSlot(currentSlot, tipSlot)
	t.progress.UpdatePerformance(currentBlocksPerSec, totalBlocks, memoryUsage, cpuUsage)
	t.progress.UpdateEra(currentEra)
}

// AddActivity adds an activity (terminal doesn't show activities)
func (t *TerminalDashboardAdapter) AddActivity(actType, message string) {
	// Terminal progress doesn't display activities
	// Could optionally flash the message briefly
}

// UpdateErrors updates error count (terminal doesn't show detailed errors)
func (t *TerminalDashboardAdapter) UpdateErrors(totalErrors int, categorizedErrors []string) {
	// Terminal progress doesn't display errors
	// Could add error count to status line if desired
}

// AddError adds a new error (terminal doesn't show errors to keep display clean)
func (t *TerminalDashboardAdapter) AddError(errorType, component, message string) {
	// Terminal doesn't display errors to keep the interface clean
}