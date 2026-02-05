package patcher

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrVRChatNotFound = errors.New("VRChat installation not found")
	ErrFileNotFound   = errors.New("file not found")
)

// Patcher handles VRChat/Resonite yt-dlp patching
type Patcher struct {
	stubData []byte
	stubHash string
}

// NewPatcher creates a new patcher
func NewPatcher(stubData []byte) *Patcher {
	return &Patcher{
		stubData: stubData,
		stubHash: computeHash(stubData),
	}
}

// DetectVRChatPath attempts to find VRChat Tools directory
func DetectVRChatPath() (string, error) {
	// Try common VRChat installation paths on Windows
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", ErrVRChatNotFound
	}

	// VRChat stores files in LocalLow
	localLow := filepath.Join(filepath.Dir(localAppData), "LocalLow")
	toolsPath := filepath.Join(localLow, "VRChat", "VRChat", "Tools")

	// Check if directory exists
	if _, err := os.Stat(toolsPath); os.IsNotExist(err) {
		return "", ErrVRChatNotFound
	}

	return toolsPath, nil
}

// PatchVRChat patches VRChat's yt-dlp.exe with stub
func (p *Patcher) PatchVRChat(toolsPath string) error {
	ytdlpPath := filepath.Join(toolsPath, "yt-dlp.exe")
	backupPath := filepath.Join(toolsPath, "yt-dlp.exe.bkp")

	// Check if already patched
	if patched, err := p.IsPatched(toolsPath); err == nil && patched {
		return nil // Already patched
	}

	// Check if yt-dlp.exe exists
	if _, err := os.Stat(ytdlpPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrFileNotFound, ytdlpPath)
	}

	// Backup original if backup doesn't exist
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		// Remove read-only attribute if present
		if err := makeWritable(ytdlpPath); err != nil {
			return fmt.Errorf("failed to make file writable: %w", err)
		}

		// Copy to backup
		originalData, err := os.ReadFile(ytdlpPath)
		if err != nil {
			return fmt.Errorf("failed to read original: %w", err)
		}

		if err := os.WriteFile(backupPath, originalData, 0644); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Remove old file
	if err := os.Remove(ytdlpPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove original: %w", err)
	}

	// Write stub
	if err := os.WriteFile(ytdlpPath, p.stubData, 0644); err != nil {
		return fmt.Errorf("failed to write stub: %w", err)
	}

	// Make read-only
	if err := makeReadOnly(ytdlpPath); err != nil {
		return fmt.Errorf("failed to make read-only: %w", err)
	}

	return nil
}

// UnpatchVRChat restores original yt-dlp.exe
func (p *Patcher) UnpatchVRChat(toolsPath string) error {
	ytdlpPath := filepath.Join(toolsPath, "yt-dlp.exe")
	backupPath := filepath.Join(toolsPath, "yt-dlp.exe.bkp")

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return nil // No backup, nothing to do
	}

	// Make writable if needed
	if _, err := os.Stat(ytdlpPath); err == nil {
		if err := makeWritable(ytdlpPath); err != nil {
			return fmt.Errorf("failed to make writable: %w", err)
		}

		// Remove stub
		if err := os.Remove(ytdlpPath); err != nil {
			return fmt.Errorf("failed to remove stub: %w", err)
		}
	}

	// Restore from backup
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	if err := os.WriteFile(ytdlpPath, backupData, 0644); err != nil {
		return fmt.Errorf("failed to restore original: %w", err)
	}

	// Make writable
	if err := makeWritable(ytdlpPath); err != nil {
		return fmt.Errorf("failed to make writable: %w", err)
	}

	// Remove backup
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to remove backup: %w", err)
	}

	return nil
}

// IsPatched checks if yt-dlp.exe is patched with stub
func (p *Patcher) IsPatched(toolsPath string) (bool, error) {
	ytdlpPath := filepath.Join(toolsPath, "yt-dlp.exe")

	// Read file
	data, err := os.ReadFile(ytdlpPath)
	if err != nil {
		return false, err
	}

	// Compare hash
	fileHash := computeHash(data)
	return fileHash == p.stubHash, nil
}

// computeHash computes SHA256 hash of data
func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// makeReadOnly makes file read-only
func makeReadOnly(path string) error {
	return os.Chmod(path, 0444)
}

// makeWritable makes file writable
func makeWritable(path string) error {
	return os.Chmod(path, 0644)
}
