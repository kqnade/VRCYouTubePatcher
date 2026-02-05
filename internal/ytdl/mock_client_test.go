package ytdl

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// MockHTTPClient is a mock HTTP client for testing
type MockHTTPClient struct {
	GetFunc func(url string) (*http.Response, error)
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	if m.GetFunc != nil {
		return m.GetFunc(url)
	}
	return nil, nil
}

// NewMockReleaseResponse creates a mock GitHub release response
func NewMockReleaseResponse(tagName string, assetName string) *http.Response {
	release := GitHubRelease{
		TagName: tagName,
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{Name: assetName, BrowserDownloadURL: "http://example.com/" + assetName},
		},
	}

	body, _ := json.Marshal(release)

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

// NewMockBinaryResponse creates a mock binary download response
func NewMockBinaryResponse(data []byte) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(data)),
	}
}
