package updater

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckForUpdate_HasUpdate tests checking for updates when update is available
func TestCheckForUpdate_HasUpdate(t *testing.T) {
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return NewMockReleaseResponse("v1.1.0", "VRCVideoCacher-windows-amd64.exe"), nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	version, hasUpdate, err := updater.CheckForUpdate()
	require.NoError(t, err)
	assert.True(t, hasUpdate)
	assert.Equal(t, "v1.1.0", version)
}

// TestCheckForUpdate_NoUpdate tests when already up to date
func TestCheckForUpdate_NoUpdate(t *testing.T) {
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return NewMockReleaseResponse("v1.0.0", "VRCVideoCacher-windows-amd64.exe"), nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	version, hasUpdate, err := updater.CheckForUpdate()
	require.NoError(t, err)
	assert.False(t, hasUpdate)
	assert.Equal(t, "v1.0.0", version)
}

// TestCheckForUpdate_HTTPError tests error handling
func TestCheckForUpdate_HTTPError(t *testing.T) {
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	_, _, err := updater.CheckForUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check for updates")
}

// TestCheckForUpdate_Non200Status tests handling of non-200 status
func TestCheckForUpdate_Non200Status(t *testing.T) {
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
			}, nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	_, _, err := updater.CheckForUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

// TestCheckForUpdate_InvalidJSON tests handling of invalid JSON
func TestCheckForUpdate_InvalidJSON(t *testing.T) {
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	_, _, err := updater.CheckForUpdate()
	assert.Error(t, err)
}

// TestDownload_Success tests successful download
func TestDownload_Success(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"

	// Create original executable
	err := os.WriteFile(exePath, []byte("old version"), 0755)
	require.NoError(t, err)

	callCount := 0
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call: get release info
				return NewMockReleaseResponse("v1.1.0", detectAssetName()), nil
			}
			// Second call: download binary
			return NewMockBinaryResponse([]byte("new version")), nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	err = updater.Download(exePath)
	require.NoError(t, err)

	// Verify file was updated
	data, err := os.ReadFile(exePath)
	require.NoError(t, err)
	assert.Equal(t, "new version", string(data))
}

// TestDownload_NoMatchingAsset tests error when no matching asset found
func TestDownload_NoMatchingAsset(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			// Return release with no matching asset
			return NewMockReleaseResponse("v1.1.0", "wrong-platform.exe"), nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	err := updater.Download(exePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no asset found for platform")
}

// TestDownload_HTTPError tests download with HTTP error
func TestDownload_HTTPError(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, fmt.Errorf("connection error")
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	err := updater.Download(exePath)
	assert.Error(t, err)
}

// TestDownload_DownloadFailed tests binary download failure
func TestDownload_DownloadFailed(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"

	// Create original executable
	err := os.WriteFile(exePath, []byte("old version"), 0755)
	require.NoError(t, err)

	callCount := 0
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call succeeds (release info)
				return NewMockReleaseResponse("v1.1.0", detectAssetName()), nil
			}
			// Second call fails (binary download)
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
			}, nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	err = updater.Download(exePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")

	// Original file should be restored
	data, err := os.ReadFile(exePath)
	require.NoError(t, err)
	assert.Equal(t, "old version", string(data))
}

// TestDetectAssetName tests platform detection
func TestDetectAssetName(t *testing.T) {
	asset := detectAssetName()

	// Verify it returns a valid asset name
	validAssets := []string{
		"VRCVideoCacher-windows-amd64.exe",
		"VRCVideoCacher-linux-amd64",
		"VRCVideoCacher-linux-arm64",
		"VRCVideoCacher-darwin-amd64",
		"VRCVideoCacher-darwin-arm64",
		"VRCVideoCacher",
	}

	assert.Contains(t, validAssets, asset)

	// Verify current platform matches expected
	switch runtime.GOOS {
	case "windows":
		assert.Equal(t, "VRCVideoCacher-windows-amd64.exe", asset)
	case "linux":
		if runtime.GOARCH == "arm64" {
			assert.Equal(t, "VRCVideoCacher-linux-arm64", asset)
		} else {
			assert.Equal(t, "VRCVideoCacher-linux-amd64", asset)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			assert.Equal(t, "VRCVideoCacher-darwin-arm64", asset)
		} else {
			assert.Equal(t, "VRCVideoCacher-darwin-amd64", asset)
		}
	default:
		assert.Equal(t, "VRCVideoCacher", asset)
	}
}

