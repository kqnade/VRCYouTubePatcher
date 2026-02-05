package ytdl

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckForUpdate_NotInstalled_HasUpdate tests checking for updates when not installed
func TestCheckForUpdate_NotInstalled_HasUpdate(t *testing.T) {
	utilsDir := t.TempDir()

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return NewMockReleaseResponse("2024.01.01", detectPlatform()), nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	version, hasUpdate, err := mgr.CheckForUpdate()
	require.NoError(t, err)
	assert.True(t, hasUpdate)
	assert.Equal(t, "2024.01.01", version)
}

// TestCheckForUpdate_AlreadyUpToDate tests when already up to date
func TestCheckForUpdate_AlreadyUpToDate(t *testing.T) {
	utilsDir := t.TempDir()

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return NewMockReleaseResponse("2024.01.01", detectPlatform()), nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)
	mgr.currentVersion = "2024.01.01"

	// Create fake installed file
	err := os.WriteFile(mgr.GetYtdlpPath(), []byte("test"), 0755)
	require.NoError(t, err)

	version, hasUpdate, err := mgr.CheckForUpdate()
	require.NoError(t, err)
	assert.False(t, hasUpdate)
	assert.Equal(t, "2024.01.01", version)
}

// TestCheckForUpdate_HTTPError tests error handling
func TestCheckForUpdate_HTTPError(t *testing.T) {
	utilsDir := t.TempDir()

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	_, _, err := mgr.CheckForUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check for updates")
}

// TestDownload_Success tests successful download
func TestDownload_Success(t *testing.T) {
	utilsDir := t.TempDir()

	callCount := 0
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call: get release info
				return NewMockReleaseResponse("2024.01.01", detectPlatform()), nil
			}
			// Second call: download binary
			return NewMockBinaryResponse([]byte("fake yt-dlp binary")), nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	err := mgr.Download()
	require.NoError(t, err)

	// Verify file was created
	assert.True(t, mgr.IsInstalled())

	// Verify version was set
	assert.Equal(t, "2024.01.01", mgr.GetCurrentVersion())
}

// TestDownload_NoMatchingAsset tests error when no matching asset found
func TestDownload_NoMatchingAsset(t *testing.T) {
	utilsDir := t.TempDir()

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			// Return release with no matching asset
			return NewMockReleaseResponse("2024.01.01", "wrong-platform.exe"), nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	err := mgr.Download()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no asset found for platform")
}

// TestAutoUpdate_HasUpdate tests AutoUpdate when update is available
func TestAutoUpdate_HasUpdate(t *testing.T) {
	utilsDir := t.TempDir()

	callCount := 0
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			callCount++
			if callCount <= 2 {
				// CheckForUpdate and Download first call
				return NewMockReleaseResponse("2024.02.01", detectPlatform()), nil
			}
			// Download binary
			return NewMockBinaryResponse([]byte("new version")), nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)
	mgr.currentVersion = "2024.01.01"

	// Create old version
	err := os.WriteFile(mgr.GetYtdlpPath(), []byte("old"), 0755)
	require.NoError(t, err)

	err = mgr.AutoUpdate()
	require.NoError(t, err)

	// Should have new version
	assert.Equal(t, "2024.02.01", mgr.GetCurrentVersion())
}

// TestAutoUpdate_NoUpdate tests AutoUpdate when already up to date
func TestAutoUpdate_NoUpdate(t *testing.T) {
	utilsDir := t.TempDir()

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return NewMockReleaseResponse("2024.01.01", detectPlatform()), nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)
	mgr.currentVersion = "2024.01.01"

	// Create file
	err := os.WriteFile(mgr.GetYtdlpPath(), []byte("current"), 0755)
	require.NoError(t, err)

	err = mgr.AutoUpdate()
	require.NoError(t, err)

	// Should still have same version
	assert.Equal(t, "2024.01.01", mgr.GetCurrentVersion())
}

// TestDetectPlatform_AllPlatforms tests platform detection logic
func TestDetectPlatform_AllPlatforms(t *testing.T) {
	// Test current platform
	platform := detectPlatform()

	// Verify it returns a valid platform string
	validPlatforms := []string{
		"yt-dlp.exe",           // Windows
		"yt-dlp_linux",         // Linux x86_64
		"yt-dlp_linux_aarch64", // Linux ARM64
		"yt-dlp_macos",         // macOS x86_64
		"yt-dlp_macos_arm64",   // macOS ARM64
		"yt-dlp",               // Fallback
	}

	assert.Contains(t, validPlatforms, platform)

	// Verify current platform matches expected
	switch runtime.GOOS {
	case "windows":
		assert.Equal(t, "yt-dlp.exe", platform)
	case "linux":
		if runtime.GOARCH == "arm64" {
			assert.Equal(t, "yt-dlp_linux_aarch64", platform)
		} else {
			assert.Equal(t, "yt-dlp_linux", platform)
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			assert.Equal(t, "yt-dlp_macos_arm64", platform)
		} else {
			assert.Equal(t, "yt-dlp_macos", platform)
		}
	default:
		// Other platforms should return "yt-dlp"
		assert.Equal(t, "yt-dlp", platform)
	}

	// Call multiple times to ensure consistency
	for i := 0; i < 5; i++ {
		assert.Equal(t, platform, detectPlatform())
	}
}

