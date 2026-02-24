package mcp

import (
	"github.com/steipete/gogcli/internal/mcp/providers/google"
	"github.com/steipete/gogcli/internal/mcp/server"
)

func NewGoogleServer(executor google.Executor) *server.Server {
	s := server.New()
	google.Register(s, executor)

	return s
}
