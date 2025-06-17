package dashboard

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// DashboardRenderer handles efficient terminal rendering with double buffering
type DashboardRenderer struct {
	// Double buffering
	currentFrame  *bytes.Buffer
	previousFrame *bytes.Buffer
	frameMutex    sync.Mutex
	
	// Screen dimensions
	width  int
	height int
	
	// Component positions (for differential updates)
	headerLines    int
	eraStartLine   int
	perfStartLine  int
	feedStartLine  int
	errorStartLine int
	
	// Animation state
	animationTick  uint64
	lastRenderTime time.Time
	
	// Terminal state
	initialized bool
}

// NewDashboardRenderer creates a new renderer
func NewDashboardRenderer() *DashboardRenderer {
	return &DashboardRenderer{
		currentFrame:  &bytes.Buffer{},
		previousFrame: &bytes.Buffer{},
		width:         65,
		height:        40,
		headerLines:   5,
		eraStartLine:  6,
		perfStartLine: 15,
		feedStartLine: 22,
		errorStartLine: 28,
	}
}

// Initialize prepares the terminal
func (dr *DashboardRenderer) Initialize() {
	dr.frameMutex.Lock()
	defer dr.frameMutex.Unlock()
	
	if !dr.initialized {
		// Add a small delay to ensure previous output is visible
		time.Sleep(100 * time.Millisecond)
		
		// Clear screen completely, reset cursor, and hide it
		fmt.Print("\033[2J\033[3J\033[H\033[?25l")
		fmt.Fprint(os.Stdout, "\033[0m") // Reset all attributes
		
		dr.initialized = true
	}
}

// Cleanup restores terminal state
func (dr *DashboardRenderer) Cleanup() {
	// Show cursor and clear screen
	fmt.Print("\033[?25h\033[2J\033[H")
}

// RenderFrame renders a complete dashboard frame
func (dr *DashboardRenderer) RenderFrame(data *DashboardData) {
	dr.frameMutex.Lock()
	defer dr.frameMutex.Unlock()
	
	// Swap buffers
	dr.previousFrame, dr.currentFrame = dr.currentFrame, dr.previousFrame
	dr.currentFrame.Reset()
	
	// Increment animation tick
	dr.animationTick++
	
	// Build the new frame
	dr.renderHeader()
	dr.renderEraProgress(data)
	dr.renderPerformance(data)
	dr.renderActivityFeed(data)
	dr.renderErrorMonitor(data)
	
	// Output differential updates
	dr.outputDifferentialUpdates()
	
	dr.lastRenderTime = time.Now()
}

// renderHeader renders the static header
func (dr *DashboardRenderer) renderHeader() {
	// Move to top
	dr.currentFrame.WriteString("\033[1;1H")
	
	// Header
	dr.currentFrame.WriteString("\033[38;5;99m")
	dr.currentFrame.WriteString(strings.Repeat("=", dr.width))
	dr.currentFrame.WriteString("\033[0m\n")
	
	dr.currentFrame.WriteString("\033[38;5;99m")
	dr.currentFrame.WriteString(centerText("NECTAR INDEXER", dr.width))
	dr.currentFrame.WriteString("\033[0m\n")
	
	dr.currentFrame.WriteString("\033[38;5;226m")
	dr.currentFrame.WriteString(centerText("Cardano Blockchain Indexer", dr.width))
	dr.currentFrame.WriteString("\033[0m\n")
	
	dr.currentFrame.WriteString("\033[38;5;99m")
	dr.currentFrame.WriteString(strings.Repeat("=", dr.width))
	dr.currentFrame.WriteString("\033[0m\n\n")
}

// renderEraProgress renders era progress bars with animations
func (dr *DashboardRenderer) renderEraProgress(data *DashboardData) {
	// Move to era section
	dr.currentFrame.WriteString(fmt.Sprintf("\033[%d;1H", dr.eraStartLine))
	
	dr.currentFrame.WriteString("\033[38;5;226m| ERA PROGRESS\033[0m\n")
	dr.currentFrame.WriteString("\033[38;5;226m")
	dr.currentFrame.WriteString(strings.Repeat("-", 60))
	dr.currentFrame.WriteString("\033[0m\n")
	
	// Render each era
	dr.renderProgressBar("Byron", data.ByronProgress)
	dr.renderProgressBar("Shelley", data.ShelleyProgress)
	dr.renderProgressBar("Allegra", data.AllegraProgress)
	dr.renderProgressBar("Mary", data.MaryProgress)
	dr.renderProgressBar("Alonzo", data.AlonzoProgress)
	dr.renderProgressBar("Babbage", data.BabbageProgress)
	dr.renderProgressBar("Conway", data.ConwayProgress)
}