// TestDownload_ReplaceExisting tests replacing an existing file
func TestDownload_ReplaceExisting(t *testing.T) {
	utilsDir := t.TempDir()

	callCount := 0
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				return NewMockReleaseResponse("2024.02.01", detectPlatform()), nil
			}
			return NewMockBinaryResponse([]byte("new version")), nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	// Create old version
	err := os.WriteFile(mgr.GetYtdlpPath(), []byte("old version"), 0755)
	require.NoError(t, err)

	// Download new version
	err = mgr.Download()
	require.NoError(t, err)

	// Verify new version
	data, err := os.ReadFile(mgr.GetYtdlpPath())
	require.NoError(t, err)
	assert.Equal(t, "new version", string(data))
}

// TestCheckForUpdate_InvalidJSON tests handling of invalid JSON
func TestCheckForUpdate_InvalidJSON(t *testing.T) {
	utilsDir := t.TempDir()

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			}, nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	_, _, err := mgr.CheckForUpdate()
	assert.Error(t, err)
}

// TestCheckForUpdate_Non200Status tests handling of non-200 status
func TestCheckForUpdate_Non200Status(t *testing.T) {
	utilsDir := t.TempDir()

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
			}, nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	_, _, err := mgr.CheckForUpdate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

// TestDownload_HTTPError tests download with HTTP error
func TestDownload_HTTPError(t *testing.T) {
	utilsDir := t.TempDir()

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, fmt.Errorf("connection error")
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	err := mgr.Download()
	assert.Error(t, err)
}

// TestDownload_DownloadFailed tests binary download failure
func TestDownload_DownloadFailed(t *testing.T) {
	utilsDir := t.TempDir()

	callCount := 0
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call succeeds (release info)
				return NewMockReleaseResponse("2024.01.01", detectPlatform()), nil
			}
			// Second call fails (binary download)
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       http.NoBody,
			}, nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	err := mgr.Download()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

// TestAutoUpdate_CheckError tests AutoUpdate when check fails
func TestAutoUpdate_CheckError(t *testing.T) {
	utilsDir := t.TempDir()

	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	err := mgr.AutoUpdate()
	assert.Error(t, err)
}

// TestEnsureInstalled_NotInstalled tests installation when not present
func TestEnsureInstalled_NotInstalled(t *testing.T) {
	utilsDir := t.TempDir()

	callCount := 0
	mockClient := &MockHTTPClient{
		GetFunc: func(url string) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				return NewMockReleaseResponse("2024.01.01", detectPlatform()), nil
			}
			return NewMockBinaryResponse([]byte("binary data")), nil
		},
	}

	mgr := NewManagerWithClient(utilsDir, mockClient)

	// Should not be installed
	assert.False(t, mgr.IsInstalled())

	// Ensure installed - should download
	err := mgr.EnsureInstalled()
	require.NoError(t, err)

	// Should now be installed
	assert.True(t, mgr.IsInstalled())
}

// TestIsInstalled_EdgeCases tests edge cases for IsInstalled
func TestIsInstalled_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(string) error
		expected bool
	}{
		{
			name: "file exists",
			setup: func(dir string) error {
				mgr := NewManager(dir)
				return os.WriteFile(mgr.GetYtdlpPath(), []byte("test"), 0755)
			},
			expected: true,
		},
		{
			name: "directory instead of file",
			setup: func(dir string) error {
				mgr := NewManager(dir)
				return os.Mkdir(mgr.GetYtdlpPath(), 0755)
			},
			expected: true, // os.Stat succeeds for directories too
		},
		{
			name:     "file does not exist",
			setup:    func(dir string) error { return nil },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utilsDir := t.TempDir()
			mgr := NewManager(utilsDir)

			err := tt.setup(utilsDir)
			require.NoError(t, err)

			result := mgr.IsInstalled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetYtdlpPath_Consistency tests path consistency
func TestGetYtdlpPath_Consistency(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	// Call multiple times, should return same path
	path1 := mgr.GetYtdlpPath()
	path2 := mgr.GetYtdlpPath()
	path3 := mgr.GetYtdlpPath()

	assert.Equal(t, path1, path2)
	assert.Equal(t, path2, path3)
}

// TestNewManager_CreatesDirectory tests that NewManager creates utils dir
func TestNewManager_CreatesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	utilsDir := fmt.Sprintf("%s/nonexistent/utils", baseDir)

	// Directory should not exist yet
	_, err := os.Stat(utilsDir)
	assert.True(t, os.IsNotExist(err))

	// Create manager
	mgr := NewManager(utilsDir)

	// Directory should now exist
	info, err := os.Stat(mgr.utilsDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestGetCurrentVersion_InitiallyEmpty tests initial state
func TestGetCurrentVersion_InitiallyEmpty(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	version := mgr.GetCurrentVersion()
	assert.Equal(t, "", version, "Initial version should be empty")
}

// TestGetCurrentVersion_AfterSet tests version tracking
func TestGetCurrentVersion_AfterSet(t *testing.T) {
	utilsDir := t.TempDir()
	mgr := NewManager(utilsDir)

	// Manually set version (simulating what Download does)
	testVersion := "2024.12.31"
	mgr.currentVersion = testVersion

	version := mgr.GetCurrentVersion()
	assert.Equal(t, testVersion, version)
}
