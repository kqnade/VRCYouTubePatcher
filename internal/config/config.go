package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"vrcvideocacher/pkg/models"
)

var (
	ErrInvalidPort       = errors.New("invalid port: must be between 1 and 65535")
	ErrInvalidResolution = errors.New("invalid resolution: must be between 144 and 4320")
	ErrInvalidCacheSize  = errors.New("invalid cache size: must be non-negative")
)

// Manager handles configuration loading, saving, and updates
type Manager struct {
	mu         sync.RWMutex
	config     *models.Config
	configPath string
}

// NewManager creates a new configuration manager
// If the config file doesn't exist, it creates one with default values
func NewManager(configPath string) (*Manager, error) {
	manager := &Manager{
		configPath: configPath,
		config:     models.DefaultConfig(),
	}

	// Try to load existing config
	if _, err := os.Stat(configPath); err == nil {
		if err := manager.load(); err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	} else {
		// Create config directory if it doesn't exist
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		// Save default config
		if err := manager.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}

	// Validate config
	if err := Validate(manager.config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return manager, nil
}

// Get returns a copy of the current configuration
func (m *Manager) Get() *models.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	cfg := *m.config
	return &cfg
}

// Update applies a function to the configuration and saves it
func (m *Manager) Update(fn func(*models.Config)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Apply updates
	fn(m.config)

	// Validate
	if err := Validate(m.config); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Save to disk
	return m.save()
}

// Save writes the current configuration to disk
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.save()
}

// load reads configuration from disk
func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into a temporary config
	var cfg models.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Merge with defaults (for new fields)
	m.config = mergeWithDefaults(&cfg)

	return nil
}

// save writes configuration to disk (must be called with lock held)
func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// mergeWithDefaults fills in default values for missing fields
func mergeWithDefaults(cfg *models.Config) *models.Config {
	defaults := models.DefaultConfig()

	// Only set defaults if values are zero/empty
	if cfg.WebServerURL == "" {
		cfg.WebServerURL = defaults.WebServerURL
	}
	if cfg.WebServerPort == 0 {
		cfg.WebServerPort = defaults.WebServerPort
	}
	if cfg.YtdlPath == "" {
		cfg.YtdlPath = defaults.YtdlPath
	}
	if cfg.CacheYouTubeMaxRes == 0 {
		cfg.CacheYouTubeMaxRes = defaults.CacheYouTubeMaxRes
	}
	if cfg.CacheYouTubeMaxLength == 0 {
		cfg.CacheYouTubeMaxLength = defaults.CacheYouTubeMaxLength
	}
	if cfg.BlockedURLs == nil {
		cfg.BlockedURLs = defaults.BlockedURLs
	}

	return cfg
}

// Validate checks if the configuration is valid
func Validate(cfg *models.Config) error {
	// Validate port
	if cfg.WebServerPort < 1 || cfg.WebServerPort > 65535 {
		return ErrInvalidPort
	}

	// Validate resolution
	if cfg.CacheYouTubeMaxRes < 144 || cfg.CacheYouTubeMaxRes > 4320 {
		return ErrInvalidResolution
	}

	// Validate cache size
	if cfg.CacheMaxSizeGB < 0 {
		return ErrInvalidCacheSize
	}

	return nil
}

// GetDataDir returns the application data directory
func GetDataDir() string {
	// Try to use LocalAppData on Windows
	if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
		dataDir := filepath.Join(appData, "VRCVideoCacher")
		os.MkdirAll(dataDir, 0755)
		return dataDir
	}

	// Fallback to home directory
	if home, err := os.UserHomeDir(); err == nil {
		dataDir := filepath.Join(home, ".vrcvideocacher")
		os.MkdirAll(dataDir, 0755)
		return dataDir
	}

	// Last resort: current directory
	return "."
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	return filepath.Join(GetDataDir(), "config.json")
}
