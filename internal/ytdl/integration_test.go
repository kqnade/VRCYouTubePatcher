//go:build integration

package ytdl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadYtdlp(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	// Download yt-dlp
	err := mgr.Download()
	require.NoError(t, err)

	// Verify installation
	assert.True(t, mgr.IsInstalled())

	// Verify file exists and is executable
	ytdlpPath := mgr.GetYtdlpPath()
	info, err := os.Stat(ytdlpPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(1000000), "yt-dlp should be at least 1MB")

	// Verify version is set
	assert.NotEmpty(t, mgr.GetCurrentVersion())
	t.Logf("Downloaded yt-dlp version: %s", mgr.GetCurrentVersion())
}

func TestCheckForUpdate(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	// Check for updates (not installed)
	version, hasUpdate, err := mgr.CheckForUpdate()
	require.NoError(t, err)
	assert.True(t, hasUpdate, "Should have update when not installed")
	assert.NotEmpty(t, version)
	t.Logf("Latest version: %s", version)
}

func TestEnsureInstalled(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	// First call should download
	err := mgr.EnsureInstalled()
	require.NoError(t, err)
	assert.True(t, mgr.IsInstalled())

	// Second call should be no-op
	err = mgr.EnsureInstalled()
	require.NoError(t, err)
}
