package web

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"nectar/auth"
)

//go:embed templates/* templates/partials/*
var templatesFS embed.FS

// Server represents the web dashboard server
type Server struct {
	router       *gin.Engine
	server       *http.Server
	data         *DashboardData
	wsHub        *WebSocketHub
	handlers     *DashboardHandlers
	mu           sync.RWMutex
	isRunning    bool
	port         int
	updateTicker *time.Ticker
	authConfig   *auth.Config
}

// NewServer creates a new web dashboard server
func NewServer(port int) (*Server, error) {
	// Set Gin to release mode for production
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	
	// Initialize auth configuration - will be set later via SetAuthConfig
	authConfig := auth.GetDefaultConfig()
	
	// Initialize JWT secret
	if err := auth.InitJWTSecret(authConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize JWT secret: %v", err)
	}
	
	// Add simple request logging
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		
		// Skip logging for WebSocket and static assets
		if c.Request.URL.Path != "/ws" && 
		   c.Request.URL.Path != "/api/status" &&
		   !strings.Contains(c.Writer.Header().Get("Content-Type"), "text/html") {
			log.Printf("[Web] %s %s %d %s",
				c.Request.Method,
				c.Request.URL.Path,
				c.Writer.Status(),
				time.Since(start))
		}
	})

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"} // In production, specify your domain
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept"}
	router.Use(cors.New(config))

	// Initialize components
	data := NewDashboardData()
	wsHub := NewWebSocketHub()
	handlers := NewDashboardHandlers(data)

	server := &Server{
		router:     router,
		data:       data,
		wsHub:      wsHub,
		handlers:   handlers,
		port:       port,
		authConfig: authConfig,
	}

	// Load templates
	if err := server.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %v", err)
	}

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// loadTemplates loads HTML templates
func (s *Server) loadTemplates() error {
	// Get the executable directory to find templates
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	
	// Try multiple paths for templates
	templatePaths := []string{
		"web/templates",                           // Relative to current directory
		filepath.Join(execDir, "web/templates"),   // Relative to executable
		"/app/web/templates",                      // Docker container path
	}
	
	var templateDir string
	for _, path := range templatePaths {
		if _, err := os.Stat(path); err == nil {
			templateDir = path
			break
		}
	}
	
	if templateDir != "" {
		// Load from filesystem
		log.Printf("[Web] Loading templates from: %s", templateDir)
		
		// Create a new template with custom functions
		tmpl := template.New("").Funcs(template.FuncMap{
			"upper": strings.ToUpper,
		})
		
		// Load index template
		indexPath := filepath.Join(templateDir, "index.html")
		tmpl = template.Must(tmpl.New("index.html").ParseFiles(indexPath))
		
		// Load login template
		loginPath := filepath.Join(templateDir, "login.html")
		if _, err := os.Stat(loginPath); err == nil {
			tmpl = template.Must(tmpl.New("login.html").ParseFiles(loginPath))
		}
		
		// Load partial templates with their proper names
		partialFiles := []struct {
			name string
			path string
		}{
			{"partials/status.html", filepath.Join(templateDir, "partials", "status.html")},
			{"partials/eras.html", filepath.Join(templateDir, "partials", "eras.html")},
			{"partials/activities.html", filepath.Join(templateDir, "partials", "activities.html")},
			{"partials/errors.html", filepath.Join(templateDir, "partials", "errors.html")},
		}
		
		for _, pf := range partialFiles {
			content, err := os.ReadFile(pf.path)
			if err != nil {
				return fmt.Errorf("failed to read template %s: %w", pf.path, err)
			}
			tmpl = template.Must(tmpl.New(pf.name).Funcs(template.FuncMap{
				"upper": strings.ToUpper,
			}).Parse(string(content)))
		}
		
		s.router.SetHTMLTemplate(tmpl)
		log.Printf("[Web] Loaded %d templates from filesystem", len(partialFiles)+1)
	} else {
		// Use embedded templates
		log.Println("[Web] Using embedded templates")
		tmpl := template.Must(template.New("").Funcs(template.FuncMap{
			"upper": strings.ToUpper,
		}).ParseFS(templatesFS, "templates/*", "templates/partials/*"))
		s.router.SetHTMLTemplate(tmpl)
	}
	
	return nil
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Static files
	s.router.Static("/static", "./static")
	
	// Login page (no auth required)
	s.router.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	
	// Login/logout API endpoints (no auth required)
	s.router.POST("/api/login", auth.LoginHandler(s.authConfig))
	s.router.POST("/api/logout", auth.LogoutHandler(s.authConfig))
	
	// Apply auth middleware to all other routes
	authGroup := s.router.Group("/")
	authGroup.Use(auth.BasicAuth(s.authConfig))
	
	// Main page
	authGroup.GET("/", s.handlers.HandleIndex)
	
	// API endpoints
	api := authGroup.Group("/api")
	{
		api.GET("/status", s.handlers.HandleAPIStatus)
		api.GET("/eras", s.handlers.HandleAPIEras)
		api.GET("/performance", s.handlers.HandleAPIPerformance)
		api.GET("/activities", s.handlers.HandleAPIActivities)
		api.GET("/errors", s.handlers.HandleAPIErrors)
		api.GET("/memory", s.handlers.HandleAPIMemory)
	}
	
	// HTMX partials
	partials := authGroup.Group("/partials")
	{
		partials.GET("/status", s.handlers.HandlePartialStatus)
		partials.GET("/eras", s.handlers.HandlePartialEras)
		partials.GET("/activities", s.handlers.HandlePartialActivities)
		partials.GET("/errors", s.handlers.HandlePartialErrors)
	}
	
	// WebSocket endpoint (requires auth)
	authGroup.GET("/ws", HandleWebSocket(s.wsHub))
	
	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	
	// Favicon handler to prevent 404 errors
	s.router.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(204) // No content
	})
}

