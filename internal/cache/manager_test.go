package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tempDir := t.TempDir()

	manager := NewManager(tempDir, 0)
	assert.NotNil(t, manager)
	assert.Equal(t, tempDir, manager.cachePath)
	assert.Equal(t, int64(0), manager.maxSizeBytes)
}

func TestAddEntry(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir, 0)

	// Create a test file
	testFile := filepath.Join(tempDir, "test_video.mp4")
	testData := []byte("test video content")
	err := os.WriteFile(testFile, testData, 0644)
	require.NoError(t, err)

	// Add entry
	err = manager.AddEntry("test_video", "test_video.mp4")
	require.NoError(t, err)

	// Verify entry exists
	entry, err := manager.GetEntry("test_video")
	require.NoError(t, err)
	assert.Equal(t, "test_video", entry.ID)
	assert.Equal(t, "test_video.mp4", entry.FileName)
	assert.Equal(t, int64(len(testData)), entry.Size)
}

func TestGetEntry(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir, 0)

	// Add entry
	testFile := filepath.Join(tempDir, "video.mp4")
	os.WriteFile(testFile, []byte("content"), 0644)
	manager.AddEntry("video", "video.mp4")

	// Get existing entry
	entry, err := manager.GetEntry("video")
	require.NoError(t, err)
	assert.Equal(t, "video", entry.ID)

	// Get non-existing entry
	_, err = manager.GetEntry("nonexistent")
	assert.ErrorIs(t, err, ErrEntryNotFound)
}

func TestDeleteEntry(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir, 0)

	// Create and add entry
	testFile := filepath.Join(tempDir, "video.mp4")
	os.WriteFile(testFile, []byte("content"), 0644)
	manager.AddEntry("video", "video.mp4")

	// Delete entry
	err := manager.DeleteEntry("video")
	require.NoError(t, err)

	// Verify entry is gone
	_, err = manager.GetEntry("video")
	assert.ErrorIs(t, err, ErrEntryNotFound)

	// Verify file is deleted
	assert.NoFileExists(t, testFile)
}

func TestListEntries(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir, 0)

	// Add multiple entries
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("video%d.mp4", i))
		os.WriteFile(filename, []byte("content"), 0644)
		manager.AddEntry(fmt.Sprintf("video%d", i), fmt.Sprintf("video%d.mp4", i))
	}

	entries := manager.ListEntries()
	assert.Equal(t, 3, len(entries))
}

func TestGetSize(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir, 0)

	// Add entries with known sizes
	file1 := filepath.Join(tempDir, "video1.mp4")
	file2 := filepath.Join(tempDir, "video2.mp4")
	os.WriteFile(file1, make([]byte, 1000), 0644)
	os.WriteFile(file2, make([]byte, 2000), 0644)

	manager.AddEntry("video1", "video1.mp4")
	manager.AddEntry("video2", "video2.mp4")

	size := manager.GetSize()
	assert.Equal(t, int64(3000), size)
}

func TestClear(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir, 0)

	// Add entries
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("video%d.mp4", i))
		os.WriteFile(filename, []byte("content"), 0644)
		manager.AddEntry(fmt.Sprintf("video%d", i), fmt.Sprintf("video%d.mp4", i))
	}

	// Clear cache
	err := manager.Clear()
	require.NoError(t, err)

	// Verify all entries are gone
	entries := manager.ListEntries()
	assert.Equal(t, 0, len(entries))
}

func TestLRUEviction(t *testing.T) {
	tempDir := t.TempDir()
	// Set max size to 2000 bytes (convert bytes to GB)
	maxSizeGB := 2000.0 / (1024 * 1024 * 1024)
	manager := NewManager(tempDir, maxSizeGB)

	// Add 3 files of 1000 bytes each
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(tempDir, fmt.Sprintf("video%d.mp4", i))
		os.WriteFile(filename, make([]byte, 1000), 0644)
		manager.AddEntry(fmt.Sprintf("video%d", i), fmt.Sprintf("video%d.mp4", i))
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Only 2 files should remain (most recent)
	entries := manager.ListEntries()
	assert.LessOrEqual(t, len(entries), 2)

	// Total size should be under limit
	assert.LessOrEqual(t, manager.GetSize(), int64(2000))
}

func TestScan(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir, 0)

	// Create files directly in cache directory
	file1 := filepath.Join(tempDir, "VIDEO_ID1.mp4")
	file2 := filepath.Join(tempDir, "VIDEO_ID2.webm")
	file3 := filepath.Join(tempDir, "index.html") // Should be ignored

	os.WriteFile(file1, []byte("video1"), 0644)
	os.WriteFile(file2, []byte("video2"), 0644)
	os.WriteFile(file3, []byte("html"), 0644)

	// Scan directory
	err := manager.Scan()
	require.NoError(t, err)

	// Should have 2 video entries
	entries := manager.ListEntries()
	assert.Equal(t, 2, len(entries))

	// Verify entries
	_, err = manager.GetEntry("VIDEO_ID1")
	assert.NoError(t, err)
	_, err = manager.GetEntry("VIDEO_ID2")
	assert.NoError(t, err)
}

func TestUpdateLastAccess(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir, 0)

	// Add entry
	testFile := filepath.Join(tempDir, "video.mp4")
	os.WriteFile(testFile, []byte("content"), 0644)
	manager.AddEntry("video", "video.mp4")

	// Get initial access time
	entry1, _ := manager.GetEntry("video")
	time.Sleep(100 * time.Millisecond)

	// Update access time
	err := manager.UpdateLastAccess("video")
	require.NoError(t, err)

	// Verify access time changed
	entry2, _ := manager.GetEntry("video")
	assert.True(t, entry2.LastAccess.After(entry1.LastAccess))
}

func TestGetFilePath(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewManager(tempDir, 0)

	// Add entry
	testFile := filepath.Join(tempDir, "video.mp4")
	os.WriteFile(testFile, []byte("content"), 0644)
	manager.AddEntry("video", "video.mp4")

	// Get file path
	path, err := manager.GetFilePath("video")
	require.NoError(t, err)
	assert.Equal(t, testFile, path)

	// Non-existent entry
	_, err = manager.GetFilePath("nonexistent")
	assert.ErrorIs(t, err, ErrEntryNotFound)
}
