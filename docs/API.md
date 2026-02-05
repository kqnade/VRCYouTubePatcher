# API Specification

## HTTP API Endpoints

Base URL: `http://127.0.0.1:9696`

### GET /api/getvideo

Resolve video URL for VRChat/Resonite.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| url | string | Yes | Video URL to resolve |
| avpro | boolean | No | Use AVPro player (default: false) |
| source | string | No | Source application: `vrchat` or `resonite` (default: `vrchat`) |

**Response:**

- **200 OK**: Video URL (text/plain)
- **400 Bad Request**: Invalid parameters
- **500 Internal Server Error**: Processing error

**Examples:**

```bash
# YouTube video
curl "http://127.0.0.1:9696/api/getvideo?url=https://www.youtube.com/watch?v=VIDEO_ID"

# With AVPro
curl "http://127.0.0.1:9696/api/getvideo?url=https://www.youtube.com/watch?v=VIDEO_ID&avpro=true"

# From Resonite
curl "http://127.0.0.1:9696/api/getvideo?url=https://example.com/video.mp4&source=resonite"
```

**Response Examples:**

```
# Cached file
http://localhost:9696/VIDEO_ID.mp4

# Direct URL
https://manifest.googlevideo.com/...

# Empty (bypass)
(empty string)
```

### POST /api/youtube-cookies

Receive YouTube cookies from browser extension.

**Request Body:**

- Content-Type: text/plain
- Body: Netscape cookies.txt format

**Response:**

- **200 OK**: Cookies received
- **400 Bad Request**: Invalid cookies

**Example:**

```bash
curl -X POST http://127.0.0.1:9696/api/youtube-cookies \
  -H "Content-Type: text/plain" \
  --data-binary @cookies.txt
```

### GET /api/status

Get service status.

**Response:**

```json
{
  "running": true,
  "version": "1.0.0",
  "cacheSize": 1024000000,
  "cacheCount": 42,
  "downloadsActive": 1,
  "downloadsQueued": 3
}
```

### GET /api/cache/list

List cached videos.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| limit | int | No | Max results (default: 100) |
| offset | int | No | Offset for pagination (default: 0) |
| sort | string | No | Sort by: `date`, `size`, `name` (default: `date`) |

**Response:**

```json
{
  "total": 42,
  "items": [
    {
      "id": "VIDEO_ID",
      "filename": "VIDEO_ID.mp4",
      "size": 50000000,
      "lastAccess": "2026-02-05T12:00:00Z",
      "created": "2026-02-04T10:00:00Z"
    }
  ]
}
```

### DELETE /api/cache/{id}

Delete cached video by ID.

**Response:**

- **200 OK**: Deleted successfully
- **404 Not Found**: Video not found

**Example:**

```bash
curl -X DELETE http://127.0.0.1:9696/api/cache/VIDEO_ID
```

### GET /{filename}

Serve cached video file.

**Example:**

```bash
curl http://127.0.0.1:9696/VIDEO_ID.mp4
```

---

## Wails Bindings (Go ↔ Frontend)

### App Methods

#### GetConfig() *models.Config

Get current configuration.

**TypeScript:**

```typescript
import { GetConfig } from '../wailsjs/go/main/App'

const config = await GetConfig()
console.log(config.cacheYouTube)
```

#### SaveConfig(config: models.Config) error

Save configuration.

**TypeScript:**

```typescript
import { SaveConfig } from '../wailsjs/go/main/App'

await SaveConfig({
  ...config,
  cacheYouTube: true
})
```

#### StartServer() error

Start HTTP server.

**TypeScript:**

```typescript
import { StartServer } from '../wailsjs/go/main/App'

await StartServer()
```

#### StopServer() error

Stop HTTP server.

**TypeScript:**

```typescript
import { StopServer } from '../wailsjs/go/main/App'

await StopServer()
```

#### GetCacheList() []models.CacheEntry

Get list of cached videos.

**TypeScript:**

```typescript
import { GetCacheList } from '../wailsjs/go/main/App'

const entries = await GetCacheList()
console.log(entries.length)
```

#### DeleteCache(id: string) error

Delete cached video.

**TypeScript:**

```typescript
import { DeleteCache } from '../wailsjs/go/main/App'

await DeleteCache('VIDEO_ID')
```

#### ClearCache() error

Delete all cached videos.

**TypeScript:**

```typescript
import { ClearCache } from '../wailsjs/go/main/App'

await ClearCache()
```

#### GetLogs() []string

Get recent log entries.

**TypeScript:**

```typescript
import { GetLogs } from '../wailsjs/go/main/App'

const logs = await GetLogs()
console.log(logs.join('\n'))
```

#### PatchVRChat() error

Apply VRChat yt-dlp patch.

**TypeScript:**

```typescript
import { PatchVRChat } from '../wailsjs/go/main/App'

await PatchVRChat()
```

#### UnpatchVRChat() error

Remove VRChat yt-dlp patch.

**TypeScript:**

```typescript
import { UnpatchVRChat } from '../wailsjs/go/main/App'

await UnpatchVRChat()
```

### Events (Go → Frontend)

#### download:progress

Download progress update.

**Payload:**

```json
{
  "videoId": "VIDEO_ID",
  "progress": 45.5,
  "status": "downloading"
}
```

**TypeScript:**

```typescript
import { EventsOn } from '../wailsjs/runtime/runtime'

EventsOn('download:progress', (data) => {
  console.log(`${data.videoId}: ${data.progress}%`)
})
```

#### log:entry

New log entry.

**Payload:**

```json
{
  "level": "info",
  "message": "Server started",
  "timestamp": "2026-02-05T12:00:00Z"
}
```

**TypeScript:**

```typescript
import { EventsOn } from '../wailsjs/runtime/runtime'

EventsOn('log:entry', (entry) => {
  console.log(`[${entry.level}] ${entry.message}`)
})
```

#### cache:updated

Cache was updated (add/delete).

**TypeScript:**

```typescript
import { EventsOn } from '../wailsjs/runtime/runtime'

EventsOn('cache:updated', () => {
  // Refresh cache list
  refreshCacheList()
})
```

#### server:status

Server status changed.

**Payload:**

```json
{
  "running": true
}
```

**TypeScript:**

```typescript
import { EventsOn } from '../wailsjs/runtime/runtime'

EventsOn('server:status', (status) => {
  console.log(`Server ${status.running ? 'started' : 'stopped'}`)
})
```

---

## Error Responses

All API endpoints return errors in this format:

```json
{
  "error": "error message",
  "code": "ERROR_CODE"
}
```

**Error Codes:**

| Code | Description |
|------|-------------|
| `INVALID_URL` | URL is malformed or empty |
| `CACHE_FULL` | Cache size limit exceeded |
| `DOWNLOAD_FAILED` | Video download failed |
| `PATCH_FAILED` | VRChat patching failed |
| `CONFIG_INVALID` | Configuration validation failed |
| `SERVER_ERROR` | Internal server error |

---

## Rate Limiting

Currently no rate limiting is implemented (local server).

Future consideration: Limit download queue to 5 concurrent items.

---

## Authentication

None required (local server, 127.0.0.1 only).

---

## CORS

CORS is disabled (local server).

Frontend served from same origin.
