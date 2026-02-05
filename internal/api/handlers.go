package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"vrcvideocacher/pkg/models"
)

var (
	ErrNoURL           = errors.New("no URL provided")
	ErrInvalidCookies  = errors.New("invalid cookies")
	ErrVideoIDNotFound = errors.New("video ID not found")
)

// handleGetVideo handles the /api/getvideo endpoint
func (s *Server) handleGetVideo(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	videoURL := r.URL.Query().Get("url")
	avproStr := r.URL.Query().Get("avpro")
	source := r.URL.Query().Get("source")

	if videoURL == "" {
		http.Error(w, "No URL provided", http.StatusBadRequest)
		return
	}

	// Determine avpro (default true)
	avpro := true
	if avproStr == "false" {
		avpro = false
	}
	_ = avpro // Will be used for download queue

	// Default source
	if source == "" {
		source = "vrchat"
	}
	_ = source // Will be used for download queue

	// Check if it's a YouTube URL
	if !isYouTubeURL(videoURL) {
		// Non-YouTube URLs are bypassed (return empty)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(""))
		return
	}

	// Extract video ID
	videoID, err := extractYouTubeVideoID(videoURL)
	if err != nil {
		// If can't extract ID, bypass
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(""))
		return
	}

	// Try to find cached file
	cachedPath, err := s.cache.GetFilePath(videoID)
	if err == nil {
		// Cache hit - return cached URL
		filename := filepath.Base(cachedPath)
		cachedURL := fmt.Sprintf("%s/%s", s.config.WebServerURL, filename)

		// Update last access time
		s.cache.UpdateLastAccess(videoID)

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(cachedURL))
		return
	}

	// Cache miss - queue download
	format := models.DownloadFormatMP4
	if avpro {
		format = models.DownloadFormatWebm
	}

	if err := s.downloader.Queue(videoID, videoURL, format); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to queue download for %s: %v\n", videoID, err)
	}

	// Return empty (download will happen in background)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(""))
}

// handleYouTubeCookies handles the /api/youtube-cookies endpoint
func (s *Server) handleYouTubeCookies(w http.ResponseWriter, r *http.Request) {
	// Read cookies from body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	cookies := string(body)

	// Validate cookies
	if !validateCookies(cookies) {
		http.Error(w, "Invalid cookies", http.StatusBadRequest)
		return
	}

	// Save cookies to file
	cookiesPath := filepath.Join(s.config.CachePath, "youtube_cookies.txt")
	if err := s.saveCookies(cookiesPath, cookies); err != nil {
		http.Error(w, "Failed to save cookies", http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Cookies received",
	})
}

// extractYouTubeVideoID extracts video ID from YouTube URL
func extractYouTubeVideoID(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// Handle different YouTube URL formats
	host := parsedURL.Hostname()

	// youtu.be short links
	if host == "youtu.be" {
		// Path is /VIDEO_ID
		videoID := strings.TrimPrefix(parsedURL.Path, "/")
		if videoID != "" {
			return videoID, nil
		}
		return "", ErrVideoIDNotFound
	}

	// youtube.com URLs
	if strings.Contains(host, "youtube.com") {
		// Check for /watch?v=VIDEO_ID
		if parsedURL.Path == "/watch" {
			videoID := parsedURL.Query().Get("v")
			if videoID != "" {
				return videoID, nil
			}
		}

		// Check for /embed/VIDEO_ID
		if strings.HasPrefix(parsedURL.Path, "/embed/") {
			videoID := strings.TrimPrefix(parsedURL.Path, "/embed/")
			if videoID != "" {
				return videoID, nil
			}
		}

		// Check for /v/VIDEO_ID
		if strings.HasPrefix(parsedURL.Path, "/v/") {
			videoID := strings.TrimPrefix(parsedURL.Path, "/v/")
			if videoID != "" {
				return videoID, nil
			}
		}
	}

	return "", ErrVideoIDNotFound
}

// isYouTubeURL checks if URL is a YouTube URL
func isYouTubeURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	host := parsedURL.Hostname()
	return strings.Contains(host, "youtube.com") || host == "youtu.be"
}

// validateCookies validates YouTube cookies
func validateCookies(cookies string) bool {
	if cookies == "" {
		return false
	}

	// Check for youtube.com domain
	if !strings.Contains(cookies, "youtube.com") {
		return false
	}

	// Check for LOGIN_INFO cookie (indicates logged in)
	if !strings.Contains(cookies, "LOGIN_INFO") {
		return false
	}

	return true
}

// saveCookies saves cookies to file
func (s *Server) saveCookies(path string, cookies string) error {
	// Write cookies to file
	if err := os.WriteFile(path, []byte(cookies), 0644); err != nil {
		return fmt.Errorf("failed to write cookies file: %w", err)
	}

	return nil
}