// Start starts the web server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("server already running")
	}

	log.Printf("[Web] Starting web dashboard server on port %d...", s.port)

	// Start WebSocket hub
	log.Println("[Web] Starting WebSocket hub...")
	go s.wsHub.Run()

	// Start update broadcaster
	log.Println("[Web] Starting update broadcaster...")
	s.startUpdateBroadcaster()

	// Create HTTP server
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("[Web] Dashboard server starting on http://0.0.0.0:%d", s.port)
		log.Printf("[Web] Dashboard accessible at http://<VM-IP>:%d", s.port)
		
		// Try to bind and listen
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Web] Server error: %v", err)
		}
	}()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)
	
	// Verify server is listening
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", s.port), 2*time.Second)
	if err != nil {
		log.Printf("[Web] WARNING: Server may not be accessible on port %d: %v", s.port, err)
	} else {
		conn.Close()
		log.Printf("[Web] Server verified listening on port %d", s.port)
	}

	s.isRunning = true
	return nil
}

// Stop gracefully stops the web server
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	// Stop update broadcaster
	if s.updateTicker != nil {
		s.updateTicker.Stop()
	}

	// Shutdown server with timeout
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("[Web] Server shutdown error: %v", err)
		}
	}

	s.isRunning = false
	log.Println("[Web] Dashboard server stopped")
}

// startUpdateBroadcaster starts periodic WebSocket updates
func (s *Server) startUpdateBroadcaster() {
	s.updateTicker = time.NewTicker(1 * time.Second)
	
	go func() {
		for range s.updateTicker.C {
			snapshot := s.data.GetSnapshot()
			
			// Broadcast different update types
			s.wsHub.BroadcastUpdate("status", map[string]interface{}{
				"currentSlot":    snapshot.CurrentSlot,
				"tipSlot":        snapshot.TipSlot,
				"syncPercentage": snapshot.SyncPercentage,
				"blocksPerSec":   snapshot.BlocksPerSec,
				"currentEra":     snapshot.CurrentEra,
			})
			
			// Send performance update every 5 seconds
			if time.Now().Unix()%5 == 0 {
				s.wsHub.BroadcastUpdate("performance", map[string]interface{}{
					"history": snapshot.PerformanceHistory,
				})
			}
		}
	}()
}

// SetAuthConfig updates the authentication configuration
func (s *Server) SetAuthConfig(enabled bool, username, password, secret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.authConfig = auth.NewConfigFromToml(enabled, username, password, secret)
	
	// Re-initialize JWT secret with new config
	auth.InitJWTSecret(s.authConfig)
}

// Update methods to be called from the indexer

// UpdateStatus updates the server status
func (s *Server) UpdateStatus(status string) {
	s.data.UpdateStatus(status)
}

// UpdateEraProgress updates progress for a specific era
func (s *Server) UpdateEraProgress(era string, progress float64) {
	s.data.UpdateEraProgress(era, progress)
	
	// Broadcast era update
	s.wsHub.BroadcastUpdate("era", map[string]interface{}{
		"era":      era,
		"progress": progress,
	})
}

// UpdatePerformance updates performance metrics
func (s *Server) UpdatePerformance(slot, tip uint64, blocksPerSec float64, totalBlocks int64, mem, cpu string) {
	s.data.UpdatePerformance(slot, tip, blocksPerSec, totalBlocks, mem, cpu)
}

// AddActivity adds a new activity
func (s *Server) AddActivity(activityType, message string, data map[string]interface{}) {
	s.data.AddActivity(activityType, message, data)
	
	// Get the latest activity
	snapshot := s.data.GetSnapshot()
	if len(snapshot.Activities) > 0 {
		s.wsHub.BroadcastUpdate("activity", snapshot.Activities[0])
	}
}

// AddError adds a new error
func (s *Server) AddError(errorType, component, message string) {
	s.data.AddError(errorType, component, message)
	
	// Broadcast error update
	snapshot := s.data.GetSnapshot()
	s.wsHub.BroadcastUpdate("error", map[string]interface{}{
		"totalErrors": snapshot.TotalErrors,
		"latest":      snapshot.RecentErrors[0],
	})
}