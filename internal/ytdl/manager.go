package ytdl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	ytdlpNightlyAPI = "https://api.github.com/repos/yt-dlp/yt-dlp-nightly-builds/releases/latest"
)

// Manager handles yt-dlp installation and updates
type Manager struct {
	utilsDir       string
	currentVersion string
	lastCheckTime  time.Time
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// NewManager creates a new yt-dlp manager
func NewManager(utilsDir string) *Manager {
	// Ensure utils directory exists
	os.MkdirAll(utilsDir, 0755)

	return &Manager{
		utilsDir: utilsDir,
	}
}

// GetYtdlpPath returns the path to yt-dlp executable
func (m *Manager) GetYtdlpPath() string {
	filename := detectPlatform()
	return filepath.Join(m.utilsDir, filename)
}

// IsInstalled checks if yt-dlp is installed
func (m *Manager) IsInstalled() bool {
	_, err := os.Stat(m.GetYtdlpPath())
	return err == nil
}

// GetCurrentVersion returns the currently installed version
func (m *Manager) GetCurrentVersion() string {
	return m.currentVersion
}

// CheckForUpdate checks if a newer version is available
func (m *Manager) CheckForUpdate() (string, bool, error) {
	// Get latest release from GitHub
	resp, err := http.Get(ytdlpNightlyAPI)
	if err != nil {
		return "", false, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", false, fmt.Errorf("failed to parse release info: %w", err)
	}

	m.lastCheckTime = time.Now()

	// If not installed, any version is an update
	if !m.IsInstalled() {
		return release.TagName, true, nil
	}

	// Compare versions
	if m.currentVersion == "" || m.currentVersion != release.TagName {
		return release.TagName, true, nil
	}

	return release.TagName, false, nil
}

// Download downloads and installs yt-dlp
func (m *Manager) Download() error {
	// Get latest release info
	resp, err := http.Get(ytdlpNightlyAPI)
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release info: %w", err)
	}

	// Find the correct asset for this platform
	platform := detectPlatform()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == platform {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no asset found for platform: %s", platform)
	}

	// Download the file
	fmt.Printf("Downloading yt-dlp %s...\n", release.TagName)
	resp, err = http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download yt-dlp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Write to file
	ytdlpPath := m.GetYtdlpPath()
	tmpPath := ytdlpPath + ".tmp"

	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to make executable: %w", err)
	}

	// Replace old file
	if m.IsInstalled() {
		if err := os.Remove(ytdlpPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to remove old file: %w", err)
		}
	}

	if err := os.Rename(tmpPath, ytdlpPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	// Update version
	m.currentVersion = release.TagName
	fmt.Printf("yt-dlp %s installed successfully\n", release.TagName)

	return nil
}

// EnsureInstalled ensures yt-dlp is installed, downloading if necessary
func (m *Manager) EnsureInstalled() error {
	if m.IsInstalled() {
		return nil
	}

	fmt.Println("yt-dlp not found, downloading...")
	return m.Download()
}

// AutoUpdate checks for and applies updates if available
func (m *Manager) AutoUpdate() error {
	latestVersion, hasUpdate, err := m.CheckForUpdate()
	if err != nil {
		return err
	}

	if !hasUpdate {
		fmt.Println("yt-dlp is up to date")
		return nil
	}

	fmt.Printf("Updating yt-dlp to %s...\n", latestVersion)
	return m.Download()
}

// detectPlatform returns the appropriate yt-dlp binary name for the current platform
func detectPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "yt-dlp.exe"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "yt-dlp_linux_aarch64"
		}
		return "yt-dlp_linux"
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "yt-dlp_macos_arm64"
		}
		return "yt-dlp_macos"
	default:
		return "yt-dlp"
	}
}
