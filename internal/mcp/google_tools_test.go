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

func TestGoogleTools_DriveUploadFile_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"id":"f1","name":"backup.tar.gz"}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.uploadFile", map[string]any{
		"localPath":           "/var/backups/backup.tar.gz",
		"parentId":             "pid1",
		"name":                 "backup.tar.gz",
		"keepRevisionForever":  true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{
		"--json",
		"drive", "upload", "/var/backups/backup.tar.gz",
		"--name", "backup.tar.gz",
		"--parent", "pid1",
		"--keep-revision-forever",
	}
	for _, a := range want {
		if !slices.Contains(gotArgs, a) {
			t.Fatalf("expected args to contain %q, got %v", a, gotArgs)
		}
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

func TestGoogleTools_DocsGet_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"file":{"id":"d1","name":"Doc1"},"document":{"documentId":"d1","title":"Doc1","revisionId":"rev1"}}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.get", map[string]any{
		"docId":   "d1",
		"account": "a@example.com",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Service != "docs" || env.Operation != "get" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
	want := []string{"--json", "--account", "a@example.com", "docs", "info", "d1"}
	for _, bit := range want {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

func TestGoogleTools_DocsGet_MissingDocId(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) { return "{}", "", nil })
	env := s.ExecuteTool(context.Background(), "docs.get", map[string]any{})
	if env.OK {
		t.Fatal("expected invalid_argument when docId missing")
	}
	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
	}
}

func TestGoogleTools_DocsCat_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"text":"Hello world"}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.cat", map[string]any{
		"docId":    "d1",
		"maxBytes": float64(50000),
		"tab":      "Sheet1",
		"account":  "a@example.com",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Service != "docs" || env.Operation != "cat" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
	want := []string{"--json", "--account", "a@example.com", "docs", "cat", "d1", "--max-bytes", "50000", "--tab", "Sheet1"}
	for _, bit := range want {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

func TestGoogleTools_DocsCat_AllTabs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"tabs":[{"id":"t1","text":"x"}]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.cat", map[string]any{
		"docId":   "d1",
		"allTabs": true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "--all-tabs") {
		t.Fatalf("expected --all-tabs in %v", gotArgs)
	}
}

func TestGoogleTools_DocsListTabs_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"tabs":[{"id":"t1","title":"Tab 1","index":0}]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.listTabs", map[string]any{
		"docId": "d1",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Service != "docs" || env.Operation != "listTabs" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
	want := []string{"--json", "docs", "list-tabs", "d1"}
	for _, bit := range want {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

func TestGoogleTools_DocsListTabs_MissingDocId(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) { return "{}", "", nil })
	env := s.ExecuteTool(context.Background(), "docs.listTabs", map[string]any{})
	if env.OK {
		t.Fatal("expected invalid_argument when docId missing")
	}
	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
	}
}

func TestGoogleTools_DocsPositionsEnd_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"docId":"d1","appendIndex":42}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.positionsEnd", map[string]any{
		"docId":   "d1",
		"account": "a@example.com",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Service != "docs" || env.Operation != "positionsEnd" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
	want := []string{"--json", "--account", "a@example.com", "docs", "positions", "end", "d1"}
	for _, bit := range want {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

func TestGoogleTools_DocsPositionsSearch_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"docId":"d1","text":"foo","ranges":[{"startIndex":1,"endIndex":4}]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.positionsSearch", map[string]any{
		"docId":     "d1",
		"text":      "foo",
		"matchCase": true,
		"account":   "a@example.com",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Service != "docs" || env.Operation != "positionsSearch" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
	want := []string{"--json", "--account", "a@example.com", "docs", "positions", "search", "d1", "--text", "foo", "--match-case"}
	for _, bit := range want {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

func TestGoogleTools_DocsPositionsSearch_MissingText(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) { return "{}", "", nil })
	env := s.ExecuteTool(context.Background(), "docs.positionsSearch", map[string]any{
		"docId": "d1",
	})
	if env.OK {
		t.Fatal("expected invalid_argument when text missing")
	}
	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
	}
}

func TestGoogleTools_DocsPositionsHeadings_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"docId":"d1","headings":[{"startIndex":1,"endIndex":10,"style":"HEADING_1","text":"Title"}]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "docs.positionsHeadings", map[string]any{
		"docId": "d1",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if env.Service != "docs" || env.Operation != "positionsHeadings" {
		t.Fatalf("unexpected service/op: %s %s", env.Service, env.Operation)
	}
	want := []string{"--json", "docs", "positions", "headings", "d1"}
	for _, bit := range want {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

func TestGoogleTools_DocsPositionsHeadings_MissingDocId(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) { return "{}", "", nil })
	env := s.ExecuteTool(context.Background(), "docs.positionsHeadings", map[string]any{})
	if env.OK {
		t.Fatal("expected invalid_argument when docId missing")
	}
	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
	}
}

func TestGoogleTools_DriveMoveFile_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"file":{"id":"f1","name":"x","parents":["p2"]}}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.moveFile", map[string]any{
		"fileId":   "f1",
		"parentId": "p2",
		"account":  "a@example.com",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{"--json", "--account", "a@example.com", "drive", "move", "f1", "--parent", "p2"}
	for _, bit := range want {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

func TestGoogleTools_DriveRenameFile_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"file":{"id":"f1","name":"NewName"}}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.renameFile", map[string]any{
		"fileId": "f1",
		"name":   "NewName",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "drive") || !slices.Contains(gotArgs, "rename") || !slices.Contains(gotArgs, "f1") || !slices.Contains(gotArgs, "NewName") {
		t.Fatalf("unexpected args: %v", gotArgs)
	}
}

