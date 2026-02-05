package downloader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vrcvideocacher/internal/cache"
	"vrcvideocacher/pkg/models"
)

// TestExecuteDownloadWithCookies tests download with cookies enabled
func TestExecuteDownloadWithCookies(t *testing.T) {
	cacheDir := t.TempDir()

	// Create cookies file
	cookiesPath := filepath.Join(cacheDir, "youtube_cookies.txt")
	err := os.WriteFile(cookiesPath, []byte("# Netscape HTTP Cookie File"), 0644)
	require.NoError(t, err)

	cfg := &models.Config{
		YtdlPath:              "echo", // Use echo as fake yt-dlp
		CacheYouTubeMaxRes:    1080,
		CacheYouTubeMaxLength: 120,
		YtdlUseCookies:        true,
		CachePath:             cacheDir,
	}

	cacheMgr := cache.NewManager(cacheDir, 0)
	dl := NewDownloader(cfg, cacheMgr, 1)

	// Start to initialize context
	err = dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	req := &DownloadRequest{
		VideoID:   "TEST",
		VideoURL:  "https://youtube.com/watch?v=TEST",
		Format:    models.DownloadFormatMP4,
		MaxRes:    1080,
		MaxLength: 120,
	}

	// Don't create the file beforehand - echo won't create it
	// so executeDownload should fail to find the file

	// Execute download - should fail because echo doesn't create actual file
	err = dl.executeDownload(req)

	// Should succeed because we have file detection logic,
	// but the file won't actually be there so it should error
	assert.Error(t, err, "Should fail when no file is created")
}

// TestExecuteDownloadWithAdditionalArgs tests additional arguments
func TestExecuteDownloadWithAdditionalArgs(t *testing.T) {
	cacheDir := t.TempDir()

	cfg := &models.Config{
		YtdlPath:              "echo",
		CacheYouTubeMaxRes:    720,
		CacheYouTubeMaxLength: 300,
		YtdlAdditionalArgs:    "--proxy http://proxy:8080",
		CachePath:             cacheDir,
	}

	cacheMgr := cache.NewManager(cacheDir, 0)
	dl := NewDownloader(cfg, cacheMgr, 1)

	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	req := &DownloadRequest{
		VideoID:   "TEST2",
		VideoURL:  "https://youtube.com/watch?v=TEST2",
		Format:    models.DownloadFormatWebm,
		MaxRes:    720,
		MaxLength: 300,
	}

	// Don't create file - let echo fail to create it
	err = dl.executeDownload(req)
	assert.Error(t, err, "Should fail when no file is created")
}

// TestExecuteDownloadFileDetection tests various file detection scenarios
func TestExecuteDownloadFileDetection(t *testing.T) {
	tests := []struct {
		name           string
		videoID        string
		format         models.DownloadFormat
		createFiles    []string
		expectSuccess  bool
		expectedFile   string
	}{
		{
			name:          "exact match",
			videoID:       "VIDEO1",
			format:        models.DownloadFormatMP4,
			createFiles:   []string{"VIDEO1.mp4"},
			expectSuccess: true,
			expectedFile:  "VIDEO1.mp4",
		},
		{
			name:          "with format code",
			videoID:       "VIDEO2",
			format:        models.DownloadFormatMP4,
			createFiles:   []string{"VIDEO2.f137.mp4"},
			expectSuccess: true,
			expectedFile:  "VIDEO2.f137.mp4",
		},
		{
			name:          "multiple files prefer exact extension",
			videoID:       "VIDEO3",
			format:        models.DownloadFormatMP4,
			createFiles:   []string{"VIDEO3.f140.m4a", "VIDEO3.f395.mp4"},
			expectSuccess: true,
			expectedFile:  "VIDEO3.f395.mp4",
		},
		{
			name:          "webm format",
			videoID:       "VIDEO4",
			format:        models.DownloadFormatWebm,
			createFiles:   []string{"VIDEO4.webm"},
			expectSuccess: true,
			expectedFile:  "VIDEO4.webm",
		},
		{
			name:          "no matching file",
			videoID:       "VIDEO5",
			format:        models.DownloadFormatMP4,
			createFiles:   []string{},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheDir := t.TempDir()

			cfg := &models.Config{
				YtdlPath:  "echo",
				CachePath: cacheDir,
			}

			cacheMgr := cache.NewManager(cacheDir, 0)
			dl := NewDownloader(cfg, cacheMgr, 1)

			startErr := dl.Start()
			require.NoError(t, startErr)
			defer dl.Stop()

			req := &DownloadRequest{
				VideoID:  tt.videoID,
				VideoURL: "https://youtube.com/watch?v=" + tt.videoID,
				Format:   tt.format,
				MaxRes:   1080,
			}

			// Create test files
			for _, filename := range tt.createFiles {
				filePath := filepath.Join(cacheDir, filename)
				fileErr := os.WriteFile(filePath, []byte("test"), 0644)
				require.NoError(t, fileErr)
			}

			err := dl.executeDownload(req)

			if tt.expectSuccess {
				// Should succeed in adding to cache
				require.NoError(t, err)

				// Verify correct file was added
				entry, err := cacheMgr.GetEntry(tt.videoID)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedFile, entry.FileName)
			} else {
				// Should fail
				assert.Error(t, err)
			}
		})
	}
}

// TestProcessDownloadSuccess tests successful download processing
func TestProcessDownloadSuccess(t *testing.T) {
	cacheDir := t.TempDir()

	cfg := &models.Config{
		YtdlPath:  "echo",
		CachePath: cacheDir,
	}

	cacheMgr := cache.NewManager(cacheDir, 0)
	dl := NewDownloader(cfg, cacheMgr, 1)

	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	req := &DownloadRequest{
		VideoID:  "SUCCESS",
		VideoURL: "https://youtube.com/watch?v=SUCCESS",
		Format:   models.DownloadFormatMP4,
		MaxRes:   1080,
	}

	// Create fake file
	testFile := filepath.Join(cacheDir, "SUCCESS.mp4")
	err = os.WriteFile(testFile, []byte("video"), 0644)
	require.NoError(t, err)

	// Process download
	dl.processDownload(req)

	// Verify completion
	assert.Equal(t, StatusCompleted, req.Status)
	assert.Nil(t, req.Error)
	assert.False(t, req.FinishedAt.IsZero())
}

// TestProcessDownloadFailure tests failed download processing
func TestProcessDownloadFailure(t *testing.T) {
	cacheDir := t.TempDir()

	cfg := &models.Config{
		YtdlPath:  "nonexistent-command",
		CachePath: cacheDir,
	}

	cacheMgr := cache.NewManager(cacheDir, 0)
	dl := NewDownloader(cfg, cacheMgr, 1)

	err := dl.Start()
	require.NoError(t, err)
	defer dl.Stop()

	req := &DownloadRequest{
		VideoID:  "FAIL",
		VideoURL: "https://youtube.com/watch?v=FAIL",
		Format:   models.DownloadFormatMP4,
		MaxRes:   1080,
	}

	// Process download (will fail)
	dl.processDownload(req)

	// Verify failure
	assert.Equal(t, StatusFailed, req.Status)
	assert.NotNil(t, req.Error)
	assert.False(t, req.FinishedAt.IsZero())
}

// TestFormatString tests DownloadFormat.String()
func TestFormatString(t *testing.T) {
	tests := []struct {
		format models.DownloadFormat
		want   string
	}{
		{models.DownloadFormatMP4, "mp4"},
		{models.DownloadFormatWebm, "webm"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.format.String())
		})
	}
}