// TestVerifyChecksum_Success tests successful checksum verification
func TestVerifyChecksum_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.txt"

	data := []byte("test data")
	err := os.WriteFile(filePath, data, 0644)
	require.NoError(t, err)

	// Calculate expected checksum
	expectedChecksum := "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"

	updater := NewUpdater("test/repo", "1.0.0")

	err = updater.VerifyChecksum(filePath, expectedChecksum)
	assert.NoError(t, err)
}

// TestVerifyChecksum_Mismatch tests checksum mismatch
func TestVerifyChecksum_Mismatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := tmpDir + "/test.txt"

	data := []byte("test data")
	err := os.WriteFile(filePath, data, 0644)
	require.NoError(t, err)

	updater := NewUpdater("test/repo", "1.0.0")

	err = updater.VerifyChecksum(filePath, "wrongchecksum")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

// TestVerifyChecksum_FileNotFound tests missing file
func TestVerifyChecksum_FileNotFound(t *testing.T) {
	updater := NewUpdater("test/repo", "1.0.0")

	err := updater.VerifyChecksum("/nonexistent/file.txt", "somechecksum")
	assert.Error(t, err)
}

// TestDownload_BackupFailure tests backup creation failure
func TestDownload_BackupFailure(t *testing.T) {
	// Try to backup a non-existent file
	tmpDir := t.TempDir()
	exePath := tmpDir + "/nonexistent.exe"

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return NewMockReleaseResponse("v1.1.0", detectAssetName()), nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	err := updater.Download(exePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to backup executable")
}

// TestDownload_ReleaseInfoError tests error getting release info
func TestDownload_ReleaseInfoError(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"

	// Create original executable
	err := os.WriteFile(exePath, []byte("old version"), 0755)
	require.NoError(t, err)

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       http.NoBody,
			}, nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	err = updater.Download(exePath)
	assert.Error(t, err)
}

// TestRestoreBackup_ReadError tests restore with missing backup
func TestRestoreBackup_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"
	backupPath := tmpDir + "/nonexistent.bak"

	updater := NewUpdater("test/repo", "1.0.0")

	err := updater.restoreBackup(exePath, backupPath)
	assert.Error(t, err)
}

// TestBackupExecutable_ReadError tests backup with missing file
func TestBackupExecutable_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/nonexistent.exe"

	updater := NewUpdater("test/repo", "1.0.0")

	_, err := updater.backupExecutable(exePath)
	assert.Error(t, err)
}

// TestDownload_InvalidReleaseJSON tests invalid JSON in release response
func TestDownload_InvalidReleaseJSON(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"

	// Create original executable
	err := os.WriteFile(exePath, []byte("old version"), 0755)
	require.NoError(t, err)

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	err = updater.Download(exePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse release info")
}

// TestDownload_WriteFailed tests failure during file write
func TestDownload_WriteFailed(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"

	// Create original executable
	err := os.WriteFile(exePath, []byte("old version"), 0755)
	require.NoError(t, err)

	callCount := 0
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call: get release info
				return NewMockReleaseResponse("v1.1.0", detectAssetName()), nil
			}
			// Second call: return error reader for binary download
			return NewMockErrorBinaryResponse(), nil
		},
	}

	updater := NewUpdaterWithClient("myuser/myrepo", "v1.0.0", mockClient)

	err = updater.Download(exePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write update")

	// Original file should be restored
	data, err := os.ReadFile(exePath)
	require.NoError(t, err)
	assert.Equal(t, "old version", string(data))
}
