package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vrcvideocacher/internal/cache"
	"vrcvideocacher/pkg/models"
)

func TestNewServer(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()

	server := NewServer(cfg, cacheMgr)
	require.NotNil(t, server)
	assert.Equal(t, cfg, server.config)
	assert.Equal(t, cacheMgr, server.cache)
}

func TestServerStart(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()
	cfg.WebServerPort = 0 // Use random available port

	server := NewServer(cfg, cacheMgr)

	// Start server
	err := server.Start()
	require.NoError(t, err)
	assert.True(t, server.IsRunning())

	// Stop server
	err = server.Stop()
	require.NoError(t, err)
	assert.False(t, server.IsRunning())
}

func TestServerStartAlreadyRunning(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()
	cfg.WebServerPort = 0

	server := NewServer(cfg, cacheMgr)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Try to start again
	err = server.Start()
	assert.ErrorIs(t, err, ErrServerAlreadyRunning)
}

func TestServerStopNotRunning(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()

	server := NewServer(cfg, cacheMgr)

	err := server.Stop()
	assert.ErrorIs(t, err, ErrServerNotRunning)
}

func TestStaticFileServing(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()

	// Create a test file in cache
	testFile := filepath.Join(tempDir, "test_video.mp4")
	testContent := []byte("test video content")
	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	server := NewServer(cfg, cacheMgr)

	// Test static file serving
	req := httptest.NewRequest("GET", "/test_video.mp4", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, testContent, w.Body.Bytes())
}

func TestHealthEndpoint(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()

	server := NewServer(cfg, cacheMgr)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body, _ := io.ReadAll(w.Body)
	assert.Contains(t, string(body), "ok")
}

func TestStatusEndpoint(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()

	// Add some test entries
	testFile := filepath.Join(tempDir, "video.mp4")
	os.WriteFile(testFile, make([]byte, 1000), 0644)
	cacheMgr.AddEntry("video", "video.mp4")

	server := NewServer(cfg, cacheMgr)
	server.Start()
	defer server.Stop()

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	body, _ := io.ReadAll(w.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "running")
	assert.Contains(t, bodyStr, "cacheSize")
	assert.Contains(t, bodyStr, "cacheCount")
}

func TestCORSHeaders(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()

	server := NewServer(cfg, cacheMgr)

	req := httptest.NewRequest("OPTIONS", "/api/status", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	// CORS should be disabled for local server
	// But we should handle OPTIONS requests
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetAddr(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()
	cfg.WebServerPort = 8080

	server := NewServer(cfg, cacheMgr)

	addr := server.GetAddr()
	assert.Equal(t, "127.0.0.1:8080", addr)
}

func TestServerGracefulShutdown(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()
	cfg.WebServerPort = 0

	server := NewServer(cfg, cacheMgr)

	err := server.Start()
	require.NoError(t, err)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop should complete gracefully
	done := make(chan bool)
	go func() {
		err := server.Stop()
		assert.NoError(t, err)
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Server shutdown timeout")
	}
}