// renderProgressBar renders an animated progress bar
func (dr *DashboardRenderer) renderProgressBar(name string, progress float64) {
	barWidth := 50
	filled := int(progress * float64(barWidth) / 100.0)
	
	// Always show at least one animated dot if progress > 0
	if progress > 0 && filled == 0 {
		filled = 1
	}
	
	// Animation patterns
	dots9 := []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}
	sand := []string{"⠏", "⠛", "⠹", "⢸", "⣰", "⣤", "⣆", "⡇", "⠟", "⠻", "⠽", "⠾", "⠷"}
	
	// Build progress bar
	dr.currentFrame.WriteString("\033[38;5;99m")
	dr.currentFrame.WriteString(fmt.Sprintf("%-8s [", name))
	
	for i := 0; i < barWidth; i++ {
		if i < filled {
			// Animate filled portion
			if i%2 == 0 {
				// Use animation tick for smooth, consistent animation
				offset := int((dr.animationTick + uint64(i*2)) % uint64(len(dots9)))
				dr.currentFrame.WriteString("\033[38;5;226m" + dots9[offset] + "\033[38;5;99m")
			} else {
				offset := int((dr.animationTick + uint64(i*3)) % uint64(len(sand)))
				dr.currentFrame.WriteString("\033[38;5;226m" + sand[offset] + "\033[38;5;99m")
			}
		} else {
			dr.currentFrame.WriteString(".")
		}
	}
	
	dr.currentFrame.WriteString(fmt.Sprintf("] \033[38;5;226m%6.1f%%\033[0m\n", progress))
}

// renderPerformance renders performance metrics
func (dr *DashboardRenderer) renderPerformance(data *DashboardData) {
	// Move to performance section
	dr.currentFrame.WriteString(fmt.Sprintf("\033[%d;1H", dr.perfStartLine))
	
	dr.currentFrame.WriteString("\033[38;5;226m| PERFORMANCE\033[0m\n")
	dr.currentFrame.WriteString("\033[38;5;226m")
	dr.currentFrame.WriteString(strings.Repeat("-", 60))
	dr.currentFrame.WriteString("\033[0m\n")
	
	// Performance metrics
	dr.currentFrame.WriteString("\033[38;5;99m")
	dr.currentFrame.WriteString(fmt.Sprintf("Speed: \033[38;5;226m%8.0f blocks/sec\033[38;5;99m | RAM: \033[38;5;226m%6s\033[38;5;99m | CPU: \033[38;5;226m%6s\033[38;5;99m\n",
		data.CurrentRate, data.MemoryUsage, data.CPUUsage))
	
	dr.currentFrame.WriteString(fmt.Sprintf("Blocks: \033[38;5;226m%10d\033[38;5;99m | Runtime: \033[38;5;226m%8s\033[38;5;99m | Era: \033[38;5;226m%8s\033[38;5;99m\n",
		data.BlockCount, data.Runtime, data.CurrentEra))
	
	dr.currentFrame.WriteString(fmt.Sprintf("Slot: \033[38;5;226m%12d\033[38;5;99m | Peak: \033[38;5;226m%8.0f b/s\033[38;5;99m | Progress: \033[38;5;226m%5.1f%%\033[38;5;99m\033[0m\n",
		data.CurrentSlot, data.PeakRate, data.EraProgress))
	
	// Tip tracking
	if data.TipSlot > 0 {
		syncStatus := "SYNCING"
		syncColor := "\033[38;5;226m" // Yellow
		if data.TipDistance < 50 {
			syncStatus = "SYNCED"
			syncColor = "\033[38;5;82m" // Green
		}
		
		dr.currentFrame.WriteString(fmt.Sprintf("\033[38;5;99mTip: \033[38;5;226m%13d\033[38;5;99m | Behind: \033[38;5;226m%8d slots\033[38;5;99m | Status: %s%s\033[0m\n",
			data.TipSlot, data.TipDistance, syncColor, syncStatus))
	}
	
	// Parallel processing metrics
	dr.currentFrame.WriteString(fmt.Sprintf("\033[38;5;99mWorkers: \033[38;5;226m%d/%d active\033[38;5;99m | Queue: \033[38;5;226m%4d blocks\033[38;5;99m | Mode: \033[38;5;82mASYNC BLOCKS\033[0m\n",
		data.ActiveWorkers, data.TotalWorkers, data.QueueDepth))
	
	dr.currentFrame.WriteString("\n")
}

