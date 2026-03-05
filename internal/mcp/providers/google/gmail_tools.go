package google

import (
	"github.com/steipete/gogcli/internal/mcp/server"
)

func gmailSpecs(p *provider) []server.ToolSpec {
	return []server.ToolSpec{
		{
			Name:        "gmail_search",
			Description: "Search Gmail threads using Gmail query syntax. Returns threads with id/snippet; use pageToken for pagination. For 'emails from [person]' always use the from: operator (e.g. from:Charlie Brekke or from:charlie@example.com); a bare phrase searches subject/body and may return the wrong thread. Results are roughly newest-first; use max to get enough and take the first as 'latest'.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"query"},
				"properties": map[string]any{
					"query":          map[string]any{"type": "string", "description": "Gmail search query. Use from:name or from:email for 'emails from X'; use is:unread, newer_than:7d, etc. as needed. Do not pass a bare name for sender—use from:Name."},
					"max":            map[string]any{"type": "integer", "description": "Max results per page (default 10)"},
					"page":           map[string]any{"type": "string", "description": "Page token from previous response"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
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
					"to":             map[string]any{"type": "string", "description": "Recipients (comma-separated)"},
					"subject":        map[string]any{"type": "string"},
					"body":           map[string]any{"type": "string", "description": "Plain text body (use bodyHtml for HTML)"},
					"bodyHtml":       map[string]any{"type": "string", "description": "HTML body (optional)"},
					"cc":             map[string]any{"type": "string"},
					"bcc":            map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"from":           map[string]any{"type": "string", "description": "Send-from address (verified alias)"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.gmailSend,
		},
	}
}
