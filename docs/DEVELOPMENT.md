# Development Guide

## Prerequisites

### Required

- **Go 1.22+**: [Download](https://go.dev/dl/)
- **Node.js 18+**: [Download](https://nodejs.org/)
- **Wails CLI v2.11+**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Windows-specific

- **WebView2 Runtime**: Usually pre-installed on Windows 11
- **gcc**: MinGW-w64 for CGO (optional for some dependencies)

### Optional

- **yt-dlp**: For testing downloads
- **ffmpeg**: For video processing
- **Git**: For version control

## Setup

### 1. Clone Repository

```bash
git clone https://github.com/yourusername/vrcvideocacher
cd vrcvideocacher
```

### 2. Install Dependencies

```bash
# Go dependencies
go mod download

# Frontend dependencies
cd frontend
npm install
cd ..
```

### 3. Build yt-dlp-stub

```bash
# Windows
go build -o resources/ytdlp-stub.exe ./cmd/ytdlp-stub

# Or use Makefile
make build-stub
```

### 4. Run Development Server

```bash
wails dev
```

This will:
- Start Go backend with hot-reload
- Start Vite dev server for frontend
- Open the application window

## Project Structure

```
vrcvideocacher/
├── cmd/
│   └── ytdlp-stub/          # Standalone stub executable
├── internal/                # Private application packages
│   ├── api/                 # HTTP server and handlers
│   ├── cache/               # Cache management
│   ├── config/              # Configuration
│   ├── downloader/          # Download queue
│   ├── patcher/             # VRChat patching
│   ├── updater/             # Auto-updates
│   └── platform/            # Platform-specific code
├── pkg/                     # Public packages
│   └── models/              # Shared data models
├── frontend/                # React + TypeScript UI
│   ├── src/
│   │   ├── components/      # Reusable components
│   │   ├── pages/           # Page components
│   │   ├── hooks/           # Custom hooks
│   │   └── lib/             # Utilities
│   └── wailsjs/             # Generated Wails bindings
├── resources/               # Embedded resources
├── docs/                    # Documentation
├── main.go                  # Application entry point
├── app.go                   # Wails app definition
└── wails.json               # Wails configuration
```

## Development Workflow

### TDD Cycle

1. **Write Test** (Red)
   ```bash
   # Create test file
   touch internal/config/config_test.go

   # Write failing test
   go test ./internal/config -v
   ```

2. **Implement** (Green)
   ```bash
   # Implement minimum code to pass
   go test ./internal/config -v
   ```

3. **Refactor** (Refactor)
   ```bash
   # Improve code while keeping tests green
   go test ./internal/config -v
   ```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/config -v

# With coverage
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Quality

```bash
# Format code
go fmt ./...

# Lint
golangci-lint run

# Vet
go vet ./...
```

### Frontend Development

```bash
cd frontend

# Start Vite dev server only
npm run dev

# Build for production
npm run build

# Lint
npm run lint
```

### Building

```bash
# Development build
wails build

# Production build
wails build -clean

# With specific platform
wails build -platform windows/amd64

# Custom build script
make build
```

## Configuration

### wails.json

```json
{
  "name": "vrcvideocacher",
  "outputfilename": "VRCVideoCacher",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "http://localhost:5173",
  "author": {
    "name": "Your Name",
    "email": "your@email.com"
  }
}
```

### Environment Variables

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Custom cache path
export CACHE_PATH=/path/to/cache

# Skip WebView2 check (dev only)
export WAILS_SKIP_WEBVIEW2_CHECK=1
```

## Debugging

### Go Backend

```bash
# Enable debug logs
go run -tags dev .

# With delve debugger
dlv debug
```

### Frontend

- Use browser DevTools (F12 in development)
- React DevTools extension
- Check console for errors

### Wails Bindings

```bash
# Regenerate bindings
wails dev -generate
```

## Common Issues

### Port Already in Use

```bash
# Change port in wails.json or kill process
netstat -ano | findstr :9696
taskkill /PID <pid> /F
```

### WebView2 Not Found

- Install WebView2 Runtime from Microsoft
- Or set `WAILS_SKIP_WEBVIEW2_CHECK=1` (dev only)

### Build Errors

```bash
# Clean build cache
wails build -clean
go clean -cache

# Reinstall dependencies
go mod tidy
cd frontend && npm install
```

### Import Cycle Detected

- Review package dependencies
- Use interfaces to break cycles
- Move shared types to `pkg/models`

## Git Workflow

### Branching

```bash
# Feature branch
git checkout -b feat/config-management

# Bug fix
git checkout -b fix/cache-eviction
```

### Committing

```bash
# Stage changes
git add .

# Commit with conventional format
git commit -m "feat(config): add JSON configuration support

- Implement Load/Save methods
- Add default value handling
- Add validation logic

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

### Before Pushing

```bash
# Run tests
go test ./...

# Check coverage
go test -cover ./... | grep -E "coverage:|FAIL"

# Format code
go fmt ./...
```

## Resources

- [Wails Documentation](https://wails.io/docs)
- [Go Documentation](https://go.dev/doc/)
- [React Documentation](https://react.dev/)
- [shadcn/ui Components](https://ui.shadcn.com/)
- [Go Testing Best Practices](https://go.dev/blog/examples)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
