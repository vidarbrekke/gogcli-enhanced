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
		"docId":        "d1",
		"text":         "hello",
		"index":        float64(3),
		"validateOnly": true,
		"opId":         "op-1",
		"timeoutMs":    float64(8000),
		"retries":      float64(2),
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
		"query":          "budget q1",
		"rawQuery":       true,
		"allDrives":      false,
		"max":            float64(50),
		"page":           "p1",
		"account":        "a@example.com",
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

func TestGoogleTools_DriveSearchFiles_NoPage_UsesPaginatedMode(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"files":[],"nextPageToken":""}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.searchFiles", map[string]any{
		"query":    "mimeType = 'application/vnd.google-apps.folder'",
		"rawQuery": true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	// No --all; one page of mcpDrivePageSize (4) + --compact so response fits gateway limit.
	if slices.Contains(gotArgs, "--all") {
		t.Fatalf("expected no --all (paginated mode), got %v", gotArgs)
	}
	if !slices.Contains(gotArgs, "--max") {
		t.Fatalf("expected --max for page size, got %v", gotArgs)
	}
	if !slices.Contains(gotArgs, "--compact") {
		t.Fatalf("expected --compact, got %v", gotArgs)
	}
	// Default page size should be 4
	for i := range gotArgs {
		if gotArgs[i] == "--max" && i+1 < len(gotArgs) && gotArgs[i+1] != "4" {
			t.Fatalf("expected --max 4 when no page/max passed, got --max %s", gotArgs[i+1])
		}
	}
}

// TestGoogleTools_DriveSearchFiles_PageTokenMaxResultsAliases verifies Drive API-style args are mapped to our CLI.
func TestGoogleTools_DriveSearchFiles_PageTokenMaxResultsAliases(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"files":[],"nextPageToken":""}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.searchFiles", map[string]any{
		"query":       "mimeType = 'application/vnd.google-apps.folder'",
		"rawQuery":    true,
		"pageToken":  "token-from-drive-api",
		"maxResults": float64(25),
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "--page") {
		t.Fatalf("expected --page from pageToken alias, got %v", gotArgs)
	}
	for i := range gotArgs {
		if gotArgs[i] == "--page" && i+1 < len(gotArgs) && gotArgs[i+1] != "token-from-drive-api" {
			t.Fatalf("expected --page token-from-drive-api, got --page %s", gotArgs[i+1])
		}
		if gotArgs[i] == "--max" && i+1 < len(gotArgs) && gotArgs[i+1] != "25" {
			t.Fatalf("expected --max 25 from maxResults alias, got --max %s", gotArgs[i+1])
		}
	}
}

func TestGoogleTools_DriveListFiles_Global_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"files":[]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.listFiles", map[string]any{
		"global":         true,
		"maxResults":     float64(15),
		"retryBackoffMs": float64(700),
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "--global") {
		t.Fatalf("expected --global, got %v", gotArgs)
	}
	if slices.Contains(gotArgs, "--parent") {
		t.Fatalf("did not expect --parent with global listing, got %v", gotArgs)
	}
}

