package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	checkTimeout = 30 * time.Second
)

// HTTPClient interface for mocking
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

// Updater handles application updates
type Updater struct {
	repo           string
	currentVersion string
	httpClient     HTTPClient
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
	Body string `json:"body"`
}

// NewUpdater creates a new updater
func NewUpdater(repo, currentVersion string) *Updater {
	return &Updater{
		repo:           repo,
		currentVersion: currentVersion,
		httpClient:     &http.Client{Timeout: checkTimeout},
	}
}

// NewUpdaterWithClient creates an updater with custom HTTP client
func NewUpdaterWithClient(repo, currentVersion string, client HTTPClient) *Updater {
	return &Updater{
		repo:           repo,
		currentVersion: currentVersion,
		httpClient:     client,
	}
}

// GetCurrentVersion returns the current version
func (u *Updater) GetCurrentVersion() string {
	return u.currentVersion
}

// CheckForUpdate checks if a new version is available
func (u *Updater) CheckForUpdate() (string, bool, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.repo)

	resp, err := u.httpClient.Get(apiURL)
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

	// Compare versions
	hasUpdate := compareVersions(u.currentVersion, release.TagName)

	return release.TagName, hasUpdate, nil
}

// Download downloads and applies the update
func (u *Updater) Download(exePath string) error {
	// Get latest release info
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.repo)

	resp, err := u.httpClient.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release info: %w", err)
	}

	// Find the correct asset for this platform
	assetName := detectAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no asset found for platform: %s", assetName)
	}

	// Backup current executable
	backupPath, err := u.backupExecutable(exePath)
	if err != nil {
		return fmt.Errorf("failed to backup executable: %w", err)
	}

	// Download new version
	fmt.Printf("Downloading update %s...\n", release.TagName)
	resp, err = u.httpClient.Get(downloadURL)
	if err != nil {
		u.restoreBackup(exePath, backupPath)
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		u.restoreBackup(exePath, backupPath)
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Write to temporary file
	tmpPath := exePath + ".new"
	out, err := os.Create(tmpPath)
	if err != nil {
		u.restoreBackup(exePath, backupPath)
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.Remove(tmpPath)
		u.restoreBackup(exePath, backupPath)
		return fmt.Errorf("failed to write update: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		u.restoreBackup(exePath, backupPath)
		return fmt.Errorf("failed to make executable: %w", err)
	}

	// Replace old executable
	if err := os.Remove(exePath); err != nil {
		os.Remove(tmpPath)
		u.restoreBackup(exePath, backupPath)
		return fmt.Errorf("failed to remove old executable: %w", err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		u.restoreBackup(exePath, backupPath)
		return fmt.Errorf("failed to rename new executable: %w", err)
	}

	// Remove backup on success
	os.Remove(backupPath)

	fmt.Printf("Update to %s completed successfully\n", release.TagName)
	return nil
}

// backupExecutable creates a backup of the current executable
func (u *Updater) backupExecutable(exePath string) (string, error) {
	backupPath := exePath + ".bak"

	data, err := os.ReadFile(exePath)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(backupPath, data, 0755); err != nil {
		return "", err
	}

	return backupPath, nil
}

// restoreBackup restores from backup
func (u *Updater) restoreBackup(exePath, backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(exePath, data, 0755); err != nil {
		return err
	}

	os.Remove(backupPath)
	return nil
}

// VerifyChecksum verifies the checksum of a file
func (u *Updater) VerifyChecksum(filePath, expectedChecksum string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(data)
	actualChecksum := hex.EncodeToString(hash[:])

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// compareVersions returns true if latest > current
func compareVersions(current, latest string) bool {
	currentParts := parseVersion(current)
	latestParts := parseVersion(latest)

	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}

	return false
}

// parseVersion parses a version string into [major, minor, patch]
func parseVersion(version string) [3]int {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	result := [3]int{0, 0, 0}

	for i := 0; i < len(parts) && i < 3; i++ {
		if num, err := strconv.Atoi(parts[i]); err == nil {
			result[i] = num
		}
	}

	return result
}

// detectAssetName returns the appropriate asset name for the current platform
func detectAssetName() string {
	switch runtime.GOOS {
	case "windows":
		return "VRCVideoCacher-windows-amd64.exe"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "VRCVideoCacher-linux-arm64"
		}
		return "VRCVideoCacher-linux-amd64"
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "VRCVideoCacher-darwin-arm64"
		}
		return "VRCVideoCacher-darwin-amd64"
	default:
		return "VRCVideoCacher"
	}
}
