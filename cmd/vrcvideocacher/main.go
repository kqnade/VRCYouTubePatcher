package main

import (
	"fmt"
	"os"
	"path/filepath"

	"vrcvideocacher/internal/api"
	"vrcvideocacher/internal/cache"
	"vrcvideocacher/internal/cli"
	"vrcvideocacher/internal/config"
	"vrcvideocacher/internal/patcher"
	"vrcvideocacher/internal/updater"
	"vrcvideocacher/internal/ytdl"
)

const (
	Version      = "0.1.0"
	GitHubRepo   = "kqnade/VRCYouTubePatcher"
	StubDataSize = 1024 // Placeholder size
)

func main() {
	// Create CLI instance
	cliApp := cli.NewCLI(Version)

	// Parse command-line arguments
	if len(os.Args) < 2 {
		cliApp.PrintHelp(os.Stderr)
		os.Exit(1)
	}

	cmd, err := cliApp.ParseCommand(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		cliApp.PrintHelp(os.Stderr)
		os.Exit(1)
	}

	// Handle help and version commands
	if cmd.Type == cli.CommandHelp {
		cliApp.PrintHelp(os.Stdout)
		os.Exit(0)
	}

	if cmd.Type == cli.CommandVersion {
		cliApp.PrintVersion(os.Stdout)
		os.Exit(0)
	}

	// Execute command
	exitCode := executeCommand(cmd)
	os.Exit(exitCode)
}

func executeCommand(cmd *cli.Command) int {
	switch cmd.Type {
	case cli.CommandServer:
		return runServer(cmd.Port)
	case cli.CommandPatch:
		return runPatch(cmd.Path)
	case cli.CommandUnpatch:
		return runUnpatch(cmd.Path)
	case cli.CommandUpdate:
		return runUpdate(cmd.CheckOnly)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd.String())
		return 1
	}
}

func runServer(port int) int {
	fmt.Printf("Starting VRCYouTubePatcher server on port %d...\n", port)

	// Initialize configuration
	configPath := config.GetDefaultConfigPath()
	cfgMgr, err := config.NewManager(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return 1
	}

	// Get config and override port if specified
	cfg := cfgMgr.Get()
	if port != 8080 {
		cfg.WebServerPort = port
	}

	// Initialize yt-dlp manager
	utilsDir := filepath.Join(config.GetDataDir(), "Utils")
	ytdlManager := ytdl.NewManager(utilsDir)

	// Ensure yt-dlp is installed
	fmt.Println("Checking yt-dlp installation...")
	if err := ytdlManager.EnsureInstalled(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to install yt-dlp: %v\n", err)
	}

	// Initialize cache manager
	cacheDir := filepath.Join(config.GetDataDir(), "Cache")
	maxSize := float64(cfg.CacheMaxSizeGB) * 1024 * 1024 * 1024
	cacheMgr := cache.NewManager(cacheDir, maxSize)

	// Initialize API server (downloader is created inside)
	server := api.NewServer(cfg, cacheMgr)

	// Start server (downloader is started automatically)
	fmt.Printf("Server listening on :%d\n", cfg.WebServerPort)
	fmt.Println("Press Ctrl+C to stop")

	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		return 1
	}

	// Keep server running (Start returns immediately)
	select {}
}

func runPatch(toolsPath string) int {
	fmt.Println("Patching VRChat's yt-dlp.exe...")

	// Detect VRChat path if not provided
	if toolsPath == "" {
		detectedPath, err := patcher.DetectVRChatPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintln(os.Stderr, "Please specify the VRChat Tools directory with -path flag")
			return 1
		}
		toolsPath = detectedPath
		fmt.Printf("Detected VRChat Tools directory: %s\n", toolsPath)
	}

	// Load stub data
	stubData, err := loadStubData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading stub: %v\n", err)
		return 1
	}

	// Create patcher
	p := patcher.NewPatcher(stubData)

	// Check if already patched
	if patched, err := p.IsPatched(toolsPath); err == nil && patched {
		fmt.Println("Already patched!")
		return 0
	}

	// Patch
	if err := p.PatchVRChat(toolsPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error patching: %v\n", err)
		return 1
	}

	fmt.Println("Successfully patched VRChat's yt-dlp.exe")
	return 0
}

func runUnpatch(toolsPath string) int {
	fmt.Println("Unpatching VRChat's yt-dlp.exe...")

	// Detect VRChat path if not provided
	if toolsPath == "" {
		detectedPath, err := patcher.DetectVRChatPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintln(os.Stderr, "Please specify the VRChat Tools directory with -path flag")
			return 1
		}
		toolsPath = detectedPath
		fmt.Printf("Detected VRChat Tools directory: %s\n", toolsPath)
	}

	// Load stub data (for patcher instance)
	stubData, err := loadStubData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading stub: %v\n", err)
		return 1
	}

	// Create patcher
	p := patcher.NewPatcher(stubData)

	// Unpatch
	if err := p.UnpatchVRChat(toolsPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error unpatching: %v\n", err)
		return 1
	}

	fmt.Println("Successfully restored original yt-dlp.exe")
	return 0
}

func runUpdate(checkOnly bool) int {
	if checkOnly {
		fmt.Println("Checking for updates...")
	} else {
		fmt.Println("Updating VRCYouTubePatcher...")
	}

	// Create updater
	u := updater.NewUpdater(GitHubRepo, Version)

	// Check for updates
	latestVersion, hasUpdate, err := u.CheckForUpdate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		return 1
	}

	if !hasUpdate {
		fmt.Printf("Already up to date (version %s)\n", Version)
		return 0
	}

	fmt.Printf("Update available: %s -> %s\n", Version, latestVersion)

	if checkOnly {
		fmt.Println("Run 'vrcvideocacher update' to install the update")
		return 0
	}

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		return 1
	}

	// Download and install update
	if err := u.Download(exePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating: %v\n", err)
		return 1
	}

	fmt.Printf("Successfully updated to version %s\n", latestVersion)
	fmt.Println("Please restart the application")
	return 0
}

func loadStubData() ([]byte, error) {
	// Try to load stub from cmd/ytdlp-stub
	stubPath := "../../cmd/ytdlp-stub/ytdlp-stub.exe"
	data, err := os.ReadFile(stubPath)
	if err == nil {
		return data, nil
	}

	// If not found, create a placeholder stub
	// In production, this should be embedded in the binary
	fmt.Println("Warning: Using placeholder stub data")
	return make([]byte, StubDataSize), nil
}
