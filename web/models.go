package web

import (
	"fmt"
	"sync"
	"time"
)

// DashboardData holds all data for the web dashboard
type DashboardData struct {
	mu sync.RWMutex

	// Status
	Status      string    `json:"status"`
	LastUpdate  time.Time `json:"lastUpdate"`
	StartTime   time.Time `json:"startTime"`

	// Era Progress
	EraProgress map[string]float64 `json:"eraProgress"`
	CurrentEra  string             `json:"currentEra"`

	// Performance
	CurrentSlot      uint64    `json:"currentSlot"`
	TipSlot          uint64    `json:"tipSlot"`
	BlocksPerSec     float64   `json:"blocksPerSec"`
	PeakBlocksPerSec float64   `json:"peakBlocksPerSec"`
	TotalBlocks      int64     `json:"totalBlocks"`
	MemoryUsage      string    `json:"memoryUsage"`
	CPUUsage         string    `json:"cpuUsage"`
	SyncPercentage   float64   `json:"syncPercentage"`

	// Performance History (for charts)
	PerformanceHistory []PerformancePoint `json:"performanceHistory"`

	// Activities
	Activities []Activity `json:"activities"`

	// Errors
	TotalErrors       int64         `json:"totalErrors"`
	ErrorsByType      map[string]int64 `json:"errorsByType"`
	RecentErrors      []ErrorEntry  `json:"recentErrors"`
}

// PerformancePoint represents a point in the performance chart
type PerformancePoint struct {
	Timestamp    time.Time `json:"timestamp"`
	BlocksPerSec float64   `json:"blocksPerSec"`
	MemoryMB     float64   `json:"memoryMB"`
	CPUPercent   float64   `json:"cpuPercent"`
}

// Activity represents an activity feed entry
type Activity struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ErrorEntry represents an error in the dashboard
type ErrorEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Component string    `json:"component"`
	Message   string    `json:"message"`
	Count     int       `json:"count"`
}

// NewDashboardData creates a new dashboard data instance
func NewDashboardData() *DashboardData {
	eras := []string{"Byron", "Shelley", "Allegra", "Mary", "Alonzo", "Babbage", "Conway"}
	eraProgress := make(map[string]float64)
	for _, era := range eras {
		eraProgress[era] = 0
	}

	return &DashboardData{
		Status:             "initializing",
		StartTime:          time.Now(),
		LastUpdate:         time.Now(),
		EraProgress:        eraProgress,
		CurrentEra:         "Byron",
		Activities:         make([]Activity, 0),
		RecentErrors:       make([]ErrorEntry, 0),
		PerformanceHistory: make([]PerformancePoint, 0),
		ErrorsByType:       make(map[string]int64),
	}
}

// Update methods

func (d *DashboardData) UpdateStatus(status string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Status = status
	d.LastUpdate = time.Now()
}

func (d *DashboardData) UpdateEraProgress(era string, progress float64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.EraProgress[era] = progress
	d.LastUpdate = time.Now()
}

func (d *DashboardData) UpdatePerformance(slot, tip uint64, blocksPerSec float64, totalBlocks int64, mem, cpu string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	d.CurrentSlot = slot
	d.TipSlot = tip
	d.BlocksPerSec = blocksPerSec
	d.TotalBlocks = totalBlocks
	d.MemoryUsage = mem
	d.CPUUsage = cpu
	
	// Update peak
	if blocksPerSec > d.PeakBlocksPerSec {
		d.PeakBlocksPerSec = blocksPerSec
	}
	
	// Calculate sync percentage
	if tip > 0 {
		d.SyncPercentage = float64(slot) / float64(tip) * 100
	}
	
	// Determine current era based on slot
	d.CurrentEra = getEraFromSlot(slot)
	
	// Add to performance history
	d.addPerformancePoint(blocksPerSec, mem, cpu)
	
	d.LastUpdate = time.Now()
}

