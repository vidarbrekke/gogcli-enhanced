package google

import (
	"github.com/steipete/gogcli/internal/mcp/server"
)

func contactsSpecs(p *provider) []server.ToolSpec {
	return []server.ToolSpec{
		{
			Name:        "contacts_list",
			Description: "List the user's contacts (People API). Returns resource, name, email, phone and nextPageToken.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"max":            map[string]any{"type": "integer", "description": "Max results (default 100)"},
					"page":           map[string]any{"type": "string", "description": "Page token"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.contactsList,
		},
	}
}
