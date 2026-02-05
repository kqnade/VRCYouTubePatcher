package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantURL     string
		wantAvPro   bool
		wantSource  string
		wantErr     bool
	}{
		{
			name:        "simple URL (default avpro)",
			args:        []string{"https://www.youtube.com/watch?v=VIDEO_ID"},
			wantURL:     "https://www.youtube.com/watch?v=VIDEO_ID",
			wantAvPro:   true,
			wantSource:  "vrchat",
			wantErr:     false,
		},
		{
			name:        "URL with protocol filter (no avpro)",
			args:        []string{"-f", "bv*[protocol^=http]", "https://example.com/video.mp4"},
			wantURL:     "https://example.com/video.mp4",
			wantAvPro:   false,
			wantSource:  "vrchat",
			wantErr:     false,
		},
		{
			name:        "URL without protocol filter (avpro)",
			args:        []string{"-f", "bv*[height<=1080]", "https://example.com/video.webm"},
			wantURL:     "https://example.com/video.webm",
			wantAvPro:   true,
			wantSource:  "vrchat",
			wantErr:     false,
		},
		{
			name:        "Resonite with -J flag",
			args:        []string{"-J", "https://example.com/video.mp4"},
			wantURL:     "https://example.com/video.mp4",
			wantAvPro:   true,
			wantSource:  "resonite",
			wantErr:     false,
		},
		{
			name:        "no URL",
			args:        []string{"-f", "format"},
			wantURL:     "",
			wantAvPro:   false,
			wantSource:  "vrchat",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, avpro, source, err := parseArgs(tt.args)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, url)
			assert.Equal(t, tt.wantAvPro, avpro)
			assert.Equal(t, tt.wantSource, source)
		})
	}
}

func TestMakeRequest(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		assert.Equal(t, "https://example.com/video.mp4", r.URL.Query().Get("url"))
		assert.Equal(t, "true", r.URL.Query().Get("avpro"))
		assert.Equal(t, "vrchat", r.URL.Query().Get("source"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("http://localhost:9696/cached_video.mp4"))
	}))
	defer server.Close()

	// Override server URL for testing
	oldServerURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldServerURL }()

	response, err := makeRequest("https://example.com/video.mp4", true, "vrchat")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9696/cached_video.mp4", response)
}

func TestMakeRequestError(t *testing.T) {
	// Use invalid server URL
	oldServerURL := serverURL
	serverURL = "http://localhost:1" // Invalid port
	defer func() { serverURL = oldServerURL }()

	_, err := makeRequest("https://example.com/video.mp4", true, "vrchat")
	require.Error(t, err)
}

func TestMakeRequest500Error(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
	}))
	defer server.Close()

	oldServerURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldServerURL }()

	_, err := makeRequest("https://example.com/video.mp4", true, "vrchat")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Server error")
}

func TestRunWithValidArgs(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("http://localhost:9696/test.mp4"))
	}))
	defer server.Close()

	oldServerURL := serverURL
	serverURL = server.URL
	defer func() { serverURL = oldServerURL }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := run([]string{"https://www.youtube.com/watch?v=TEST"})

	w.Close()
	os.Stdout = oldStdout

	assert.Equal(t, 0, exitCode)

	// Read captured output
	var buf [1024]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])
	assert.Contains(t, output, "http://localhost:9696/test.mp4")
}

func TestRunWithNoArgs(t *testing.T) {
	exitCode := run([]string{})
	assert.Equal(t, 1, exitCode)
}

func TestRunWithConnectionError(t *testing.T) {
	oldServerURL := serverURL
	serverURL = "http://localhost:1" // Invalid
	defer func() { serverURL = oldServerURL }()

	exitCode := run([]string{"https://example.com/video.mp4"})
	assert.Equal(t, 1, exitCode)
}
