package google

import (
	"github.com/steipete/gogcli/internal/mcp/server"
)

func calendarSpecs(p *provider) []server.ToolSpec {
	return []server.ToolSpec{
		{
			Name:        "calendar_events",
			Description: "List events from a calendar (or primary). Requires from and to (RFC3339 or date). Returns events and nextPageToken.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"from", "to"},
				"properties": map[string]any{
					"calendarId":     map[string]any{"type": "string", "description": "Calendar ID (default: primary)"},
					"from":           map[string]any{"type": "string", "description": "Start time (RFC3339 or YYYY-MM-DD)"},
					"to":             map[string]any{"type": "string", "description": "End time (RFC3339 or YYYY-MM-DD)"},
					"max":            map[string]any{"type": "integer", "description": "Max results (default 10)"},
					"page":           map[string]any{"type": "string", "description": "Page token"},
					"query":          map[string]any{"type": "string", "description": "Free text search"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.calendarEvents,
		},
	}
}
