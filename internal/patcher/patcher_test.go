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

// TestPatchVRChat_FileNotFound tests error when yt-dlp.exe doesn't exist
func TestPatchVRChat_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	patcher := NewPatcher(stubData)

	// Try to patch when file doesn't exist
	err := patcher.PatchVRChat(toolsDir)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

// TestDetectVRChatPath_NoLocalAppData tests when LOCALAPPDATA is not set
func TestDetectVRChatPath_NoLocalAppData(t *testing.T) {
	// Save original
	original := os.Getenv("LOCALAPPDATA")
	defer os.Setenv("LOCALAPPDATA", original)

	// Clear LOCALAPPDATA
	os.Unsetenv("LOCALAPPDATA")

	_, err := DetectVRChatPath()
	assert.ErrorIs(t, err, ErrVRChatNotFound)
}

// TestDetectVRChatPath_DirectoryNotFound tests when VRChat directory doesn't exist
func TestDetectVRChatPath_DirectoryNotFound(t *testing.T) {
	// Save original
	original := os.Getenv("LOCALAPPDATA")
	defer os.Setenv("LOCALAPPDATA", original)

	// Set to temp directory that won't have VRChat
	tempDir := t.TempDir()
	os.Setenv("LOCALAPPDATA", tempDir)

	_, err := DetectVRChatPath()
	assert.ErrorIs(t, err, ErrVRChatNotFound)
}

// TestUnpatchVRChat_NoStubFile tests unpatch when stub file doesn't exist
func TestUnpatchVRChat_NoStubFile(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	backupPath := filepath.Join(toolsDir, "yt-dlp.exe.bkp")
	originalData := []byte("original yt-dlp")
	os.WriteFile(backupPath, originalData, 0644)

	patcher := NewPatcher(stubData)

	// Unpatch when stub doesn't exist
	err := patcher.UnpatchVRChat(toolsDir)
	require.NoError(t, err)

	// Verify file was restored
	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")
	restoredData, _ := os.ReadFile(ytdlpPath)
	assert.Equal(t, originalData, restoredData)
}

// TestPatchVRChat_ReadOnlyBackup tests patching with existing backup
func TestPatchVRChat_ReadOnlyBackup(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")
	backupPath := filepath.Join(toolsDir, "yt-dlp.exe.bkp")

	originalData := []byte("original yt-dlp")
	os.WriteFile(ytdlpPath, originalData, 0644)
	os.WriteFile(backupPath, []byte("existing backup"), 0644)

	patcher := NewPatcher(stubData)

	// Patch with existing backup
	err := patcher.PatchVRChat(toolsDir)
	require.NoError(t, err)

	// Verify existing backup wasn't overwritten
	backupData, _ := os.ReadFile(backupPath)
	assert.Equal(t, []byte("existing backup"), backupData)

	// Verify yt-dlp.exe is now stub
	patchedData, _ := os.ReadFile(ytdlpPath)
	assert.Equal(t, stubData, patchedData)
}

// TestPatchVRChat_AlreadyReadOnly tests patching when file is already read-only
func TestPatchVRChat_AlreadyReadOnly(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")
	originalData := []byte("original yt-dlp")
	os.WriteFile(ytdlpPath, originalData, 0444) // Create as read-only

	patcher := NewPatcher(stubData)

	// Patch should handle read-only file
	err := patcher.PatchVRChat(toolsDir)
	require.NoError(t, err)

	// Verify patched
	patchedData, _ := os.ReadFile(ytdlpPath)
	assert.Equal(t, stubData, patchedData)
}

// TestUnpatchVRChat_StubReadOnly tests unpatch when stub is read-only
func TestUnpatchVRChat_StubReadOnly(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")
	backupPath := filepath.Join(toolsDir, "yt-dlp.exe.bkp")

	originalData := []byte("original yt-dlp")
	os.WriteFile(ytdlpPath, stubData, 0444) // Create as read-only
	os.WriteFile(backupPath, originalData, 0644)

	patcher := NewPatcher(stubData)

	// Unpatch should handle read-only stub
	err := patcher.UnpatchVRChat(toolsDir)
	require.NoError(t, err)

	// Verify restored
	restoredData, _ := os.ReadFile(ytdlpPath)
	assert.Equal(t, originalData, restoredData)
}

// TestMakeReadOnly tests makeReadOnly function
func TestMakeReadOnly(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	err = makeReadOnly(testFile)
	require.NoError(t, err)

	// Verify file is read-only
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.True(t, info.Mode().Perm()&0200 == 0)
}

// TestMakeWritable tests makeWritable function
func TestMakeWritable(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	err := os.WriteFile(testFile, []byte("test"), 0444)
	require.NoError(t, err)

	err = makeWritable(testFile)
	require.NoError(t, err)

	// Verify file is writable
	info, err := os.Stat(testFile)
	require.NoError(t, err)
	assert.True(t, info.Mode().Perm()&0200 != 0)
}

// TestPatchVRChat_RemoveNonExistentFile tests removing file that doesn't exist
func TestPatchVRChat_RemoveNonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")

	// Create and immediately remove to test the os.IsNotExist path
	os.WriteFile(ytdlpPath, []byte("temp"), 0644)

	patcher := NewPatcher(stubData)

	// First call should succeed
	err := patcher.PatchVRChat(toolsDir)
	require.NoError(t, err)

	// Verify patched
	patchedData, _ := os.ReadFile(ytdlpPath)
	assert.Equal(t, stubData, patchedData)
}

// TestIsPatched_DirectoryInsteadOfFile tests IsPatched when path is a directory
func TestIsPatched_DirectoryInsteadOfFile(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	// Create yt-dlp.exe as a directory instead of a file
	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")
	os.Mkdir(ytdlpPath, 0755)

	patcher := NewPatcher(stubData)

	// Should return error when trying to read directory as file
	patched, err := patcher.IsPatched(toolsDir)
	assert.Error(t, err)
	assert.False(t, patched)
}

// TestPatchVRChat_MultiplePatches tests patching multiple times
func TestPatchVRChat_MultiplePatches(t *testing.T) {
	tempDir := t.TempDir()
	stubData := []byte("test stub")

	toolsDir := filepath.Join(tempDir, "Tools")
	os.MkdirAll(toolsDir, 0755)

	ytdlpPath := filepath.Join(toolsDir, "yt-dlp.exe")
	originalData := []byte("original yt-dlp")
	os.WriteFile(ytdlpPath, originalData, 0644)

	patcher := NewPatcher(stubData)

	// First patch
	err := patcher.PatchVRChat(toolsDir)
	require.NoError(t, err)

	// Verify patched
	patched, err := patcher.IsPatched(toolsDir)
	require.NoError(t, err)
	assert.True(t, patched)

	// Second patch should be no-op
	err = patcher.PatchVRChat(toolsDir)
	require.NoError(t, err)

	// Third patch should still be no-op
	err = patcher.PatchVRChat(toolsDir)
	require.NoError(t, err)

	// Still patched
	patched, err = patcher.IsPatched(toolsDir)
	require.NoError(t, err)
	assert.True(t, patched)
}
