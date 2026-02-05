package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCLI(t *testing.T) {
	cli := NewCLI("1.0.0")
	require.NotNil(t, cli)
	assert.Equal(t, "1.0.0", cli.version)
}

func TestParseCommand_NoArgs(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{})
	assert.Error(t, err)
	assert.Nil(t, cmd)
}

func TestParseCommand_Help(t *testing.T) {
	cli := NewCLI("1.0.0")

	testCases := []struct {
		name string
		args []string
	}{
		{"help flag", []string{"-h"}},
		{"help long", []string{"--help"}},
		{"help command", []string{"help"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := cli.ParseCommand(tc.args)
			require.NoError(t, err)
			assert.Equal(t, CommandHelp, cmd.Type)
		})
	}
}

func TestParseCommand_Version(t *testing.T) {
	cli := NewCLI("1.0.0")

	testCases := []struct {
		name string
		args []string
	}{
		{"version flag", []string{"-v"}},
		{"version long", []string{"--version"}},
		{"version command", []string{"version"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := cli.ParseCommand(tc.args)
			require.NoError(t, err)
			assert.Equal(t, CommandVersion, cmd.Type)
		})
	}
}

func TestParseCommand_Server(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"server"})
	require.NoError(t, err)
	assert.Equal(t, CommandServer, cmd.Type)
}

func TestParseCommand_ServerWithPort(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"server", "-port", "9000"})
	require.NoError(t, err)
	assert.Equal(t, CommandServer, cmd.Type)
	assert.Equal(t, 9000, cmd.Port)
}

func TestParseCommand_Patch(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"patch"})
	require.NoError(t, err)
	assert.Equal(t, CommandPatch, cmd.Type)
}

func TestParseCommand_PatchWithPath(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"patch", "-path", "/custom/path"})
	require.NoError(t, err)
	assert.Equal(t, CommandPatch, cmd.Type)
	assert.Equal(t, "/custom/path", cmd.Path)
}

func TestParseCommand_Unpatch(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"unpatch"})
	require.NoError(t, err)
	assert.Equal(t, CommandUnpatch, cmd.Type)
}

func TestParseCommand_Update(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"update"})
	require.NoError(t, err)
	assert.Equal(t, CommandUpdate, cmd.Type)
}

func TestParseCommand_UpdateCheck(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"update", "-check"})
	require.NoError(t, err)
	assert.Equal(t, CommandUpdate, cmd.Type)
	assert.True(t, cmd.CheckOnly)
}

func TestParseCommand_InvalidCommand(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"invalid"})
	assert.Error(t, err)
	assert.Nil(t, cmd)
}

func TestPrintHelp(t *testing.T) {
	cli := NewCLI("1.0.0")

	var buf bytes.Buffer
	cli.PrintHelp(&buf)

	output := buf.String()
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "server")
	assert.Contains(t, output, "patch")
	assert.Contains(t, output, "unpatch")
	assert.Contains(t, output, "update")
}

func TestPrintVersion(t *testing.T) {
	cli := NewCLI("1.2.3")

	var buf bytes.Buffer
	cli.PrintVersion(&buf)

	output := buf.String()
	assert.Contains(t, output, "1.2.3")
}

func TestCommand_String(t *testing.T) {
	testCases := []struct {
		cmdType  CommandType
		expected string
	}{
		{CommandHelp, "help"},
		{CommandVersion, "version"},
		{CommandServer, "server"},
		{CommandPatch, "patch"},
		{CommandUnpatch, "unpatch"},
		{CommandUpdate, "update"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			cmd := &Command{Type: tc.cmdType}
			result := cmd.String()
			assert.True(t, strings.Contains(result, tc.expected))
		})
	}
}

func TestCommand_StringWithDetails(t *testing.T) {
	testCases := []struct {
		name     string
		cmd      *Command
		contains string
	}{
		{"server with port", &Command{Type: CommandServer, Port: 9000}, "9000"},
		{"patch with path", &Command{Type: CommandPatch, Path: "/custom/path"}, "/custom/path"},
		{"unpatch with path", &Command{Type: CommandUnpatch, Path: "/custom/path"}, "/custom/path"},
		{"update check only", &Command{Type: CommandUpdate, CheckOnly: true}, "check"},
		{"unknown type", &Command{Type: CommandType(999)}, "unknown"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.cmd.String()
			assert.Contains(t, result, tc.contains)
		})
	}
}

func TestRun_Help(t *testing.T) {
	cli := NewCLI("1.0.0")

	exitCode := cli.Run([]string{"help"})
	assert.Equal(t, 0, exitCode)
}

func TestRun_Version(t *testing.T) {
	cli := NewCLI("1.0.0")

	exitCode := cli.Run([]string{"version"})
	assert.Equal(t, 0, exitCode)
}

func TestRun_NoArgs(t *testing.T) {
	cli := NewCLI("1.0.0")

	exitCode := cli.Run([]string{})
	assert.Equal(t, 1, exitCode)
}

func TestRun_InvalidCommand(t *testing.T) {
	cli := NewCLI("1.0.0")

	exitCode := cli.Run([]string{"invalid"})
	assert.Equal(t, 1, exitCode)
}

func TestRun_Server(t *testing.T) {
	cli := NewCLI("1.0.0")

	exitCode := cli.Run([]string{"server"})
	assert.Equal(t, 0, exitCode)
}

func TestParseCommand_ServerInvalidFlag(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"server", "-invalid"})
	assert.Error(t, err)
	assert.Nil(t, cmd)
}

func TestParseCommand_PatchInvalidFlag(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"patch", "-invalid"})
	assert.Error(t, err)
	assert.Nil(t, cmd)
}

func TestParseCommand_UnpatchInvalidFlag(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"unpatch", "-invalid"})
	assert.Error(t, err)
	assert.Nil(t, cmd)
}

func TestParseCommand_UpdateInvalidFlag(t *testing.T) {
	cli := NewCLI("1.0.0")

	cmd, err := cli.ParseCommand([]string{"update", "-invalid"})
	assert.Error(t, err)
	assert.Nil(t, cmd)
}
