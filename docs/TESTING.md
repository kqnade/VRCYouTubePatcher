# Testing Strategy

## Test-Driven Development (TDD)

We follow strict TDD practices:

1. **Red**: Write a failing test first
2. **Green**: Write minimal code to pass the test
3. **Refactor**: Improve code while keeping tests green

## Test Coverage Goals

- **Unit Tests**: 80%+ coverage for all packages
- **Integration Tests**: Critical paths (API endpoints, download flow)
- **E2E Tests**: Manual testing with VRChat

## Test Organization

### Unit Tests

Each package has corresponding `*_test.go` files:

```
internal/config/
├── config.go
└── config_test.go

internal/cache/
├── manager.go
└── manager_test.go
```

### Test File Structure

```go
package config

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
    // Table-driven test
    tests := []struct {
        name    string
        input   string
        want    *Config
        wantErr bool
    }{
        {
            name: "valid config",
            input: `{"cacheYouTube": true}`,
            want: &Config{CacheYouTube: true},
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := LoadConfig(tt.input)
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## Package-Specific Testing

### internal/config

**Test Cases:**
- Load valid JSON
- Load invalid JSON (should use defaults)
- Save configuration
- Default values
- Validation rules

**Example:**

```go
func TestConfigDefaults(t *testing.T) {
    cfg := NewConfig()
    assert.Equal(t, 9696, cfg.WebServerPort)
    assert.True(t, cfg.PatchVRC)
    assert.False(t, cfg.CacheYouTube)
}
```

### internal/cache

**Test Cases:**
- Add cache entry
- Get cache entry
- LRU eviction
- Size calculation
- Max size enforcement

**Mocking:**
- Use `afero` for file system mocking

```go
import "github.com/spf13/afero"

func TestCacheManager(t *testing.T) {
    fs := afero.NewMemMapFs()
    manager := NewManager(fs)
    // Test with in-memory filesystem
}
```

### internal/api

**Test Cases:**
- `/api/getvideo` valid URL
- `/api/getvideo` invalid URL
- `/api/getvideo` cached video
- `/api/youtube-cookies` valid cookies
- `/api/youtube-cookies` invalid cookies
- Static file serving

**Example:**

```go
func TestGetVideoHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/api/getvideo?url=https://youtube.com/watch?v=TEST", nil)
    w := httptest.NewRecorder()

    handler := NewHandler(mockCache, mockDownloader)
    handler.GetVideo(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
}
```

### internal/downloader

**Test Cases:**
- Queue download
- Process download
- yt-dlp execution (mocked)
- Progress reporting
- Error handling

**Mocking:**
- Mock `exec.Command` for yt-dlp

```go
func TestDownloadQueue(t *testing.T) {
    queue := NewQueue()
    task := &DownloadTask{VideoID: "TEST"}

    queue.Add(task)
    assert.Equal(t, 1, queue.Len())
}
```

### internal/patcher

**Test Cases:**
- Detect VRChat path
- Backup yt-dlp
- Replace with stub
- Restore backup
- Hash verification

**Mocking:**
- Mock file system operations

### cmd/ytdlp-stub

**Test Cases:**
- Parse arguments
- Detect URL
- Detect avpro flag
- HTTP request to server
- Error handling

## Integration Tests

Tag integration tests with `//go:build integration`:

```go
//go:build integration

package api

import "testing"

func TestFullAPIFlow(t *testing.T) {
    // Start real server
    // Make real HTTP requests
    // Verify cache behavior
}
```

Run with:

```bash
go test -tags=integration ./...
```

## Mocking Strategy

### Interface-Based Mocking

Define interfaces for dependencies:

```go
// internal/cache/manager.go
type Manager interface {
    Get(id string) (*Entry, error)
    Add(entry *Entry) error
    Delete(id string) error
}

// internal/cache/mock.go
type MockManager struct {
    entries map[string]*Entry
}

func (m *MockManager) Get(id string) (*Entry, error) {
    entry, ok := m.entries[id]
    if !ok {
        return nil, ErrNotFound
    }
    return entry, nil
}
```

### File System Mocking

Use `afero.Fs` interface:

```go
import "github.com/spf13/afero"

type CacheManager struct {
    fs afero.Fs
}

// In tests:
func TestWithMockFS(t *testing.T) {
    fs := afero.NewMemMapFs()
    manager := NewCacheManager(fs)
}
```

