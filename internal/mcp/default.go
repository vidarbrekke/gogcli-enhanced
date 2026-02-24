package mcp

import (
	"github.com/steipete/gogcli/internal/mcp/providers/google"
	"github.com/steipete/gogcli/internal/mcp/server"
)

func NewGoogleServer() *server.Server {
	s := server.New()
	google.Register(s)

	return s
}
