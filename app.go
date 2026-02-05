package main

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"

	"vrcvideocacher/internal/api"
	"vrcvideocacher/internal/cache"
	"vrcvideocacher/internal/config"
	"vrcvideocacher/internal/patcher"
	"vrcvideocacher/internal/ytdl"
	"vrcvideocacher/pkg/models"
)

//go:embed resources/ytdlp-stub.exe
var stubData []byte

// App struct
type App struct {
	ctx           context.Context
	configManager *config.Manager
	cacheManager  *cache.Manager
	server        *api.Server
	patcher       *patcher.Patcher
	ytdlManager   *ytdl.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize configuration
	configPath := config.GetDefaultConfigPath()
	cfgManager, err := config.NewManager(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}
	a.configManager = cfgManager

	cfg := cfgManager.Get()

	// Set cache path if not configured
	if cfg.CachePath == "" {
		cfg.CachePath = filepath.Join(config.GetDataDir(), "cache")
		cfgManager.Update(func(c *models.Config) {
			c.CachePath = cfg.CachePath
		})
	}

	// Initialize cache manager
	a.cacheManager = cache.NewManager(cfg.CachePath, cfg.CacheMaxSizeGB)

	// Initialize HTTP server
	a.server = api.NewServer(cfg, a.cacheManager)

	// Initialize patcher
	a.patcher = patcher.NewPatcher(stubData)

	// Initialize yt-dlp manager
	utilsDir := filepath.Join(config.GetDataDir(), "Utils")
	a.ytdlManager = ytdl.NewManager(utilsDir)

	// Ensure yt-dlp is installed
	if err := a.ytdlManager.EnsureInstalled(); err != nil {
		fmt.Printf("Warning: Failed to install yt-dlp: %v\n", err)
	}

	// Auto-update yt-dlp if configured
	if cfg.YtdlAutoUpdate {
		if err := a.ytdlManager.AutoUpdate(); err != nil {
			fmt.Printf("Warning: Failed to update yt-dlp: %v\n", err)
		}
	}

	// Update config with yt-dlp path
	if cfg.YtdlPath == "" || cfg.YtdlPath == "Utils/yt-dlp.exe" {
		cfgManager.Update(func(c *models.Config) {
			c.YtdlPath = a.ytdlManager.GetYtdlpPath()
		})
	}

	// Auto-start server if configured
	if err := a.server.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}

	// Auto-patch VRChat if configured
	if cfg.PatchVRC {
		if err := a.PatchVRChat(); err != nil {
			fmt.Printf("Failed to patch VRChat: %v\n", err)
		}
	}
}

// GetConfig returns the current configuration
func (a *App) GetConfig() *models.Config {
	return a.configManager.Get()
}

// UpdateConfig updates the configuration
func (a *App) UpdateConfig(cfg *models.Config) error {
	return a.configManager.Update(func(c *models.Config) {
		*c = *cfg
	})
}

// StartServer starts the HTTP server
func (a *App) StartServer() error {
	return a.server.Start()
}

// StopServer stops the HTTP server
func (a *App) StopServer() error {
	return a.server.Stop()
}

// IsServerRunning returns whether the server is running
func (a *App) IsServerRunning() bool {
	return a.server.IsRunning()
}

// GetServerStatus returns server status information
func (a *App) GetServerStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":       a.server.IsRunning(),
		"addr":          a.server.GetActualAddr(),
		"cacheSize":     a.cacheManager.GetSize(),
		"cacheEntries":  len(a.cacheManager.ListEntries()),
	}
}

// PatchVRChat patches VRChat's yt-dlp.exe
func (a *App) PatchVRChat() error {
	toolsPath, err := patcher.DetectVRChatPath()
	if err != nil {
		return err
	}

	return a.patcher.PatchVRChat(toolsPath)
}

// UnpatchVRChat restores VRChat's original yt-dlp.exe
func (a *App) UnpatchVRChat() error {
	toolsPath, err := patcher.DetectVRChatPath()
	if err != nil {
		return err
	}

	return a.patcher.UnpatchVRChat(toolsPath)
}

// IsVRChatPatched checks if VRChat is patched
func (a *App) IsVRChatPatched() (bool, error) {
	toolsPath, err := patcher.DetectVRChatPath()
	if err != nil {
		return false, err
	}

	return a.patcher.IsPatched(toolsPath)
}

// GetCacheEntries returns all cache entries
func (a *App) GetCacheEntries() []*models.CacheEntry {
	return a.cacheManager.ListEntries()
}

// ClearCache clears all cache entries
func (a *App) ClearCache() error {
	return a.cacheManager.Clear()
}

// DeleteCacheEntry deletes a specific cache entry
func (a *App) DeleteCacheEntry(id string) error {
	return a.cacheManager.DeleteEntry(id)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
