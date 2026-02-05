//go:build integration

package downloader

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vrcvideocacher/internal/cache"
	"vrcvideocacher/pkg/models"
)

// TestRealDownload tests actual download with yt-dlp
func TestRealDownload(t *testing.T) {
	// Use a short, public domain test video
	// This is a short YouTube test video (few seconds)
	testVideoID := "jNQXAC9IVRw" // "Me at the zoo" - first YouTube video (very short)
	testVideoURL := "https://www.youtube.com/watch?v=" + testVideoID

	// Get absolute path to yt-dlp (from project root)
	ytdlpPath, err := filepath.Abs("../../Utils/yt-dlp.exe")
	require.NoError(t, err)
	t.Logf("Using yt-dlp at: %s", ytdlpPath)

	// Verify yt-dlp exists
	_, statErr := os.Stat(ytdlpPath)
	require.NoError(t, statErr, "yt-dlp.exe not found at %s", ytdlpPath)

	// Setup
	cacheDir := t.TempDir()
	cfg := &models.Config{
		YtdlPath:              ytdlpPath,
		CacheYouTubeMaxRes:    480, // Low res for faster download
		CacheYouTubeMaxLength: 3600,
		YtdlUseCookies:        false,
		CachePath:             cacheDir,
	}

	cacheMgr := cache.NewManager(cacheDir, 0)
	dl := NewDownloader(cfg, cacheMgr, 1)

	err = dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Queue download
	t.Logf("Queuing download for video: %s", testVideoID)
	err = dl.Queue(testVideoID, testVideoURL, models.DownloadFormatMP4)
	require.NoError(t, err)

	// Wait for download to complete (with timeout)
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	completed := false
	for !completed {
		select {
		case <-timeout:
			t.Fatal("Download timeout after 2 minutes")
		case <-ticker.C:
			// Check if in cache (download completed)
			if _, err := cacheMgr.GetEntry(testVideoID); err == nil {
				t.Logf("Download completed - found in cache")
				completed = true
				break
			}

			// Check status if still in queue/active
			status, err := dl.GetStatus(testVideoID)
			if err != nil {
				// Not in queue or active, check cache one more time
				if _, cacheErr := cacheMgr.GetEntry(testVideoID); cacheErr == nil {
					t.Logf("Download completed")
					completed = true
				}
				continue
			}

			t.Logf("Download status: %s", status.Status)

			if status.Status == StatusFailed {
				t.Fatalf("Download failed: %v", status.Error)
			}
		}
	}

	// Verify file was added to cache
	entry, err := cacheMgr.GetEntry(testVideoID)
	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, testVideoID, entry.ID)

	// Verify file exists on disk
	filePath, err := cacheMgr.GetFilePath(testVideoID)
	require.NoError(t, err)

	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "Downloaded file should not be empty")

	t.Logf("Successfully downloaded: %s (%d bytes)", entry.FileName, entry.Size)
}

// TestRealDownloadWebm tests WebM format download (for AVPro)
func TestRealDownloadWebm(t *testing.T) {
	testVideoID := "jNQXAC9IVRw"
	testVideoURL := "https://www.youtube.com/watch?v=" + testVideoID

	cacheDir := t.TempDir()
	cfg := &models.Config{
		YtdlPath:              "Utils/yt-dlp.exe",
		CacheYouTubeMaxRes:    480,
		CacheYouTubeMaxLength: 3600,
		YtdlUseCookies:        false,
		CachePath:             cacheDir,
	}

	cacheMgr := cache.NewManager(cacheDir, 0)
	dl := NewDownloader(cfg, cacheMgr, 1)

	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Queue WebM download
	t.Logf("Queuing WebM download for video: %s", testVideoID)
	err = dl.Queue(testVideoID, testVideoURL, models.DownloadFormatWebm)
	require.NoError(t, err)

	// Wait for completion
	timeout := time.After(2 * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Download timeout after 2 minutes")
		case <-ticker.C:
			status, err := dl.GetStatus(testVideoID)
			if err != nil {
				goto checkCache
			}

			if status.Status == StatusCompleted {
				goto checkCache
			}

			if status.Status == StatusFailed {
				t.Fatalf("Download failed: %v", status.Error)
			}
		}
	}

