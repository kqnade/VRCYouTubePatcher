package ytdl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	assert.NotNil(t, mgr)
	assert.Equal(t, utilsDir, mgr.utilsDir)
}

func TestGetYtdlpPath(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	expected := filepath.Join(utilsDir, "yt-dlp.exe")
	assert.Equal(t, expected, mgr.GetYtdlpPath())
}

func TestIsInstalled(t *testing.T) {
	tests := []struct {
		name        string
		createFile  bool
		wantInstalled bool
	}{
		{
			name:        "not installed",
			createFile:  false,
			wantInstalled: false,
		},
		{
			name:        "installed",
			createFile:  true,
			wantInstalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utilsDir := t.TempDir()
			mgr := NewManager(utilsDir)

			if tt.createFile {
				ytdlpPath := mgr.GetYtdlpPath()
				err := os.WriteFile(ytdlpPath, []byte("fake"), 0755)
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantInstalled, mgr.IsInstalled())
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	// Initially should be empty
	version := mgr.GetCurrentVersion()
	assert.Equal(t, "", version)

	// After setting
	mgr.currentVersion = "2024.01.01"
	version = mgr.GetCurrentVersion()
	assert.Equal(t, "2024.01.01", version)
}

func TestDetectPlatform(t *testing.T) {
	platform := detectPlatform()

	// Should return one of the expected platforms
	validPlatforms := []string{"yt-dlp.exe", "yt-dlp_linux", "yt-dlp_linux_aarch64", "yt-dlp_macos", "yt-dlp_macos_arm64", "yt-dlp"}
	assert.Contains(t, validPlatforms, platform)
}

func TestCheckForUpdate_NotInstalled(t *testing.T) {
	utilsDir := t.TempDir()
	_ = NewManager(utilsDir)

	// Mock HTTP client would go here in production
	// For now, skip actual API call in unit test
	t.Skip("Requires mock HTTP client")
}

func TestEnsureInstalled_AlreadyInstalled(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	// Create fake yt-dlp
	ytdlpPath := mgr.GetYtdlpPath()
	err := os.WriteFile(ytdlpPath, []byte("fake"), 0755)
	require.NoError(t, err)

	// Should not download
	err = mgr.EnsureInstalled()
	require.NoError(t, err)
}

func TestGetYtdlpPath_CrossPlatform(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	path := mgr.GetYtdlpPath()
	assert.Contains(t, path, utilsDir)
	assert.Contains(t, path, detectPlatform())
}
