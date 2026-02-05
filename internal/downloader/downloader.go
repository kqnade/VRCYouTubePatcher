package downloader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"vrcvideocacher/internal/cache"
	"vrcvideocacher/pkg/models"
)

var (
	ErrDownloadFailed  = errors.New("download failed")
	ErrAlreadyQueued   = errors.New("video already queued or downloading")
	ErrDownloaderStopped = errors.New("downloader is stopped")
)

// DownloadStatus represents the status of a download
type DownloadStatus int

const (
	StatusQueued DownloadStatus = iota
	StatusDownloading
	StatusCompleted
	StatusFailed
)

func (s DownloadStatus) String() string {
	switch s {
	case StatusQueued:
		return "queued"
	case StatusDownloading:
		return "downloading"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// DownloadRequest represents a download request
type DownloadRequest struct {
	VideoID    string
	VideoURL   string
	Format     models.DownloadFormat
	MaxRes     int
	MaxLength  int
	QueuedAt   time.Time
	StartedAt  time.Time
	FinishedAt time.Time
	Status     DownloadStatus
	Error      error
}

// Downloader manages video downloads
type Downloader struct {
	mu           sync.RWMutex
	config       *models.Config
	cache        *cache.Manager
	queue        []*DownloadRequest
	active       map[string]*DownloadRequest
	ctx          context.Context
	cancel       context.CancelFunc
	workerWg     sync.WaitGroup
	running      bool
	maxWorkers   int
}

// NewDownloader creates a new downloader
func NewDownloader(config *models.Config, cache *cache.Manager, maxWorkers int) *Downloader {
	if maxWorkers <= 0 {
		maxWorkers = 2
	}

	return &Downloader{
		config:     config,
		cache:      cache,
		queue:      make([]*DownloadRequest, 0),
		active:     make(map[string]*DownloadRequest),
		maxWorkers: maxWorkers,
	}
}

// Start starts the downloader workers
func (d *Downloader) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return nil
	}

	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.running = true

	// Start worker goroutines
	for i := 0; i < d.maxWorkers; i++ {
		d.workerWg.Add(1)
		go d.worker()
	}

	return nil
}

// Stop stops the downloader workers
func (d *Downloader) Stop() error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}

	d.cancel()
	d.running = false
	d.mu.Unlock()

	// Wait for workers to finish
	d.workerWg.Wait()

	return nil
}

// Queue adds a video to the download queue
func (d *Downloader) Queue(videoID, videoURL string, format models.DownloadFormat) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return ErrDownloaderStopped
	}

	// Check if already in queue or downloading
	if _, ok := d.active[videoID]; ok {
		return ErrAlreadyQueued
	}

	for _, req := range d.queue {
		if req.VideoID == videoID {
			return ErrAlreadyQueued
		}
	}

	// Check if already cached
	if _, err := d.cache.GetEntry(videoID); err == nil {
		return nil // Already cached
	}

	// Add to queue
	req := &DownloadRequest{
		VideoID:   videoID,
		VideoURL:  videoURL,
		Format:    format,
		MaxRes:    d.config.CacheYouTubeMaxRes,
		MaxLength: d.config.CacheYouTubeMaxLength,
		QueuedAt:  time.Now(),
		Status:    StatusQueued,
	}

	d.queue = append(d.queue, req)

	return nil
}

// GetStatus returns the status of a video download
func (d *Downloader) GetStatus(videoID string) (*DownloadRequest, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check active downloads
	if req, ok := d.active[videoID]; ok {
		reqCopy := *req
		return &reqCopy, nil
	}

	// Check queue
	for _, req := range d.queue {
		if req.VideoID == videoID {
			reqCopy := *req
			return &reqCopy, nil
		}
	}

	return nil, errors.New("video not found")
}

// worker processes download requests from the queue
func (d *Downloader) worker() {
	defer d.workerWg.Done()

	for {
		// Check if stopped
		select {
		case <-d.ctx.Done():
			return
		default:
		}

		// Get next request from queue
		req := d.dequeue()
		if req == nil {
			// No work, sleep a bit
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Process download
		d.processDownload(req)
	}
}

// dequeue removes and returns the next request from the queue
func (d *Downloader) dequeue() *DownloadRequest {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.queue) == 0 {
		return nil
	}

	req := d.queue[0]
	d.queue = d.queue[1:]

	// Mark as active
	d.active[req.VideoID] = req

	return req
}

// processDownload processes a download request
func (d *Downloader) processDownload(req *DownloadRequest) {
	defer func() {
		// Remove from active
		d.mu.Lock()
		delete(d.active, req.VideoID)
		d.mu.Unlock()
	}()

	// Update status
	req.Status = StatusDownloading
	req.StartedAt = time.Now()

	// Execute download
	err := d.executeDownload(req)
	req.FinishedAt = time.Now()

	if err != nil {
		req.Status = StatusFailed
		req.Error = err
		fmt.Printf("Download failed for %s: %v\n", req.VideoID, err)
		return
	}

	req.Status = StatusCompleted
	fmt.Printf("Download completed for %s\n", req.VideoID)
}

// executeDownload executes yt-dlp to download the video
func (d *Downloader) executeDownload(req *DownloadRequest) error {
	// Determine output filename
	ext := req.Format.String()
	outputTemplate := filepath.Join(d.cache.GetCachePath(), fmt.Sprintf("%s.%s", req.VideoID, ext))

	// Build yt-dlp command
	args := []string{
		"--no-playlist",
		"--no-warnings",
		"--no-check-certificate",
		"-o", outputTemplate,
	}

	// Add format selection
	if req.Format == models.DownloadFormatWebm {
		// AVPro: prefer webm VP8/VP9
		args = append(args, "-f", fmt.Sprintf("bestvideo[height<=%d][ext=webm]+bestaudio[ext=webm]/best[height<=%d][ext=webm]/best[height<=%d]", req.MaxRes, req.MaxRes, req.MaxRes))
	} else {
		// Non-AVPro: prefer mp4 H264
		args = append(args, "-f", fmt.Sprintf("bestvideo[height<=%d][ext=mp4]+bestaudio[ext=m4a]/best[height<=%d][ext=mp4]/best[height<=%d]", req.MaxRes, req.MaxRes, req.MaxRes))
	}

	// Add cookies if enabled
	if d.config.YtdlUseCookies {
		cookiesPath := filepath.Join(d.cache.GetCachePath(), "youtube_cookies.txt")
		if _, err := os.Stat(cookiesPath); err == nil {
			args = append(args, "--cookies", cookiesPath)
		}
	}

	// Add additional args
	if d.config.YtdlAdditionalArgs != "" {
		// TODO: Parse additional args properly
		args = append(args, d.config.YtdlAdditionalArgs)
	}

	// Add URL
	args = append(args, req.VideoURL)

	// Execute yt-dlp
	cmd := exec.CommandContext(d.ctx, d.config.YtdlPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrDownloadFailed, string(output))
	}

	// Add to cache
	filename := filepath.Base(outputTemplate)
	if err := d.cache.AddEntry(req.VideoID, filename); err != nil {
		return fmt.Errorf("failed to add to cache: %w", err)
	}

	return nil
}

// GetQueueLength returns the number of queued downloads
func (d *Downloader) GetQueueLength() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.queue)
}

// GetActiveDownloads returns the number of active downloads
func (d *Downloader) GetActiveDownloads() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.active)
}