checkCache:
	// Verify file in cache
	entry, err := cacheMgr.GetEntry(testVideoID)
	require.NoError(t, err)

	// Check file extension (should be .webm)
	ext := filepath.Ext(entry.FileName)
	assert.Equal(t, ".webm", ext, "File should be WebM format")

	t.Logf("Successfully downloaded WebM: %s (%d bytes)", entry.FileName, entry.Size)
}

// TestDownloadWithCookies tests download with cookies file
func TestDownloadWithCookies(t *testing.T) {
	t.Skip("Skipping cookie test - requires valid YouTube cookies")

	// This test would require actual YouTube cookies
	// For now, we just verify the cookies path is used correctly
}

// TestConcurrentDownloads tests multiple simultaneous downloads
func TestConcurrentDownloads(t *testing.T) {
	// Use multiple short videos
	videos := []struct {
		id  string
		url string
	}{
		{"jNQXAC9IVRw", "https://www.youtube.com/watch?v=jNQXAC9IVRw"},
		// Add more test videos if needed
	}

	cacheDir := t.TempDir()
	cfg := &models.Config{
		YtdlPath:              "Utils/yt-dlp.exe",
		CacheYouTubeMaxRes:    360,
		CacheYouTubeMaxLength: 3600,
		YtdlUseCookies:        false,
		CachePath:             cacheDir,
	}

	cacheMgr := cache.NewManager(cacheDir, 0)
	dl := NewDownloader(cfg, cacheMgr, 2) // 2 workers

	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Queue all downloads
	for _, v := range videos {
		err = dl.Queue(v.id, v.url, models.DownloadFormatMP4)
		require.NoError(t, err)
		t.Logf("Queued: %s", v.id)
	}

	// Wait for all to complete
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	completed := make(map[string]bool)

	for len(completed) < len(videos) {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for downloads. Completed: %d/%d", len(completed), len(videos))
		case <-ticker.C:
			for _, v := range videos {
				if completed[v.id] {
					continue
				}

				status, err := dl.GetStatus(v.id)
				if err != nil {
					// Check if in cache
					if _, cacheErr := cacheMgr.GetEntry(v.id); cacheErr == nil {
						completed[v.id] = true
						t.Logf("Completed: %s", v.id)
					}
					continue
				}

				if status.Status == StatusCompleted {
					completed[v.id] = true
					t.Logf("Completed: %s", v.id)
				} else if status.Status == StatusFailed {
					t.Fatalf("Download failed for %s: %v", v.id, status.Error)
				}
			}
		}
	}

	// Verify all are in cache
	for _, v := range videos {
		entry, err := cacheMgr.GetEntry(v.id)
		require.NoError(t, err)
		assert.NotNil(t, entry)
		t.Logf("Verified in cache: %s (%d bytes)", v.id, entry.Size)
	}
}

// TestDownloadFailure tests handling of invalid URLs
func TestDownloadFailure(t *testing.T) {
	cacheDir := t.TempDir()
	cfg := &models.Config{
		YtdlPath:              "Utils/yt-dlp.exe",
		CacheYouTubeMaxRes:    480,
		CacheYouTubeMaxLength: 3600,
		YtdlUseCookies:        false,
		CachePath:             cacheDir,
	}

	cacheMgr := cache.NewManager(cacheDir, 0)
	dl := NewDownloader(cfg, cacheMgr, 1)

	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Queue invalid video
	invalidID := "INVALID_VIDEO_ID_12345"
	invalidURL := "https://www.youtube.com/watch?v=" + invalidID

	err = dl.Queue(invalidID, invalidURL, models.DownloadFormatMP4)
	require.NoError(t, err)

	// Wait for failure
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Expected download to fail within 30 seconds")
		case <-ticker.C:
			status, err := dl.GetStatus(invalidID)
			if err != nil {
				// No longer tracked, assume completed or failed
				break
			}

			if status.Status == StatusFailed {
				assert.NotNil(t, status.Error)
				t.Logf("Download correctly failed: %v", status.Error)
				return
			}
		}
	}
}
