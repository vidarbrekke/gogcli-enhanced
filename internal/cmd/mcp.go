package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"

	"github.com/steipete/gogcli/internal/mcp"
)

type MCPCmd struct {
	Serve MCPServeCmd `cmd:"" name:"serve" help:"Run MCP server over stdio"`
}

type MCPServeCmd struct{}

func (c *MCPServeCmd) Run(ctx context.Context) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// Execute tool calls in a subprocess of the same binary to guarantee
	// command output is isolated from MCP stdio transport.
	s := mcp.NewGoogleServer(func(args []string) (string, string, error) {
		var outBuf bytes.Buffer
		var errBuf bytes.Buffer
		cmd := exec.CommandContext(ctx, exePath, args...) //nolint:gosec // exePath from config; user controls binary
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		runErr := cmd.Run()
		return outBuf.String(), errBuf.String(), runErr
	})
	return mcp.ServeStdio(ctx, os.Stdin, os.Stdout, s)
}
