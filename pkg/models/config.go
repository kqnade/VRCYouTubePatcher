package models

// Config represents the application configuration
type Config struct {
	WebServerURL          string   `json:"webServerUrl"`
	WebServerPort         int      `json:"webServerPort"`
	YtdlPath              string   `json:"ytdlPath"`
	YtdlUseCookies        bool     `json:"ytdlUseCookies"`
	YtdlAutoUpdate        bool     `json:"ytdlAutoUpdate"`
	YtdlAdditionalArgs    string   `json:"ytdlAdditionalArgs"`
	YtdlDubLanguage       string   `json:"ytdlDubLanguage"`
	YtdlDelay             int      `json:"ytdlDelay"`
	CachePath             string   `json:"cachePath"`
	BlockedURLs           []string `json:"blockedUrls"`
	BlockRedirect         string   `json:"blockRedirect"`
	CacheYouTube          bool     `json:"cacheYouTube"`
	CacheYouTubeMaxRes    int      `json:"cacheYouTubeMaxRes"`
	CacheYouTubeMaxLength int      `json:"cacheYouTubeMaxLength"`
	CacheMaxSizeGB        float64  `json:"cacheMaxSizeGb"`
	CachePyPyDance        bool     `json:"cachePyPyDance"`
	CacheVRDancing        bool     `json:"cacheVRDancing"`
	PatchVRC              bool     `json:"patchVRC"`
	PatchResonite         bool     `json:"patchResonite"`
	ResonitePath          string   `json:"resonitePath"`
	AutoUpdate            bool     `json:"autoUpdate"`
	StartMinimized        bool     `json:"startMinimized"`
	MinimizeToTray        bool     `json:"minimizeToTray"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		WebServerURL:          "http://localhost:9696",
		WebServerPort:         9696,
		YtdlPath:              "Utils/yt-dlp.exe",
		YtdlUseCookies:        true,
		YtdlAutoUpdate:        true,
		YtdlAdditionalArgs:    "",
		YtdlDubLanguage:       "",
		YtdlDelay:             0,
		CachePath:             "",
		BlockedURLs:           []string{},
		BlockRedirect:         "",
		CacheYouTube:          false,
		CacheYouTubeMaxRes:    1080,
		CacheYouTubeMaxLength: 120,
		CacheMaxSizeGB:        0,
		CachePyPyDance:        false,
		CacheVRDancing:        false,
		PatchVRC:              true,
		PatchResonite:         false,
		ResonitePath:          "",
		AutoUpdate:            true,
		StartMinimized:        false,
		MinimizeToTray:        true,
	}
}
