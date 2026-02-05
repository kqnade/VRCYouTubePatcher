package patcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPatcher(t *testing.T) {
	stubData := []byte("stub executable")

	patcher := NewPatcher(stubData)
	require.NotNil(t, patcher)
	assert.Equal(t, stubData, patcher.stubData)
}

func TestDetectVRChatPath(t *testing.T) {
	// This test will be platform-specific
	// For now, just test that it doesn't crash
	path, err := DetectVRChatPath()
	// May or may not find VRChat on test machine
	if err == nil {
		assert.NotEmpty(t, path)
	}
}

func TestPatchVRChat(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	// Create mock VRChat Tools directory
	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	// Create original yt-dlp.exe
	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")
	originalData := []byte("original yt-dlp")
	err := os.WriteFile(ytdlpPath, originalData, 0644)
	require.NoError(t, err)

	patcher := NewPatcher(stubData)

	// Patch
	err = patcher.PatchVRChat(toolsDir)
	require.NoError(t, err)

	// Verify backup exists
	backupPath := filepath.Join(toolsDir, "yt-dlp.exe.bkp")
	assert.FileExists(t, backupPath)

	// Verify backup contains original data
	backupData, _ := os.ReadFile(backupPath)
	assert.Equal(t, originalData, backupData)

	// Verify yt-dlp.exe is now stub
	patchedData, _ := os.ReadFile(ytdlpPath)
	assert.Equal(t, stubData, patchedData)

	// Verify file is read-only
	info, _ := os.Stat(ytdlpPath)
	mode := info.Mode()
	assert.True(t, mode.Perm()&0200 == 0, "file should be read-only")
}

func TestPatchVRChatAlreadyPatched(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")
	os.WriteFile(ytdlpPath, stubData, 0644)

	patcher := NewPatcher(stubData)

	// First patch
	err := patcher.PatchVRChat(toolsDir)
	require.NoError(t, err)

	// Second patch should detect already patched
	err = patcher.PatchVRChat(toolsDir)
	assert.NoError(t, err) // Should succeed but do nothing
}

func TestUnpatchVRChat(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")
	backupPath := filepath.Join(toolsDir, "yt-dlp.exe.bkp")

	originalData := []byte("original yt-dlp")
	os.WriteFile(ytdlpPath, stubData, 0644)
	os.WriteFile(backupPath, originalData, 0644)

	patcher := NewPatcher(stubData)

	// Unpatch
	err := patcher.UnpatchVRChat(toolsDir)
	require.NoError(t, err)

	// Verify backup is gone
	assert.NoFileExists(t, backupPath)

	// Verify yt-dlp.exe is restored
	restoredData, _ := os.ReadFile(ytdlpPath)
	assert.Equal(t, originalData, restoredData)

	// Verify file is writable
	info, _ := os.Stat(ytdlpPath)
	mode := info.Mode()
	assert.True(t, mode.Perm()&0200 != 0, "file should be writable")
}

func TestUnpatchVRChatNoBackup(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	patcher := NewPatcher(stubData)

	// Unpatch when no backup exists
	err := patcher.UnpatchVRChat(toolsDir)
	assert.NoError(t, err) // Should succeed but do nothing
}

func TestIsPatched(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")

	patcher := NewPatcher(stubData)

	// File doesn't exist
	patched, err := patcher.IsPatched(toolsDir)
	require.Error(t, err)
	assert.False(t, patched)

	// File is original
	os.WriteFile(ytdlpPath, []byte("original"), 0644)
	patched, err = patcher.IsPatched(toolsDir)
	require.NoError(t, err)
	assert.False(t, patched)

	// File is stub
	os.WriteFile(ytdlpPath, stubData, 0644)
	patched, err = patcher.IsPatched(toolsDir)
	require.NoError(t, err)
	assert.True(t, patched)
}

func TestComputeHash(t *testing.T) {
	data := []byte("test data")
	hash1 := computeHash(data)
	hash2 := computeHash(data)

	assert.NotEmpty(t, hash1)
	assert.Equal(t, hash1, hash2) // Same data should give same hash

	differentData := []byte("different data")
	hash3 := computeHash(differentData)
	assert.NotEqual(t, hash1, hash3) // Different data should give different hash
}
