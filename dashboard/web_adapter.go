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
	
	// Update all performance metrics
	w.server.UpdatePerformance(currentSlot, tipSlot, currentBlocksPerSec, totalBlocks, memoryUsage, cpuUsage)
	
	// Update era progress based on slot
	w.updateAllEraProgress(currentSlot)
}

// AddActivity adds an activity to the feed
func (w *WebDashboardAdapter) AddActivity(actType, message string) {
	w.server.AddActivity(actType, message, nil)
}

// UpdateErrors updates error information
func (w *WebDashboardAdapter) UpdateErrors(totalErrors int, categorizedErrors []string) {
	// This is called periodically with the full error list
	// The web server handles this internally through its own error tracking
}

// AddError adds a new error with proper type information
func (w *WebDashboardAdapter) AddError(errorType, component, message string) {
	// Forward the error with its proper type to the web server
	w.server.AddError(errorType, component, message)
}

// updateAllEraProgress calculates and updates progress for all eras based on current slot
func (w *WebDashboardAdapter) updateAllEraProgress(currentSlot uint64) {
	// Byron
	byronProgress := 0.0
	if currentSlot >= 4492800 {
		byronProgress = 100.0
	} else {
		byronProgress = (float64(currentSlot) / 4492799.0) * 100
	}
	w.server.UpdateEraProgress("Byron", byronProgress)

	// Shelley
	shelleyProgress := 0.0
	if currentSlot >= 16588738 {
		shelleyProgress = 100.0
	} else if currentSlot >= 4492800 {
		shelleyProgress = ((float64(currentSlot) - 4492800) / (16588738 - 4492800)) * 100
	}
	w.server.UpdateEraProgress("Shelley", shelleyProgress)

	// Allegra
	allegraProgress := 0.0
	if currentSlot >= 23068794 {
		allegraProgress = 100.0
	} else if currentSlot >= 16588738 {
		allegraProgress = ((float64(currentSlot) - 16588738) / (23068794 - 16588738)) * 100
	}
	w.server.UpdateEraProgress("Allegra", allegraProgress)

	// Mary
	maryProgress := 0.0
	if currentSlot >= 39916797 {
		maryProgress = 100.0
	} else if currentSlot >= 23068794 {
		maryProgress = ((float64(currentSlot) - 23068794) / (39916797 - 23068794)) * 100
	}
	w.server.UpdateEraProgress("Mary", maryProgress)

	// Alonzo
	alonzoProgress := 0.0
	if currentSlot >= 72316797 {
		alonzoProgress = 100.0
	} else if currentSlot >= 39916797 {
		alonzoProgress = ((float64(currentSlot) - 39916797) / (72316797 - 39916797)) * 100
	}
	w.server.UpdateEraProgress("Alonzo", alonzoProgress)

	// Babbage
	babbageProgress := 0.0
	if currentSlot >= 133660800 {
		babbageProgress = 100.0
	} else if currentSlot >= 72316797 {
		babbageProgress = ((float64(currentSlot) - 72316797) / (133660800 - 72316797)) * 100
	}
	w.server.UpdateEraProgress("Babbage", babbageProgress)

	// Conway
	conwayProgress := 0.0
	if currentSlot >= 133660800 {
		conwayProgress = 5.0 // Early stage
	}
	w.server.UpdateEraProgress("Conway", conwayProgress)
}