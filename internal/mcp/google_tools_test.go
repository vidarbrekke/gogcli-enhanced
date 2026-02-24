package mcp

import (
	"context"
	"strings"
	"testing"
)

func TestGoogleTools_DocsPlanBatch(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) {
		if strings.Join(args, " ") == "" {
			t.Fatal("expected args")
		}

		return `{"validateOnly":true,"valid":true,"requestHash":"abc","opId":"mcp-docs-plan-1"}`, "", nil
	})

	env := s.ExecuteTool(context.Background(), "docs.planBatch", map[string]any{
		"opId":  "mcp-docs-plan-1",
		"docId": "d1",
		"request": map[string]any{
			"requests": []map[string]any{
				{
					"insertText": map[string]any{
						"location": map[string]any{"index": 1},
						"text":     "hello",
					},
				},
			},
		},
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}

	if env.Service != "docs" || env.Operation != "planBatch" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
}

func TestGoogleTools_DriveEnsureFolder_InvalidInput(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) {
		return "{}", "", nil
	})

	env := s.ExecuteTool(context.Background(), "drive.ensureFolder", map[string]any{})
	if env.OK {
		t.Fatal("expected invalid_argument")
	}

	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
	}
}
