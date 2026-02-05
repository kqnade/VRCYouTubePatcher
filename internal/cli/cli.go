package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// CommandType represents the type of CLI command
type CommandType int

const (
	CommandHelp CommandType = iota
	CommandVersion
	CommandServer
	CommandPatch
	CommandUnpatch
	CommandUpdate
)

// Command represents a parsed CLI command
type Command struct {
	Type      CommandType
	Port      int
	Path      string
	CheckOnly bool
}

// String returns a string representation of the command
func (c *Command) String() string {
	switch c.Type {
	case CommandHelp:
		return "help"
	case CommandVersion:
		return "version"
	case CommandServer:
		return fmt.Sprintf("server (port: %d)", c.Port)
	case CommandPatch:
		if c.Path != "" {
			return fmt.Sprintf("patch (path: %s)", c.Path)
		}
		return "patch"
	case CommandUnpatch:
		if c.Path != "" {
			return fmt.Sprintf("unpatch (path: %s)", c.Path)
		}
		return "unpatch"
	case CommandUpdate:
		if c.CheckOnly {
			return "update (check only)"
		}
		return "update"
	default:
		return "unknown"
	}
}

// CLI represents the command-line interface
type CLI struct {
	version string
}

// NewCLI creates a new CLI instance
func NewCLI(version string) *CLI {
	return &CLI{
		version: version,
	}
}

// ParseCommand parses command-line arguments and returns a Command
func (c *CLI) ParseCommand(args []string) (*Command, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command specified")
	}

	// Check for global flags first
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		return &Command{Type: CommandHelp}, nil
	}

	if args[0] == "-v" || args[0] == "--version" || args[0] == "version" {
		return &Command{Type: CommandVersion}, nil
	}

	// Parse subcommands
	switch args[0] {
	case "server":
		return c.parseServerCommand(args[1:])
	case "patch":
		return c.parsePatchCommand(args[1:])
	case "unpatch":
		return c.parseUnpatchCommand(args[1:])
	case "update":
		return c.parseUpdateCommand(args[1:])
	default:
		return nil, fmt.Errorf("unknown command: %s", args[0])
	}
}

// parseServerCommand parses the server command
func (c *CLI) parseServerCommand(args []string) (*Command, error) {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	port := fs.Int("port", 8080, "Server port")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &Command{
		Type: CommandServer,
		Port: *port,
	}, nil
}

// parsePatchCommand parses the patch command
func (c *CLI) parsePatchCommand(args []string) (*Command, error) {
	fs := flag.NewFlagSet("patch", flag.ContinueOnError)
	path := fs.String("path", "", "VRChat Tools directory path (auto-detect if empty)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &Command{
		Type: CommandPatch,
		Path: *path,
	}, nil
}

// parseUnpatchCommand parses the unpatch command
func (c *CLI) parseUnpatchCommand(args []string) (*Command, error) {
	fs := flag.NewFlagSet("unpatch", flag.ContinueOnError)
	path := fs.String("path", "", "VRChat Tools directory path (auto-detect if empty)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &Command{
		Type: CommandUnpatch,
		Path: *path,
	}, nil
}

// parseUpdateCommand parses the update command
func (c *CLI) parseUpdateCommand(args []string) (*Command, error) {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	checkOnly := fs.Bool("check", false, "Only check for updates without installing")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &Command{
		Type:      CommandUpdate,
		CheckOnly: *checkOnly,
	}, nil
}

// PrintHelp prints the help message
func (c *CLI) PrintHelp(w io.Writer) {
	help := `VRCYouTubePatcher - YouTube video cacher for VRChat

Usage:
  vrcvideocacher [command] [flags]

Available Commands:
  server      Start HTTP API server
  patch       Patch VRChat's yt-dlp.exe with stub
  unpatch     Restore original VRChat's yt-dlp.exe
  update      Update VRCYouTubePatcher to latest version
  version     Print version information
  help        Print this help message

Server Flags:
  -port int   Server port (default: 8080)

Patch/Unpatch Flags:
  -path string   VRChat Tools directory path (auto-detect if empty)

Update Flags:
  -check   Only check for updates without installing

Examples:
  vrcvideocacher server
  vrcvideocacher server -port 9000
  vrcvideocacher patch
  vrcvideocacher patch -path "C:\Users\...\VRChat\Tools"
  vrcvideocacher unpatch
  vrcvideocacher update
  vrcvideocacher update -check
  vrcvideocacher version
`
	fmt.Fprint(w, help)
}

// PrintVersion prints the version information
func (c *CLI) PrintVersion(w io.Writer) {
	fmt.Fprintf(w, "VRCYouTubePatcher version %s\n", c.version)
}

// Run executes the CLI with the given arguments
func (c *CLI) Run(args []string) int {
	cmd, err := c.ParseCommand(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		c.PrintHelp(os.Stderr)
		return 1
	}

	switch cmd.Type {
	case CommandHelp:
		c.PrintHelp(os.Stdout)
		return 0
	case CommandVersion:
		c.PrintVersion(os.Stdout)
		return 0
	default:
		// Other commands will be handled by the main function
		return 0
	}
}
