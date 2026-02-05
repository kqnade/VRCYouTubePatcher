package downloader

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vrcvideocacher/internal/cache"
	"vrcvideocacher/pkg/models"
)

func TestNewDownloader(t *testing.T) {
	cfg := &models.Config{
		CacheYouTubeMaxRes:    1080,
		CacheYouTubeMaxLength: 120,
		YtdlPath:              "yt-dlp",
	}

	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)

	assert.NotNil(t, dl)
	assert.Equal(t, 2, dl.maxWorkers)
	assert.False(t, dl.running)
}

func TestNewDownloaderWithZeroWorkers(t *testing.T) {
	cfg := &models.Config{}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 0)

	// Should default to 2 workers
	assert.Equal(t, 2, dl.maxWorkers)
}

func TestStartStop(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)

	// Start
	err := dl.Start()
	require.NoError(t, err)
	assert.True(t, dl.running)

	// Start again should not error
	err = dl.Start()
	require.NoError(t, err)

	// Stop
	err = dl.Stop()
	require.NoError(t, err)
	assert.False(t, dl.running)

	// Stop again should not error
	err = dl.Stop()
	require.NoError(t, err)
}

func TestQueueDownload(t *testing.T) {
	cfg := &models.Config{
		CacheYouTubeMaxRes:    1080,
		CacheYouTubeMaxLength: 120,
		YtdlPath:              "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)
	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Queue a download
	err = dl.Queue("TEST123", "https://youtube.com/watch?v=TEST123", models.DownloadFormatMP4)
	require.NoError(t, err)

	assert.Equal(t, 1, dl.GetQueueLength())
}

func TestQueueDownloadWhenStopped(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)

	// Queue without starting should error
	err := dl.Queue("TEST123", "https://youtube.com/watch?v=TEST123", models.DownloadFormatMP4)
	assert.ErrorIs(t, err, ErrDownloaderStopped)
}

func TestQueueDuplicate(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)
	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Queue same video twice
	err = dl.Queue("TEST123", "https://youtube.com/watch?v=TEST123", models.DownloadFormatMP4)
	require.NoError(t, err)

	err = dl.Queue("TEST123", "https://youtube.com/watch?v=TEST123", models.DownloadFormatMP4)
	assert.ErrorIs(t, err, ErrAlreadyQueued)
}

func TestQueueAlreadyCached(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	// Add entry to cache
	testFile := filepath.Join(cacheDir, "TEST123.mp4")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	err = cacheMgr.AddEntry("TEST123", "TEST123.mp4")
	require.NoError(t, err)

	dl := NewDownloader(cfg, cacheMgr, 2)
	err = dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Queue already cached video should not error (no-op)
	err = dl.Queue("TEST123", "https://youtube.com/watch?v=TEST123", models.DownloadFormatMP4)
	assert.NoError(t, err)

	// Should not be in queue
	assert.Equal(t, 0, dl.GetQueueLength())
}

func TestGetStatus(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)
	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Queue a download
	err = dl.Queue("TEST123", "https://youtube.com/watch?v=TEST123", models.DownloadFormatMP4)
	require.NoError(t, err)

	// Get status
	status, err := dl.GetStatus("TEST123")
	require.NoError(t, err)
	assert.Equal(t, "TEST123", status.VideoID)
	assert.Equal(t, StatusQueued, status.Status)
}

func TestGetStatusNotFound(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)
	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Get status for non-existent video
	_, err = dl.GetStatus("NONEXISTENT")
	assert.Error(t, err)
}

func TestDequeue(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	// Don't start workers to prevent them from processing
	dl := NewDownloader(cfg, cacheMgr, 2)

	// Queue some downloads manually
	dl.mu.Lock()
	dl.queue = append(dl.queue, &DownloadRequest{
		VideoID:  "TEST1",
		VideoURL: "https://youtube.com/watch?v=TEST1",
		Format:   models.DownloadFormatMP4,
	})
	dl.queue = append(dl.queue, &DownloadRequest{
		VideoID:  "TEST2",
		VideoURL: "https://youtube.com/watch?v=TEST2",
		Format:   models.DownloadFormatMP4,
	})
	dl.mu.Unlock()

	// Dequeue
	req := dl.dequeue()
	require.NotNil(t, req)
	assert.Equal(t, "TEST1", req.VideoID)

	// Should be in active map
	assert.Equal(t, 1, dl.GetActiveDownloads())
	assert.Equal(t, 1, dl.GetQueueLength())
}

