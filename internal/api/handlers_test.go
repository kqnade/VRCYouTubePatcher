package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vrcvideocacher/internal/cache"
	"vrcvideocacher/pkg/models"
)

func TestHandleGetVideo(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()

	server := NewServer(cfg, cacheMgr)

	tests := []struct {
		name           string
		url            string
		avpro          string
		source         string
		setupCache     func()
		wantStatusCode int
		wantContains   string
	}{
		{
			name:           "no URL parameter",
			url:            "",
			wantStatusCode: http.StatusBadRequest,
			wantContains:   "URL",
		},
		{
			name:           "cached video exists",
			url:            "https://www.youtube.com/watch?v=TEST123",
			avpro:          "false",
			source:         "vrchat",
			setupCache: func() {
				// Create cached file
				testFile := filepath.Join(tempDir, "TEST123.mp4")
				os.WriteFile(testFile, []byte("cached video"), 0644)
				cacheMgr.AddEntry("TEST123", "TEST123.mp4")
			},
			wantStatusCode: http.StatusOK,
			wantContains:   "TEST123.mp4",
		},
		{
			name:           "bypass for non-YouTube URL",
			url:            "https://example.com/video.mp4",
			wantStatusCode: http.StatusOK,
			wantContains:   "", // Empty response for bypass
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupCache != nil {
				tt.setupCache()
			}

			// Build query string
			query := ""
			if tt.url != "" {
				query = "url=" + tt.url
			}
			if tt.avpro != "" {
				if query != "" {
					query += "&"
				}
				query += "avpro=" + tt.avpro
			}
			if tt.source != "" {
				if query != "" {
					query += "&"
				}
				query += "source=" + tt.source
			}

			req := httptest.NewRequest("GET", "/api/getvideo?"+query, nil)
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantContains != "" {
				body, _ := io.ReadAll(w.Body)
				assert.Contains(t, string(body), tt.wantContains)
			}
		})
	}
}

func TestHandleYouTubeCookies(t *testing.T) {
	tempDir := t.TempDir()
	cacheMgr := cache.NewManager(tempDir, 0)
	cfg := models.DefaultConfig()

	server := NewServer(cfg, cacheMgr)

	tests := []struct {
		name           string
		body           string
		wantStatusCode int
		wantContains   string
	}{
		{
			name: "valid cookies",
			body: `# Netscape HTTP Cookie File
.youtube.com	TRUE	/	TRUE	0	LOGIN_INFO	test_cookie`,
			wantStatusCode: http.StatusOK,
			wantContains:   "received",
		},
		{
			name:           "invalid cookies",
			body:           "not a valid cookie",
			wantStatusCode: http.StatusBadRequest,
			wantContains:   "invalid",
		},
		{
			name:           "empty body",
			body:           "",
			wantStatusCode: http.StatusBadRequest,
			wantContains:   "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/youtube-cookies", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			body, _ := io.ReadAll(w.Body)
			assert.Contains(t, strings.ToLower(string(body)), tt.wantContains)
		})
	}
}

func TestExtractYouTubeVideoID(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "standard watch URL",
			url:  "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			want: "dQw4w9WgXcQ",
		},
		{
			name: "short URL",
			url:  "https://youtu.be/dQw4w9WgXcQ",
			want: "dQw4w9WgXcQ",
		},
		{
			name: "watch URL with additional params",
			url:  "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=10s",
			want: "dQw4w9WgXcQ",
		},
		{
			name: "embed URL",
			url:  "https://www.youtube.com/embed/dQw4w9WgXcQ",
			want: "dQw4w9WgXcQ",
		},
		{
			name:    "non-YouTube URL",
			url:     "https://example.com/video",
			want:    "",
			wantErr: true,
		},
		{
			name:    "invalid YouTube URL",
			url:     "https://www.youtube.com/",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractYouTubeVideoID(tt.url)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsYouTubeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"youtube.com", "https://www.youtube.com/watch?v=TEST", true},
		{"youtu.be", "https://youtu.be/TEST", true},
		{"m.youtube.com", "https://m.youtube.com/watch?v=TEST", true},
		{"other domain", "https://example.com/video", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isYouTubeURL(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateCookies(t *testing.T) {
	tests := []struct {
		name    string
		cookies string
		want    bool
	}{
		{
			name: "valid cookies",
			cookies: `.youtube.com	TRUE	/	TRUE	0	LOGIN_INFO	test
.youtube.com	TRUE	/	TRUE	0	VISITOR_INFO1_LIVE	test`,
			want: true,
		},
		{
			name:    "no youtube.com",
			cookies: `.example.com	TRUE	/	TRUE	0	COOKIE	test`,
			want:    false,
		},
		{
			name:    "no LOGIN_INFO",
			cookies: `.youtube.com	TRUE	/	TRUE	0	OTHER_COOKIE	test`,
			want:    false,
		},
		{
			name:    "empty",
			cookies: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateCookies(tt.cookies)
			assert.Equal(t, tt.want, got)
		})
	}
}
