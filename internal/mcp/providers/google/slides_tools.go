package google

import (
	"github.com/steipete/gogcli/internal/mcp/server"
)

func slidesSpecs(p *provider) []server.ToolSpec {
	return []server.ToolSpec{
		{
			Name:        "slides_planBatch",
			Description: "Validate and plan a Slides batch update request without applying changes.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"presentationId", "request"},
				"properties": map[string]any{
					"presentationId": map[string]any{"type": "string"},
					"request":        map[string]any{"type": "object"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.slidesPlanBatch,
		}, {
			Name:        "slides_executeBatch",
			Description: "Execute a Slides batch update request.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-heavy",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"presentationId", "request"},
				"properties": map[string]any{
					"presentationId": map[string]any{"type": "string"},
					"request":        map[string]any{"type": "object"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.slidesExecuteBatch,
		}, {
			Name:        "slides_replaceText",
			Description: "Find and replace text across slides.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"presentationId", "find"},
				"properties": map[string]any{
					"presentationId": map[string]any{"type": "string"},
					"find":           map[string]any{"type": "string"},
					"replace":        map[string]any{"type": "string"},
					"matchCase":      map[string]any{"type": "boolean"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.slidesReplaceText,
		}, {
			Name:        "slides_createSlide",
			Description: "Create a new slide in a presentation.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"presentationId"},
				"properties": map[string]any{
					"presentationId": map[string]any{"type": "string"},
					"layout":         map[string]any{"type": "string"},
					"index":          map[string]any{"type": "integer"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.slidesCreateSlide,
		},
	}
}
