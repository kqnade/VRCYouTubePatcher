package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"vrcvideocacher/internal/cache"
	"vrcvideocacher/internal/downloader"
	"vrcvideocacher/pkg/models"
)

var (
	ErrServerAlreadyRunning = errors.New("server is already running")
	ErrServerNotRunning     = errors.New("server is not running")
)

// Server represents the HTTP server
type Server struct {
	config     *models.Config
	cache      *cache.Manager
	downloader *downloader.Downloader
	router     *chi.Mux
	server     *http.Server
	listener   net.Listener
	running    bool
	mu         sync.RWMutex
}

// NewServer creates a new HTTP server
func NewServer(config *models.Config, cache *cache.Manager) *Server {
	dl := downloader.NewDownloader(config, cache, 2)

	s := &Server{
		config:     config,
		cache:      cache,
		downloader: dl,
		router:     chi.NewRouter(),
	}

	s.setupRoutes()

	return s
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Middleware
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(30 * time.Second))

	// API routes
	s.router.Route("/api", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Get("/status", s.handleStatus)
		r.Get("/getvideo", s.handleGetVideo)
		r.Post("/youtube-cookies", s.handleYouTubeCookies)
	})

	// Static file serving (cache directory)
	fileServer := http.FileServer(http.Dir(s.cache.GetCachePath()))
	s.router.Handle("/*", fileServer)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrServerAlreadyRunning
	}

	addr := s.GetAddr()

	// Create listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	s.listener = listener
	httpServer := &http.Server{
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	s.server = httpServer

	s.running = true

	// Start downloader
	if err := s.downloader.Start(); err != nil {
		return fmt.Errorf("failed to start downloader: %w", err)
	}

	// Start server in goroutine
	go func() {
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrServerNotRunning
	}

	// Stop downloader first
	if err := s.downloader.Stop(); err != nil {
		fmt.Printf("Downloader stop error: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.running = false
	s.server = nil
	s.listener = nil

	return nil
}

// IsRunning returns whether the server is currently running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetAddr returns the server address
func (s *Server) GetAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", s.config.WebServerPort)
}

// GetActualAddr returns the actual listening address (useful when port is 0)
func (s *Server) GetActualAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.listener != nil {
		return s.listener.Addr().String()
	}

	return s.GetAddr()
}

// handleHealth handles health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleStatus handles status endpoint
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	cacheSize := s.cache.GetSize()
	cacheEntries := s.cache.ListEntries()

	response := map[string]interface{}{
		"running":    running,
		"cacheSize":  cacheSize,
		"cacheCount": len(cacheEntries),
		"version":    "0.1.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
