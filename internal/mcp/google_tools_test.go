package mcp

import (
	"context"
	"slices"
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

func TestGoogleTools_DocsInsertText_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"documentId":"d1","insertedChars":5}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.insertText", map[string]any{
		"docId":      "d1",
		"text":       "hello",
		"index":      float64(3),
		"validateOnly": true,
		"opId":       "op-1",
		"timeoutMs":  float64(8000),
		"retries":    float64(2),
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{
		"--json",
		"--request-timeout", "8000ms",
		"--retries", "2",
		"--op-id", "op-1",
		"docs", "edit", "insert", "d1", "hello",
		"--index", "3",
		"--validate-only",
	}
	if !slices.Equal(gotArgs, want) {
		t.Fatalf("unexpected args:\nwant=%v\ngot=%v", want, gotArgs)
	}
}

func TestGoogleTools_DriveSearchFiles_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"files":[]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.searchFiles", map[string]any{
		"query":       "budget q1",
		"rawQuery":    true,
		"allDrives":   false,
		"max":         float64(50),
		"page":        "p1",
		"account":     "a@example.com",
		"retryBackoffMs": float64(700),
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{
		"--json",
		"--retry-backoff", "700ms",
		"--account", "a@example.com",
		"drive", "search",
		"--raw-query",
		"--max", "50",
		"--page", "p1",
		"--no-all-drives",
		"budget q1",
	}
	if !slices.Equal(gotArgs, want) {
		t.Fatalf("unexpected args:\nwant=%v\ngot=%v", want, gotArgs)
	}
}

func TestGoogleTools_SheetsValuesUpdate_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"updatedCells":2}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets.valuesUpdate", map[string]any{
		"spreadsheetId": "s1",
		"range":         "Sheet1!A1:B1",
		"values":        []any{[]any{"a", "b"}},
		"valueInput":    "USER_ENTERED",
		"validateOnly":  true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "sheets") || !slices.Contains(gotArgs, "--values-json") {
		t.Fatalf("expected sheets values args, got=%v", gotArgs)
	}
}

func TestGoogleTools_SlidesReplaceText_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"occurrencesChanged":1}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "slides.replaceText", map[string]any{
		"presentationId": "p1",
		"find":           "Draft",
		"replace":        "Final",
		"matchCase":      true,
		"validateOnly":   true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	wantBits := []string{"slides", "edit", "replace-text", "p1", "--find", "Draft", "--replace", "Final", "--match-case", "--validate-only"}
	for _, bit := range wantBits {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}
