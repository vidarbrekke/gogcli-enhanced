package google

import (
	"github.com/steipete/gogcli/internal/mcp/server"
)

func gmailSpecs(p *provider) []server.ToolSpec {
	return []server.ToolSpec{
		{
			Name:        "gmail_search",
			Description: "Search Gmail threads with Gmail query syntax. Returns threads + snippets; use page for pagination.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"query"},
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "Gmail search query (use from: for sender lookup)"},
					"max":   map[string]any{"type": "integer", "description": "Max results per page (default 10)"},
					"page":  map[string]any{"type": "string", "description": "Page token from previous response"},
				},
			},
			Handler: p.gmailSearch,
		}, {
			Name:        "gmail_send",
			Description: "Send an email via Gmail. Requires to, subject, and body (or bodyHtml).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"to", "subject", "body"},
				"properties": map[string]any{
					"to":       map[string]any{"type": "string", "description": "Recipients (comma-separated)"},
					"subject":  map[string]any{"type": "string"},
					"body":     map[string]any{"type": "string", "description": "Plain text body (use bodyHtml for HTML)"},
					"bodyHtml": map[string]any{"type": "string", "description": "HTML body (optional)"},
					"cc":       map[string]any{"type": "string"},
					"bcc":      map[string]any{"type": "string"},
					"from":     map[string]any{"type": "string", "description": "Send-from address (verified alias)"},
				},
			},
			Handler: p.gmailSend,
		},
	}
}