func TestGoogleTools_DriveListFiles_GlobalAndParent_Invalid(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) {
		return `{"files":[]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.listFiles", map[string]any{
		"global":   true,
		"parentId": "root",
	})
	if env.OK {
		t.Fatal("expected invalid_argument for global+parentId")
	}
	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
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

func TestGoogleTools_SheetsLinks_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"spreadsheetId":"s1","range":"Sheet1!A1:B10","links":[]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets.links", map[string]any{
		"spreadsheetId":  "s1",
		"range":          "Sheet1!A1:B10",
		"account":        "a@example.com",
		"retryBackoffMs": float64(700),
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{
		"--json",
		"--retry-backoff", "700ms",
		"--account", "a@example.com",
		"sheets", "links", "s1", "Sheet1!A1:B10",
	}
	if !slices.Equal(gotArgs, want) {
		t.Fatalf("unexpected args:\nwant=%v\ngot=%v", want, gotArgs)
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

func TestGoogleTools_DocsSed_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"status":"ok","docId":"d1","replaced":1,"engine":"sedmat"}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.sed", map[string]any{
		"docId":      "d1",
		"expression": "s/foo/bar/",
		"dryRun":     false,
		"opId":       "op-sed-1",
		"timeoutMs":  float64(5000),
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Service != "docs" || env.Operation != "sed" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
	wantBits := []string{"--json", "docs", "sed", "d1", "s/foo/bar/"}
	for _, bit := range wantBits {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
	if env.Result != nil {
		if e, _ := env.Result["engine"].(string); e != "sedmat" {
			t.Fatalf("expected result.engine=sedmat, got %v", env.Result["engine"])
		}
	}
}

func TestGoogleTools_DocsSed_InvalidInput(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) {
		return "{}", "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.sed", map[string]any{
		"expression": "s/foo/bar/",
	})
	if env.OK {
		t.Fatal("expected invalid_argument when docId missing")
	}
	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
	}
}

func TestGoogleTools_DocsSmartEdit_RoutingAndEnvelope(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) {
		return `{"status":"ok","docId":"d1","replaced":1,"engine":"sedmat"}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.smartEdit", map[string]any{
		"docId":        "d1",
		"intentType":   "sed",
		"expressions":  []any{"s/foo/bar/"},
		"validateOnly": false,
		"opId":         "op-smart-1",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Result != nil {
		if e, _ := env.Result["engineSelected"].(string); e != "sed" && e != "batch" {
			t.Fatalf("expected result.engineSelected sed or batch, got %v", env.Result["engineSelected"])
		}
		if _, ok := env.Result["decisionReason"]; !ok {
			t.Fatalf("expected result.decisionReason")
		}
		if _, ok := env.Result["riskLevel"]; !ok {
			t.Fatalf("expected result.riskLevel")
		}
	}
}

func TestGoogleTools_DocsSmartEdit_ValidateOnlyHighRisk(t *testing.T) {
	var executorCalls int
	s := NewGoogleServer(func(args []string) (string, string, error) {
		executorCalls++
		return "{}", "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.smartEdit", map[string]any{
		"docId":        "d1",
		"intentType":   "sed",
		"expressions":  []any{"d/delete-me/"},
		"validateOnly": true,
	})
	if !env.OK {
		t.Fatalf("expected success (assessment), got error: %#v", env.Error)
	}
	if env.Result != nil {
		if r, _ := env.Result["riskLevel"].(string); r != "high" {
			t.Fatalf("expected riskLevel=high for delete, got %v", env.Result["riskLevel"])
		}
		if rc, _ := env.Result["requiresConfirmation"].(bool); !rc {
			t.Fatalf("expected requiresConfirmation=true for high risk with validateOnly")
		}
	}
}

func TestGoogleTools_DocsCreate_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"file":{"id":"newDocId","name":"Test1","mimeType":"application/vnd.google-apps.document"}}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.create", map[string]any{
		"title":    "Test1",
		"parentId": "folderIdFromEnsure",
		"opId":     "op-create-1",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Service != "docs" || env.Operation != "create" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
	wantBits := []string{"--json", "docs", "create", "Test1", "--parent", "folderIdFromEnsure"}
	for _, bit := range wantBits {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

func TestGoogleTools_DocsCreateWithBody_NoRequest_ReturnsCreateResult(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) {
		return `{"file":{"id":"doc1","name":"Test1","mimeType":"application/vnd.google-apps.document"}}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.createWithBody", map[string]any{
		"title": "Test1",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Service != "docs" || env.Operation != "create" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
	if id := env.Result["documentId"]; id != "doc1" {
		t.Fatalf("expected documentId doc1, got %v", id)
	}
}

func TestGoogleTools_DocsCreateWithBody_WithRequest_CallsCreateThenBatch(t *testing.T) {
	callCount := 0
	s := NewGoogleServer(func(args []string) (string, string, error) {
		callCount++
		if callCount == 1 {
			return `{"file":{"id":"doc1","name":"Test1"}}`, "", nil
		}
		return `{"documentId":"doc1","operations":1,"replies":[]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.createWithBody", map[string]any{
		"title": "Test1",
		"request": map[string]any{
			"requests": []map[string]any{
				{"insertText": map[string]any{"location": map[string]any{"index": 1}, "text": "Hi"}},
			},
		},
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 executor calls (create + batch), got %d", callCount)
	}
}
