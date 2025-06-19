package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// DashboardHandlers contains all HTTP handlers
type DashboardHandlers struct {
	data *DashboardData
}

// NewDashboardHandlers creates new handlers
func NewDashboardHandlers(data *DashboardData) *DashboardHandlers {
	return &DashboardHandlers{
		data: data,
	}
}

// HandleIndex serves the main dashboard page
func (h *DashboardHandlers) HandleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Nectar Dashboard",
	})
}

// HandleAPIStatus returns current status
func (h *DashboardHandlers) HandleAPIStatus(c *gin.Context) {
	snapshot := h.data.GetSnapshot()
	
	c.JSON(http.StatusOK, gin.H{
		"status":         snapshot.Status,
		"lastUpdate":     snapshot.LastUpdate,
		"startTime":      snapshot.StartTime,
		"uptime":         time.Since(snapshot.StartTime).String(),
		"currentEra":     snapshot.CurrentEra,
		"currentSlot":    snapshot.CurrentSlot,
		"tipSlot":        snapshot.TipSlot,
		"syncPercentage": snapshot.SyncPercentage,
		"blocksPerSec":   snapshot.BlocksPerSec,
		"totalBlocks":    snapshot.TotalBlocks,
	})
}

// HandleAPIEras returns era progress data
func (h *DashboardHandlers) HandleAPIEras(c *gin.Context) {
	snapshot := h.data.GetSnapshot()
	
	eras := []map[string]interface{}{
		{"name": "Byron", "progress": snapshot.EraProgress["Byron"], "current": snapshot.CurrentEra == "Byron"},
		{"name": "Shelley", "progress": snapshot.EraProgress["Shelley"], "current": snapshot.CurrentEra == "Shelley"},
		{"name": "Allegra", "progress": snapshot.EraProgress["Allegra"], "current": snapshot.CurrentEra == "Allegra"},
		{"name": "Mary", "progress": snapshot.EraProgress["Mary"], "current": snapshot.CurrentEra == "Mary"},
		{"name": "Alonzo", "progress": snapshot.EraProgress["Alonzo"], "current": snapshot.CurrentEra == "Alonzo"},
		{"name": "Babbage", "progress": snapshot.EraProgress["Babbage"], "current": snapshot.CurrentEra == "Babbage"},
		{"name": "Conway", "progress": snapshot.EraProgress["Conway"], "current": snapshot.CurrentEra == "Conway"},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"eras":       eras,
		"currentEra": snapshot.CurrentEra,
	})
}

// HandleAPIPerformance returns performance metrics
func (h *DashboardHandlers) HandleAPIPerformance(c *gin.Context) {
	snapshot := h.data.GetSnapshot()
	
	// For HTMX partial update
	if c.GetHeader("HX-Request") == "true" {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(
			`<p>Blocks/sec: <span class="font-mono">%.1f</span></p>
<p>Total Blocks: <span class="font-mono">%d</span></p>
<p>Memory: <span class="font-mono">%s</span></p>
<p>CPU: <span class="font-mono">%s</span></p>`, 
			snapshot.BlocksPerSec, snapshot.TotalBlocks, snapshot.MemoryUsage, snapshot.CPUUsage)))
		return
	}
	
	// For API calls
	c.JSON(http.StatusOK, gin.H{
		"current": gin.H{
			"blocksPerSec": snapshot.BlocksPerSec,
			"memoryUsage":  snapshot.MemoryUsage,
			"cpuUsage":     snapshot.CPUUsage,
			"totalBlocks":  snapshot.TotalBlocks,
		},
		"peak": gin.H{
			"blocksPerSec": snapshot.PeakBlocksPerSec,
		},
		"history": snapshot.PerformanceHistory,
	})
}

// HandleAPIActivities returns recent activities
func (h *DashboardHandlers) HandleAPIActivities(c *gin.Context) {
	snapshot := h.data.GetSnapshot()
	
	// Limit to last 20 activities
	activities := snapshot.Activities
	if len(activities) > 20 {
		activities = activities[:20]
	}
	
	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
		"count":      len(activities),
	})
}

// HandleAPIErrors returns error statistics
func (h *DashboardHandlers) HandleAPIErrors(c *gin.Context) {
	snapshot := h.data.GetSnapshot()
	
	c.JSON(http.StatusOK, gin.H{
		"totalErrors":  snapshot.TotalErrors,
		"errorsByType": snapshot.ErrorsByType,
		"recentErrors": snapshot.RecentErrors,
	})
}

// HTMX Partial Handlers for efficient updates

// HandlePartialStatus returns status partial
func (h *DashboardHandlers) HandlePartialStatus(c *gin.Context) {
	snapshot := h.data.GetSnapshot()
	
	status := "🔄 Syncing"
	if snapshot.SyncPercentage >= 99.9 {
		status = "✅ Synced"
	}
	
	data := gin.H{
		"Status":         status,
		"CurrentSlot":    snapshot.CurrentSlot,
		"TipSlot":        snapshot.TipSlot,
		"SyncPercentage": snapshot.SyncPercentage,
		"BlocksPerSec":   snapshot.BlocksPerSec,
		"CurrentEra":     snapshot.CurrentEra,
	}
	
	c.HTML(http.StatusOK, "partials/status.html", data)
}

// HandlePartialEras returns era progress partial
func (h *DashboardHandlers) HandlePartialEras(c *gin.Context) {
	snapshot := h.data.GetSnapshot()
	
	c.HTML(http.StatusOK, "partials/eras.html", gin.H{
		"EraProgress": snapshot.EraProgress,
		"CurrentEra":  snapshot.CurrentEra,
	})
}

// HandlePartialActivities returns activities partial
func (h *DashboardHandlers) HandlePartialActivities(c *gin.Context) {
	snapshot := h.data.GetSnapshot()
	
	activities := snapshot.Activities
	if len(activities) > 10 {
		activities = activities[:10]
	}
	
	c.HTML(http.StatusOK, "partials/activities.html", gin.H{
		"Activities": activities,
	})
}

// HandlePartialErrors returns errors partial
func (h *DashboardHandlers) HandlePartialErrors(c *gin.Context) {
	snapshot := h.data.GetSnapshot()
	
	recentErrors := snapshot.RecentErrors
	if len(recentErrors) > 5 {
		recentErrors = recentErrors[:5]
	}
	
	c.HTML(http.StatusOK, "partials/errors.html", gin.H{
		"TotalErrors":  snapshot.TotalErrors,
		"RecentErrors": recentErrors,
	})
}