func (d *DashboardData) AddActivity(activityType, message string, data map[string]interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	activity := Activity{
		ID:        generateID(),
		Timestamp: time.Now(),
		Type:      activityType,
		Message:   message,
		Data:      data,
	}
	
	// Add to front of activities
	d.Activities = append([]Activity{activity}, d.Activities...)
	
	// Keep only last 50 activities
	if len(d.Activities) > 50 {
		d.Activities = d.Activities[:50]
	}
	
	d.LastUpdate = time.Now()
}

func (d *DashboardData) AddError(errorType, component, message string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Check if this error already exists
	for i, err := range d.RecentErrors {
		if err.Type == errorType && err.Component == component && err.Message == message {
			d.RecentErrors[i].Count++
			d.RecentErrors[i].Timestamp = time.Now()
			d.LastUpdate = time.Now()
			return
		}
	}
	
	// Add new error
	errorEntry := ErrorEntry{
		ID:        generateID(),
		Timestamp: time.Now(),
		Type:      errorType,
		Component: component,
		Message:   message,
		Count:     1,
	}
	
	d.RecentErrors = append([]ErrorEntry{errorEntry}, d.RecentErrors...)
	
	// Keep only last 20 errors
	if len(d.RecentErrors) > 20 {
		d.RecentErrors = d.RecentErrors[:20]
	}
	
	// Update error counts
	d.TotalErrors++
	d.ErrorsByType[errorType]++
	
	d.LastUpdate = time.Now()
}

func (d *DashboardData) GetSnapshot() DashboardData {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Create a copy of the data
	snapshot := *d
	
	// Deep copy maps and slices
	snapshot.EraProgress = make(map[string]float64)
	for k, v := range d.EraProgress {
		snapshot.EraProgress[k] = v
	}
	
	snapshot.ErrorsByType = make(map[string]int64)
	for k, v := range d.ErrorsByType {
		snapshot.ErrorsByType[k] = v
	}
	
	snapshot.Activities = make([]Activity, len(d.Activities))
	copy(snapshot.Activities, d.Activities)
	
	snapshot.RecentErrors = make([]ErrorEntry, len(d.RecentErrors))
	copy(snapshot.RecentErrors, d.RecentErrors)
	
	snapshot.PerformanceHistory = make([]PerformancePoint, len(d.PerformanceHistory))
	copy(snapshot.PerformanceHistory, d.PerformanceHistory)
	
	return snapshot
}

// Helper functions

func (d *DashboardData) addPerformancePoint(blocksPerSec float64, mem, cpu string) {
	// Parse memory and CPU values
	memMB := parseMemoryMB(mem)
	cpuPercent := parseCPUPercent(cpu)
	
	point := PerformancePoint{
		Timestamp:    time.Now(),
		BlocksPerSec: blocksPerSec,
		MemoryMB:     memMB,
		CPUPercent:   cpuPercent,
	}
	
	d.PerformanceHistory = append(d.PerformanceHistory, point)
	
	// Keep only last 300 points (5 minutes at 1 update per second)
	if len(d.PerformanceHistory) > 300 {
		d.PerformanceHistory = d.PerformanceHistory[1:]
	}
}

func getEraFromSlot(slot uint64) string {
	if slot >= 133660800 {
		return "Conway"
	} else if slot >= 72316797 {
		return "Babbage"
	} else if slot >= 39916797 {
		return "Alonzo"
	} else if slot >= 23068794 {
		return "Mary"
	} else if slot >= 16588738 {
		return "Allegra"
	} else if slot >= 4492800 {
		return "Shelley"
	}
	return "Byron"
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func parseMemoryMB(mem string) float64 {
	// Parse memory string like "1.2GB" or "512MB"
	// Simplified for now
	return 0
}

func parseCPUPercent(cpu string) float64 {
	// Parse CPU string like "45.2%"
	// Simplified for now
	return 0
}