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
	s := mcp.NewGoogleServer(func(args []string) (string, string, error) {
		var outBuf bytes.Buffer
		var errBuf bytes.Buffer
		execErr := ExecuteWithIO(args, &outBuf, &errBuf)
		return outBuf.String(), errBuf.String(), execErr
	})
	return mcp.ServeStdio(ctx, os.Stdin, os.Stdout, s)
}
