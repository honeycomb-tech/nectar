package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
)

// ProgressIndicator represents a simple terminal progress display
type ProgressIndicator struct {
	spinner  *spinner.Spinner
	mu       sync.RWMutex
	detailed bool
	writer   io.Writer

	// Current state
	currentEra      string
	currentSlot     uint64
	tipSlot         uint64
	blocksPerSec    float64
	totalBlocks     int64
	memoryUsage     string
	cpuUsage        string
	eraProgress     map[string]float64
	isRunning       bool
	lastUpdate      time.Time
	updateInterval  time.Duration
}

// NewProgressIndicator creates a new terminal progress indicator
func NewProgressIndicator(detailed bool) *ProgressIndicator {
	// Use a simple spinner that works well over SSH
	s := spinner.New(spinner.CharSets[11], 100*time.Millisecond)
	
	// Check dashboard type - use stderr for "both" mode to avoid log conflicts
	writer := io.Writer(os.Stdout)
	if os.Getenv("DASHBOARD_TYPE") == "both" {
		writer = os.Stderr
	}
	
	s.Writer = writer
	s.HideCursor = true
	
	return &ProgressIndicator{
		spinner:        s,
		detailed:       detailed,
		writer:         writer,
		eraProgress:    make(map[string]float64),
		updateInterval: 500 * time.Millisecond, // Slower updates
	}
}

// Start begins the progress display
func (p *ProgressIndicator) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isRunning {
		return nil
	}

	p.isRunning = true
	p.lastUpdate = time.Now()

	// Start the spinner
	p.spinner.Start()

	// Start update loop
	go p.updateLoop()

	return nil
}

// Stop halts the progress display
func (p *ProgressIndicator) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.isRunning {
		return
	}

	p.isRunning = false
	p.spinner.Stop()
	
	// Clear the line
	fmt.Fprint(p.writer, "\r\033[K")
}

// updateLoop handles periodic updates
func (p *ProgressIndicator) updateLoop() {
	ticker := time.NewTicker(p.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.RLock()
			if !p.isRunning {
				p.mu.RUnlock()
				return
			}
			p.mu.RUnlock()
			
			p.render()
		}
	}
}

// render updates the display
func (p *ProgressIndicator) render() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.detailed {
		// Clear current line for detailed mode
		fmt.Fprint(p.writer, "\r\033[K")
		p.renderDetailed()
	} else {
		// For simple mode, just update the spinner suffix
		p.renderSimple()
	}
}

// renderSimple shows a single line progress
func (p *ProgressIndicator) renderSimple() {
	syncPercent := float64(0)
	if p.tipSlot > 0 {
		syncPercent = float64(p.currentSlot) / float64(p.tipSlot) * 100
	}

	// Update spinner suffix with current stats
	p.spinner.Suffix = fmt.Sprintf(" Nectar | Era: %s | Slot: %d | Sync: %.1f%% | Speed: %.1f b/s | Blocks: %d",
		p.currentEra,
		p.currentSlot,
		syncPercent,
		p.blocksPerSec,
		p.totalBlocks,
	)
}

// renderDetailed shows multiple lines with progress bars
func (p *ProgressIndicator) renderDetailed() {
	// Move cursor up to overwrite previous output
	lines := 10 // Number of lines we'll write
	if p.lastUpdate.Add(p.updateInterval * 2).Before(time.Now()) {
		// First render or after pause
		fmt.Fprintf(p.writer, "\033[%dA", lines)
	}

	// Header
	fmt.Fprintln(p.writer, "\n╔════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(p.writer, "║                    NECTAR INDEXER STATUS                   ║")
	fmt.Fprintln(p.writer, "╚════════════════════════════════════════════════════════════╝")

	// Era Progress
	fmt.Fprintln(p.writer, "\nEra Progress:")
	eras := []string{"Byron", "Shelley", "Allegra", "Mary", "Alonzo", "Babbage", "Conway"}
	for _, era := range eras {
		progress := p.eraProgress[era]
		bar := p.makeProgressBar(progress, 30)
		status := ""
		if era == p.currentEra {
			status = " ← current"
		}
		fmt.Fprintf(p.writer, "  %-8s %s %.1f%%%s\n", era, bar, progress, status)
	}

	// Performance
	syncPercent := float64(0)
	if p.tipSlot > 0 {
		syncPercent = float64(p.currentSlot) / float64(p.tipSlot) * 100
	}

	fmt.Fprintf(p.writer, "\nPerformance: %.1f blocks/sec | RAM: %s | CPU: %s\n",
		p.blocksPerSec, p.memoryUsage, p.cpuUsage)
	fmt.Fprintf(p.writer, "Sync Status: %.1f%% | Slot: %d / %d | Blocks: %d\n",
		syncPercent, p.currentSlot, p.tipSlot, p.totalBlocks)
}

// makeProgressBar creates an ASCII progress bar
func (p *ProgressIndicator) makeProgressBar(percent float64, width int) string {
	filled := int(percent * float64(width) / 100)
	if filled > width {
		filled = width
	}
	
	bar := "["
	bar += strings.Repeat("█", filled)
	bar += strings.Repeat("░", width-filled)
	bar += "]"
	
	return bar
}

// Update methods for external data

func (p *ProgressIndicator) UpdateEra(era string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentEra = era
}

func (p *ProgressIndicator) UpdateSlot(current, tip uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentSlot = current
	p.tipSlot = tip
}

func (p *ProgressIndicator) UpdatePerformance(blocksPerSec float64, totalBlocks int64, memUsage, cpuUsage string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocksPerSec = blocksPerSec
	p.totalBlocks = totalBlocks
	p.memoryUsage = memUsage
	p.cpuUsage = cpuUsage
}

func (p *ProgressIndicator) UpdateEraProgress(era string, progress float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.eraProgress[era] = progress
}

// SetDetailed toggles detailed view
func (p *ProgressIndicator) SetDetailed(detailed bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.detailed = detailed
}