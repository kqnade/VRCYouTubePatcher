# VRCVideoCacher Architecture

## Overview

VRCVideoCacher is a Go application with a Wails-based GUI that caches VRChat videos locally to improve loading performance and fix YouTube playback issues.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Wails GUI (React + TS)                   │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┐  │
│  │Dashboard │ Settings │  Cache   │   Logs   │  Tray    │  │
│  └────┬─────┴─────┬────┴────┬─────┴────┬─────┴──────────┘  │
│       │           │         │          │                    │
│       └───────────┴─────────┴──────────┘                    │
│                      │                                       │
│              Wails Bindings (app.go)                        │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────┴───────────────────────────────────┐
│                    Go Backend Modules                        │
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Config    │  │    Cache    │  │     API     │        │
│  │   Manager   │◄─┤   Manager   │◄─┤   Server    │        │
│  └─────────────┘  └─────────────┘  └──────┬──────┘        │
│                                            │                 │
│  ┌─────────────┐  ┌─────────────┐  ┌──────▼──────┐        │
│  │  Downloader │◄─┤   Patcher   │  │  Handlers   │        │
│  │    Queue    │  │  (VRChat)   │  │             │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
│                                                              │
│  ┌─────────────┐  ┌─────────────┐                          │
│  │   Updater   │  │  Platform   │                          │
│  │   (Tools)   │  │  (Windows)  │                          │
│  └─────────────┘  └─────────────┘                          │
└──────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌──────────────────────────────────────────────────────────────┐
│                     External Components                       │
│                                                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │  yt-dlp    │  │   ffmpeg   │  │    deno    │            │
│  │  (stub)    │  │            │  │            │            │
│  └────────────┘  └────────────┘  └────────────┘            │
│                                                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐            │
│  │  VRChat    │  │  Resonite  │  │  Browser   │            │
│  │  (Tools)   │  │  (Tools)   │  │  Extension │            │
│  └────────────┘  └────────────┘  └────────────┘            │
└──────────────────────────────────────────────────────────────┘
```

## Package Responsibilities

### `internal/config`
**Purpose**: Configuration management

- Load/save config.json
- Provide default values
- Validate configuration
- Notify on changes

**Key Types**:
- `Config`: Main configuration struct
- `Manager`: Singleton config manager

### `internal/cache`
**Purpose**: Cache directory management

- Scan cache directory
- Track cache entries (file size, last access)
- LRU-based eviction
- Size limit enforcement

**Key Types**:
- `Manager`: Cache manager with sync.Map
- `Entry`: Cache entry metadata

### `internal/api`
**Purpose**: HTTP server and API endpoints

- Serve cached files
- `/api/getvideo`: Resolve video URLs
- `/api/youtube-cookies`: Receive cookies
- `/api/cache/*`: Cache management endpoints

**Key Types**:
- `Server`: HTTP server
- `Handler`: Request handlers

### `internal/downloader`
**Purpose**: Background video downloading

- Download queue (channel-based)
- Execute yt-dlp processes
- Progress notification
- Support YouTube/PyPyDance/VRDancing

**Key Types**:
- `Queue`: Download queue manager
- `Task`: Download task

### `internal/patcher`
**Purpose**: Patch VRChat/Resonite yt-dlp

- Detect VRChat Tools directory
- Replace yt-dlp.exe with stub
- Restore on exit
- SHA256 hash verification

**Key Types**:
- `Patcher`: Patch manager
- `Target`: Patch target (VRChat/Resonite)

### `internal/updater`
**Purpose**: Auto-update yt-dlp/ffmpeg/deno

- Check GitHub releases
- Download latest versions
- Extract and install

**Key Types**:
- `Updater`: Update manager
- `Tool`: Updateable tool

### `internal/platform`
**Purpose**: Platform-specific operations

- Windows-specific path detection
- Linux compatibility (future)

### `pkg/models`
**Purpose**: Shared data models

- `VideoInfo`: Video metadata
- `Config`: Configuration structure
- `CacheEntry`: Cache entry

### `cmd/ytdlp-stub`
**Purpose**: VRChat yt-dlp replacement stub

- Parse arguments
- Forward requests to local server
- Return video URLs

## Data Flow

### Video Request Flow

```
VRChat → yt-dlp stub → HTTP Server → Cache Check
                                      ├─ Hit → Return cached file
                                      └─ Miss → Queue download → Return direct URL
```

### Download Flow

```
Queue → yt-dlp process → Download → Move to cache → Notify GUI
```

### Configuration Flow

```
GUI Settings → Config Manager → Save to JSON → Apply changes → Restart services
```

## Threading Model

- **Main thread**: Wails GUI event loop
- **HTTP server**: Go net/http (goroutines per request)
- **Download queue**: Single goroutine worker
- **Cache manager**: Thread-safe with sync.Map

## File Structure

```
AppData/VRCVideoCacher/
├── config.json           # User configuration
├── youtube_cookies.txt   # YouTube cookies
├── cache/                # Cached videos
│   ├── VIDEO_ID.mp4
│   └── VIDEO_ID.webm
└── utils/                # Downloaded tools
    ├── yt-dlp.exe
    ├── ffmpeg.exe
    └── deno.exe
```

## Security Considerations

1. **Local-only server**: Bind to 127.0.0.1 only
2. **Cookie protection**: Restrict file permissions
3. **Input validation**: Sanitize URLs and paths
4. **Hash verification**: Verify downloaded binaries
5. **No elevation**: Don't require admin privileges

## Performance Considerations

1. **Concurrent downloads**: Single worker to avoid rate limits
2. **Cache lookup**: O(1) with sync.Map
3. **LRU eviction**: Efficient with sorted access times
4. **Static file serving**: Direct file serving without copying
5. **Memory usage**: Stream large files, don't buffer

## Error Handling

- **Config errors**: Fall back to defaults
- **Network errors**: Retry with exponential backoff
- **Disk errors**: Notify user, continue operation
- **Process errors**: Log and notify GUI

## Logging

- **Library**: zerolog
- **Levels**: Debug, Info, Warn, Error
- **Output**: Console + GUI log viewer
- **Format**: JSON for parsing, pretty for development

## Testing Strategy

- **Unit tests**: Each package (80%+ coverage)
- **Integration tests**: API endpoints
- **E2E tests**: VRChat integration (manual)
- **Mocks**: HTTP responses, file system, processes

## Build & Deployment

- **Development**: `wails dev`
- **Production**: `wails build -platform windows/amd64`
- **Installer**: NSIS/WiX (TBD)
- **Updates**: GitHub Releases (TBD)