// renderActivityFeed renders the activity feed
func (dr *DashboardRenderer) renderActivityFeed(data *DashboardData) {
	// Move to activity section
	dr.currentFrame.WriteString(fmt.Sprintf("\033[%d;1H", dr.feedStartLine))
	
	dr.currentFrame.WriteString("\033[38;5;226m| ACTIVITY FEED\033[0m\n")
	dr.currentFrame.WriteString("\033[38;5;226m")
	dr.currentFrame.WriteString(strings.Repeat("-", 60))
	dr.currentFrame.WriteString("\033[0m\n")
	
	if len(data.Activities) == 0 {
		dr.currentFrame.WriteString("\033[38;5;99mStarting blockchain processing...\033[0m\n")
	} else {
		// Show last 4 activities
		start := 0
		if len(data.Activities) > 4 {
			start = len(data.Activities) - 4
		}
		
		for i := start; i < len(data.Activities); i++ {
			activity := data.Activities[i]
			dr.currentFrame.WriteString(fmt.Sprintf("\033[38;5;226m%s\033[38;5;99m %s %s\n",
				activity.Time, activity.Type, activity.Message))
		}
	}
	
	// Pad remaining lines
	activityCount := len(data.Activities)
	if activityCount > 4 {
		activityCount = 4
	}
	for i := activityCount; i < 4; i++ {
		dr.currentFrame.WriteString("\n")
	}
	dr.currentFrame.WriteString("\n")
}

// renderErrorMonitor renders the error monitor
func (dr *DashboardRenderer) renderErrorMonitor(data *DashboardData) {
	// Move to error section
	dr.currentFrame.WriteString(fmt.Sprintf("\033[%d;1H", dr.errorStartLine))
	
	dr.currentFrame.WriteString("\033[38;5;226m| ERROR MONITOR\033[0m\n")
	dr.currentFrame.WriteString("\033[38;5;226m")
	dr.currentFrame.WriteString(strings.Repeat("-", 60))
	dr.currentFrame.WriteString("\033[0m\n")
	
	if data.TotalErrors == 0 {
		dr.currentFrame.WriteString("\033[38;5;226m[OK]\033[38;5;99m All systems operational - No errors detected\033[0m\n\n")
	} else {
		dr.currentFrame.WriteString(fmt.Sprintf("\033[38;5;196mTotal Errors: %d\033[0m | Full log: ./errors.log\n", 
			data.TotalErrors))
		
		// Show recent errors
		for _, err := range data.RecentErrors {
			dr.currentFrame.WriteString(fmt.Sprintf("\033[38;5;196m%s [\033[38;5;226m%3dx\033[38;5;196m] %s\033[0m\n",
				err.Time, err.Count, err.Message))
		}
	}
}

// outputDifferentialUpdates outputs only changed lines
func (dr *DashboardRenderer) outputDifferentialUpdates() {
	// For now, output the entire frame
	// In a production system, we'd diff line by line
	fmt.Print(dr.currentFrame.String())
	os.Stdout.Sync() // Ensure output is flushed
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	padding := (width - len(text)) / 2
	if padding < 0 {
		padding = 0
	}
	return fmt.Sprintf("%s%s%s", strings.Repeat(" ", padding), text, strings.Repeat(" ", width-len(text)-padding))
}

// DashboardData holds all data needed to render the dashboard
type DashboardData struct {
	// Era progress
	ByronProgress   float64
	ShelleyProgress float64
	AllegraProgress float64
	MaryProgress    float64
	AlonzoProgress  float64
	BabbageProgress float64
	ConwayProgress  float64
	
	// Performance
	CurrentRate  float64
	PeakRate     float64
	BlockCount   int64
	CurrentSlot  uint64
	Runtime      string
	CurrentEra   string
	EraProgress  float64
	MemoryUsage  string
	CPUUsage     string
	
	// Tip tracking
	TipSlot     uint64
	TipDistance uint64
	
	// Parallel processing
	QueueDepth    int
	ActiveWorkers int32
	TotalWorkers  int
	
	// Activities
	Activities []ActivityData
	
	// Errors
	TotalErrors  int64
	RecentErrors []ErrorData
}

type ActivityData struct {
	Time    string
	Type    string
	Message string
}

type ErrorData struct {
	Time    string
	Count   int
	Message string
}