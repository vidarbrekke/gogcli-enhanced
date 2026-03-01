package cmd

import (
	"bytes"
	"context"
	"os"

	"github.com/steipete/gogcli/internal/mcp"
)

type MCPCmd struct {
	Serve MCPServeCmd `cmd:"" name:"serve" help:"Run MCP server over stdio"`
}

type MCPServeCmd struct{}

func (c *MCPServeCmd) Run(ctx context.Context) error {
	// CLI runs use injected I/O only: each tool call gets fresh buffers and
	// ExecuteWithIO(args, outBuf, errBuf). No global os.Stdout/os.Stderr swap
	// and no pipes, so no deadlock or FD leak from concurrent or large output.
	s := mcp.NewGoogleServer(func(args []string) (string, string, error) {
		var outBuf bytes.Buffer
		var errBuf bytes.Buffer
		execErr := ExecuteWithIO(args, &outBuf, &errBuf)
		return outBuf.String(), errBuf.String(), execErr
	})
	return mcp.ServeStdio(ctx, os.Stdin, os.Stdout, s)
}
