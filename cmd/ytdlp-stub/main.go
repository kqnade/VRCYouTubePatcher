package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	serverURL       = "http://127.0.0.1:9696"
	ErrNoURL        = errors.New("no URL found in arguments")
	ErrServerError  = errors.New("server returned error")
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run executes the stub logic and returns exit code
func run(args []string) int {
	// Parse arguments
	videoURL, avPro, source, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	// Make request to local server
	response, err := makeRequest(videoURL, avPro, source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	// Output response
	fmt.Println(response)
	return 0
}

// parseArgs parses command line arguments
// Returns: url, avpro, source, error
func parseArgs(args []string) (string, bool, string, error) {
	var videoURL string
	avPro := true  // Default to avpro (webm)
	source := "vrchat"

	for _, arg := range args {
		// Check for protocol filter (indicates non-avpro)
		if strings.Contains(arg, "[protocol^=http]") {
			avPro = false
			continue
		}

		// Check for Resonite (-J flag for JSON output)
		if arg == "-J" {
			source = "resonite"
			continue
		}

		// Find URL (starts with http)
		if strings.HasPrefix(strings.ToLower(arg), "http") {
			videoURL = arg
			break
		}
	}

	if videoURL == "" {
		return "", false, "", ErrNoURL
	}

	return videoURL, avPro, source, nil
}

// makeRequest sends request to local server
func makeRequest(videoURL string, avPro bool, source string) (string, error) {
	// Build request URL
	reqURL := fmt.Sprintf("%s/api/getvideo?url=%s&avpro=%t&source=%s",
		serverURL,
		url.QueryEscape(videoURL),
		avPro,
		source,
	)

	// Make HTTP request
	resp, err := http.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("connection refused - is VRCVideoCacher running? %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %s", ErrServerError, string(body))
	}

	return string(body), nil
}
