package updater

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpdater(t *testing.T) {
	updater := NewUpdater("myuser/myrepo", "1.0.0")

	assert.NotNil(t, updater)
	assert.Equal(t, "myuser/myrepo", updater.repo)
	assert.Equal(t, "1.0.0", updater.currentVersion)
}

func TestGetCurrentVersion(t *testing.T) {
	updater := NewUpdater("myuser/myrepo", "1.2.3")

	version := updater.GetCurrentVersion()
	assert.Equal(t, "1.2.3", version)
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{
			name:     "newer version available",
			current:  "1.0.0",
			latest:   "1.1.0",
			expected: true,
		},
		{
			name:     "same version",
			current:  "1.0.0",
			latest:   "1.0.0",
			expected: false,
		},
		{
			name:     "current is newer",
			current:  "2.0.0",
			latest:   "1.9.9",
			expected: false,
		},
		{
			name:     "major version bump",
			current:  "1.0.0",
			latest:   "2.0.0",
			expected: true,
		},
		{
			name:     "minor version bump",
			current:  "1.0.0",
			latest:   "1.1.0",
			expected: true,
		},
		{
			name:     "patch version bump",
			current:  "1.0.0",
			latest:   "1.0.1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersions(tt.current, tt.latest)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    [3]int
	}{
		{
			name:    "standard version",
			version: "1.2.3",
			want:    [3]int{1, 2, 3},
		},
		{
			name:    "version with v prefix",
			version: "v2.0.0",
			want:    [3]int{2, 0, 0},
		},
		{
			name:    "single digit",
			version: "1.0.0",
			want:    [3]int{1, 0, 0},
		},
		{
			name:    "large numbers",
			version: "10.20.30",
			want:    [3]int{10, 20, 30},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersion(tt.version)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestBackupExecutable(t *testing.T) {
	// Create temp executable
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"
	err := os.WriteFile(exePath, []byte("original"), 0755)
	require.NoError(t, err)

	updater := NewUpdater("test/repo", "1.0.0")

	// Backup
	backupPath, err := updater.backupExecutable(exePath)
	require.NoError(t, err)
	defer os.Remove(backupPath)

	// Verify backup exists
	assert.FileExists(t, backupPath)

	// Verify backup content
	data, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, "original", string(data))
}

func TestRestoreBackup(t *testing.T) {
	tmpDir := t.TempDir()
	exePath := tmpDir + "/test.exe"
	backupPath := tmpDir + "/test.exe.bak"

	// Create backup
	err := os.WriteFile(backupPath, []byte("backup data"), 0755)
	require.NoError(t, err)

	updater := NewUpdater("test/repo", "1.0.0")

	// Restore
	err = updater.restoreBackup(exePath, backupPath)
	require.NoError(t, err)

	// Verify restored
	data, err := os.ReadFile(exePath)
	require.NoError(t, err)
	assert.Equal(t, "backup data", string(data))

	// Backup should be removed
	assert.NoFileExists(t, backupPath)
}