func TestDequeueEmpty(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)

	// Dequeue from empty queue
	req := dl.dequeue()
	assert.Nil(t, req)
}

func TestDownloadStatusString(t *testing.T) {
	tests := []struct {
		status DownloadStatus
		want   string
	}{
		{StatusQueued, "queued"},
		{StatusDownloading, "downloading"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{DownloadStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestWorkerStopsOnContextCancel(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 1)
	err := dl.Start()
	require.NoError(t, err)

	// Stop immediately
	err = dl.Stop()
	require.NoError(t, err)

	// Workers should have stopped
	assert.False(t, dl.running)
}

func TestGetQueueLength(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)
	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	assert.Equal(t, 0, dl.GetQueueLength())

	dl.Queue("TEST1", "https://youtube.com/watch?v=TEST1", models.DownloadFormatMP4)
	assert.Equal(t, 1, dl.GetQueueLength())

	dl.Queue("TEST2", "https://youtube.com/watch?v=TEST2", models.DownloadFormatMP4)
	assert.Equal(t, 2, dl.GetQueueLength())
}

func TestGetActiveDownloads(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)
	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	assert.Equal(t, 0, dl.GetActiveDownloads())

	// Queue and dequeue to make active
	dl.Queue("TEST1", "https://youtube.com/watch?v=TEST1", models.DownloadFormatMP4)
	req := dl.dequeue()
	require.NotNil(t, req)

	assert.Equal(t, 1, dl.GetActiveDownloads())
}

func TestDownloadRequestFields(t *testing.T) {
	cfg := &models.Config{
		CacheYouTubeMaxRes:    1080,
		CacheYouTubeMaxLength: 120,
		YtdlPath:              "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)
	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	before := time.Now()
	err = dl.Queue("TEST123", "https://youtube.com/watch?v=TEST123", models.DownloadFormatWebm)
	require.NoError(t, err)

	status, err := dl.GetStatus("TEST123")
	require.NoError(t, err)

	assert.Equal(t, "TEST123", status.VideoID)
	assert.Equal(t, "https://youtube.com/watch?v=TEST123", status.VideoURL)
	assert.Equal(t, models.DownloadFormatWebm, status.Format)
	assert.Equal(t, 1080, status.MaxRes)
	assert.Equal(t, 120, status.MaxLength)
	assert.Equal(t, StatusQueued, status.Status)
	assert.True(t, status.QueuedAt.After(before) || status.QueuedAt.Equal(before))
}

func TestExecuteDownloadBuildArgs(t *testing.T) {
	// This is a unit test that would mock exec.Command
	// For now, we'll test the arg building logic separately
	// or create integration tests that actually run yt-dlp

	t.Skip("TODO: Mock exec.Command for testing executeDownload")
}

func TestProcessDownload(t *testing.T) {
	t.Skip("TODO: Test processDownload with mocked executeDownload")
}

func TestConcurrentQueueAccess(t *testing.T) {
	cfg := &models.Config{
		YtdlPath: "yt-dlp",
	}
	cacheDir := t.TempDir()
	cacheMgr := cache.NewManager(cacheDir, 0)

	dl := NewDownloader(cfg, cacheMgr, 2)
	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	// Queue from multiple goroutines
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			dl.Queue(
				fmt.Sprintf("TEST%d", n),
				fmt.Sprintf("https://youtube.com/watch?v=TEST%d", n),
				models.DownloadFormatMP4,
			)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All should be queued or active
	total := dl.GetQueueLength() + dl.GetActiveDownloads()
	assert.Equal(t, 10, total)
}