### HTTP Mocking

Use `httptest`:

```go
func TestHTTPClient(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("test response"))
    }))
    defer server.Close()

    // Test with server.URL
}
```

### Process Mocking

Mock `exec.Command`:

```go
var execCommand = exec.Command

func TestYtdlpExecution(t *testing.T) {
    execCommand = func(name string, args ...string) *exec.Cmd {
        return exec.Command("echo", "mocked output")
    }
    defer func() { execCommand = exec.Command }()

    // Test code that uses execCommand
}
```

## Test Utilities

Create test helpers in `internal/testutil`:

```go
// internal/testutil/config.go
func NewTestConfig() *config.Config {
    return &config.Config{
        WebServerPort: 9696,
        CachePath: "/tmp/test-cache",
    }
}

// internal/testutil/cache.go
func CreateTestCache(t *testing.T) string {
    dir := t.TempDir()
    // Setup test cache
    return dir
}
```

## Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Detailed coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific package
go test ./internal/config -v

# Run specific test
go test ./internal/config -run TestLoadConfig

# With race detector
go test -race ./...

# Integration tests
go test -tags=integration ./...
```

## Continuous Testing

Use `wails dev` for automatic test runs during development.

Or use `watch` tools:

```bash
# Install gotestsum
go install gotest.tools/gotestsum@latest

# Watch and run tests
gotestsum --watch
```

## Benchmarks

Write benchmarks for performance-critical code:

```go
func BenchmarkCacheLookup(b *testing.B) {
    manager := setupBenchCache()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        manager.Get("TEST_ID")
    }
}
```

Run with:

```bash
go test -bench=. ./...
go test -bench=. -benchmem ./...
```

## Code Coverage Thresholds

We aim for:

- **80%+**: Required for merge
- **90%+**: Excellent
- **100%**: Not required, but nice to have

Check coverage:

```bash
go test -cover ./... | grep -E "coverage:|FAIL"
```

## Test Data

Store test fixtures in `testdata/`:

```
internal/config/testdata/
├── valid-config.json
├── invalid-config.json
└── empty-config.json
```

Load in tests:

```go
func TestLoadFromFile(t *testing.T) {
    data, err := os.ReadFile("testdata/valid-config.json")
    require.NoError(t, err)

    cfg, err := LoadConfig(string(data))
    require.NoError(t, err)
}
```

## Common Patterns

### Table-Driven Tests

```go
tests := []struct {
    name string
    input interface{}
    want interface{}
    wantErr bool
}{
    {"case 1", input1, want1, false},
    {"case 2", input2, want2, true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test logic
    })
}
```

### Subtests

```go
t.Run("group", func(t *testing.T) {
    t.Run("subtest 1", func(t *testing.T) {
        // Test 1
    })
    t.Run("subtest 2", func(t *testing.T) {
        // Test 2
    })
})
```

### Test Helpers

```go
func assertEqual(t *testing.T, got, want interface{}) {
    t.Helper()
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}
```

## Frontend Testing

### Unit Tests (Vitest)

```typescript
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import Dashboard from './Dashboard'

describe('Dashboard', () => {
  it('renders server status', () => {
    render(<Dashboard />)
    expect(screen.getByText(/server status/i)).toBeInTheDocument()
  })
})
```

### Component Tests

```bash
cd frontend
npm run test
npm run test:coverage
```

## E2E Testing Checklist

Manual testing with VRChat:

- [ ] Start VRCVideoCacher
- [ ] Patch VRChat
- [ ] Launch VRChat
- [ ] Join world with YouTube video
- [ ] Verify video plays (first time - downloads)
- [ ] Rejoin world
- [ ] Verify video plays from cache (faster)
- [ ] Check cache in GUI
- [ ] Delete cache entry
- [ ] Verify re-download
- [ ] Stop VRCVideoCacher
- [ ] Verify VRChat yt-dlp restored

## Resources

- [Go Testing](https://go.dev/doc/tutorial/add-a-test)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testify](https://github.com/stretchr/testify)
- [httptest](https://pkg.go.dev/net/http/httptest)
- [Afero](https://github.com/spf13/afero)
- [Vitest](https://vitest.dev/)
- [React Testing Library](https://testing-library.com/react)