func TestGoogleTools_DriveShareFile_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"permissionId":"perm1","link":"https://drive.google.com/..."}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.shareFile", map[string]any{
		"fileId": "f1",
		"to":     "user",
		"email":  "u@example.com",
		"role":   "writer",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{"--json", "drive", "share", "f1", "--to", "user", "--email", "u@example.com", "--role", "writer"}
	for _, bit := range want {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

func TestGoogleTools_DriveUnshare_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.unshare", map[string]any{
		"fileId":       "f1",
		"permissionId": "perm1",
		"force":        true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "--force") {
		t.Fatalf("expected --force in %v", gotArgs)
	}
}

func TestGoogleTools_DriveCreateComment_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"comment":{"id":"c1","content":"Hello"}}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.createComment", map[string]any{
		"fileId":  "f1",
		"content": "Hello",
		"quoted":  "anchor text",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "--quoted") || !slices.Contains(gotArgs, "anchor text") {
		t.Fatalf("expected --quoted and anchor in %v", gotArgs)
	}
}

func TestGoogleTools_DriveCopyFile_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"file":{"id":"copy1","name":"Copy of x"}}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.copyFile", map[string]any{
		"fileId":   "f1",
		"name":     "Copy of x",
		"parentId": "folder1",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{"--json", "drive", "copy", "f1", "Copy of x", "--parent", "folder1"}
	for _, bit := range want {
		if !slices.Contains(gotArgs, bit) {
			t.Fatalf("missing arg %q in %v", bit, gotArgs)
		}
	}
}

// TestGoogleTools_SuccessEnvelope_HasServiceAndOperation asserts all successful MCP tool responses include service and operation (envelope contract).
func TestGoogleTools_SuccessEnvelope_HasServiceAndOperation(t *testing.T) {
	tools := []struct {
		name string
		tool string
		args map[string]any
	}{
		{"docs_get", "docs.get", map[string]any{"docId": "d1"}},
		{"drive_listFiles", "drive.listFiles", map[string]any{"parentId": "root", "max": float64(5)}},
		{"drive_uploadFile", "drive.uploadFile", map[string]any{"localPath": "/tmp/upload-test.txt"}},
		{"sheets_links", "sheets.links", map[string]any{"spreadsheetId": "s1", "range": "A1"}},
	}
	for _, tt := range tools {
		t.Run(tt.name, func(t *testing.T) {
			s := NewGoogleServer(func(args []string) (string, string, error) {
				return "{}", "", nil
			})
			env := s.ExecuteTool(context.Background(), tt.tool, tt.args)
			if !env.OK {
				t.Fatalf("expected success, got error: %#v", env.Error)
			}
			if env.Service == "" {
				t.Fatalf("success envelope must include service")
			}
			if env.Operation == "" {
				t.Fatalf("success envelope must include operation")
			}
		})
	}
}

func TestGoogleTools_DriveDeleteFile_ValidateOnly_ReturnsPlanned(t *testing.T) {
	called := false
	s := NewGoogleServer(func(args []string) (string, string, error) {
		called = true
		return "{}", "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.deleteFile", map[string]any{
		"fileId":       "f1",
		"validateOnly": true,
		"permanent":    true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if called {
		t.Fatal("executor should not be called when validateOnly is true")
	}
	if v, _ := env.Result["validateOnly"].(bool); !v {
		t.Fatalf("expected validateOnly true, got %v", env.Result["validateOnly"])
	}
	planned, ok := env.Result["planned"].(map[string]any)
	if !ok || planned["fileId"] != "f1" || !planned["permanent"].(bool) {
		t.Fatalf("expected planned fileId and permanent, got %v", env.Result["planned"])
	}
}

func TestGoogleTools_DriveUnshare_ValidateOnly_ReturnsPlanned(t *testing.T) {
	called := false
	s := NewGoogleServer(func(args []string) (string, string, error) {
		called = true
		return "{}", "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.unshare", map[string]any{
		"fileId":       "f1",
		"permissionId": "perm1",
		"validateOnly": true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if called {
		t.Fatal("executor should not be called when validateOnly is true")
	}
	planned, ok := env.Result["planned"].(map[string]any)
	if !ok || planned["fileId"] != "f1" || planned["permissionId"] != "perm1" {
		t.Fatalf("expected planned fileId and permissionId, got %v", env.Result["planned"])
	}
}

func TestGoogleTools_DriveDeleteComment_ValidateOnly_ReturnsPlanned(t *testing.T) {
	called := false
	s := NewGoogleServer(func(args []string) (string, string, error) {
		called = true
		return "{}", "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive.deleteComment", map[string]any{
		"fileId":       "f1",
		"commentId":    "c1",
		"validateOnly": true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if called {
		t.Fatal("executor should not be called when validateOnly is true")
	}
	planned, ok := env.Result["planned"].(map[string]any)
	if !ok || planned["fileId"] != "f1" || planned["commentId"] != "c1" {
		t.Fatalf("expected planned fileId and commentId, got %v", env.Result["planned"])
	}
}
