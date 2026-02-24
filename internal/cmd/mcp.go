package cmd

import (
	"context"
	"io"
	"os"

	"github.com/steipete/gogcli/internal/mcp"
)

type MCPCmd struct {
	Serve MCPServeCmd `cmd:"" name:"serve" help:"Run MCP server over stdio"`
}

type MCPServeCmd struct{}

func (c *MCPServeCmd) Run(ctx context.Context) error {
	s := mcp.NewGoogleServer(func(args []string) (string, string, error) {
		oldOut := os.Stdout
		oldErr := os.Stderr
		outR, outW, _ := os.Pipe()
		errR, errW, _ := os.Pipe()
		os.Stdout = outW
		os.Stderr = errW
		execErr := Execute(args)
		_ = outW.Close()
		_ = errW.Close()
		outBytes, _ := io.ReadAll(outR)
		errBytes, _ := io.ReadAll(errR)
		os.Stdout = oldOut
		os.Stderr = oldErr
		return string(outBytes), string(errBytes), execErr
	})
	return mcp.ServeStdio(ctx, os.Stdin, os.Stdout, s)
}
