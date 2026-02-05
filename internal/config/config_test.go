package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vrcvideocacher/pkg/models"
)

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	manager, err := NewManager(configPath)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Should create config with defaults
	cfg := manager.Get()
	assert.Equal(t, 9696, cfg.WebServerPort)
	assert.True(t, cfg.PatchVRC)
	assert.False(t, cfg.CacheYouTube)
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		check   func(t *testing.T, cfg *Manager)
	}{
		{
			name: "valid config",
			json: `{
				"webServerPort": 8080,
				"cacheYouTube": true,
				"cacheYouTubeMaxRes": 2160
			}`,
			wantErr: false,
			check: func(t *testing.T, manager *Manager) {
				cfg := manager.Get()
				assert.Equal(t, 8080, cfg.WebServerPort)
				assert.True(t, cfg.CacheYouTube)
				assert.Equal(t, 2160, cfg.CacheYouTubeMaxRes)
			},
		},
		{
			name: "empty config uses defaults",
			json: `{}`,
			wantErr: false,
			check: func(t *testing.T, manager *Manager) {
				cfg := manager.Get()
				assert.Equal(t, 9696, cfg.WebServerPort)
				assert.False(t, cfg.CacheYouTube)
			},
		},
		{
			name:    "invalid JSON",
			json:    `{invalid json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.json")

			if tt.json != "" {
				err := os.WriteFile(configPath, []byte(tt.json), 0644)
				require.NoError(t, err)
			}

			manager, err := NewManager(configPath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, manager)
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	manager, err := NewManager(configPath)
	require.NoError(t, err)

	// Modify config using Update
	err = manager.Update(func(cfg *models.Config) {
		cfg.WebServerPort = 8080
		cfg.CacheYouTube = true
	})
	require.NoError(t, err)

	// Verify file was written
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"webServerPort": 8080`)
	assert.Contains(t, string(data), `"cacheYouTube": true`)
}

func TestUpdate(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	manager, err := NewManager(configPath)
	require.NoError(t, err)

	// Update specific field
	err = manager.Update(func(cfg *models.Config) {
		cfg.WebServerPort = 7777
		cfg.CacheYouTube = true
	})
	require.NoError(t, err)

	// Verify changes persisted
	cfg := manager.Get()
	assert.Equal(t, 7777, cfg.WebServerPort)
	assert.True(t, cfg.CacheYouTube)

	// Verify saved to disk
	newManager, err := NewManager(configPath)
	require.NoError(t, err)
	cfg = newManager.Get()
	assert.Equal(t, 7777, cfg.WebServerPort)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(cfg *models.Config)
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			setup: func(cfg *models.Config) {
				// Use defaults
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			setup: func(cfg *models.Config) {
				cfg.WebServerPort = 0
			},
			wantErr: true,
			errMsg:  "port",
		},
		{
			name: "invalid port - too high",
			setup: func(cfg *models.Config) {
				cfg.WebServerPort = 70000
			},
			wantErr: true,
			errMsg:  "port",
		},
		{
			name: "invalid cache max resolution",
			setup: func(cfg *models.Config) {
				cfg.CacheYouTubeMaxRes = 10000
			},
			wantErr: true,
			errMsg:  "resolution",
		},
		{
			name: "negative cache size",
			setup: func(cfg *models.Config) {
				cfg.CacheMaxSizeGB = -1
			},
			wantErr: true,
			errMsg:  "size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := models.DefaultConfig()
			if tt.setup != nil {
				tt.setup(cfg)
			}

			err := Validate(cfg)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetDataDir(t *testing.T) {
	dir := GetDataDir()
	assert.NotEmpty(t, dir)
	assert.DirExists(t, dir)
}
