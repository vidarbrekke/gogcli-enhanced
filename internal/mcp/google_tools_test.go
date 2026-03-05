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

	env := s.ExecuteTool(context.Background(), "docs_planBatch", map[string]any{
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

	env := s.ExecuteTool(context.Background(), "drive_ensureFolder", map[string]any{})
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
	env := s.ExecuteTool(context.Background(), "drive_uploadFile", map[string]any{
		"localPath":           "/var/backups/backup.tar.gz",
		"parentId":            "pid1",
		"name":                "backup.tar.gz",
		"keepRevisionForever": true,
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
	env := s.ExecuteTool(context.Background(), "docs_insertText", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_searchFiles", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_searchFiles", map[string]any{
		"query":    "mimeType = 'application/vnd.google-apps.folder'",
		"rawQuery": true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	// No --all; one page of mcpDrivePageSize (25) + --compact so response fits gateway limit.
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
		if gotArgs[i] == "--max" && i+1 < len(gotArgs) && gotArgs[i+1] != "25" {
			t.Fatalf("expected --max 25 when no page/max passed, got --max %s", gotArgs[i+1])
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
	env := s.ExecuteTool(context.Background(), "drive_searchFiles", map[string]any{
		"query":      "mimeType = 'application/vnd.google-apps.folder'",
		"rawQuery":   true,
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

// TestGoogleTools_DriveSearchFiles_FetchAllPages verifies fetchAllPages: true results in --all (no pageToken chaining).
func TestGoogleTools_DriveSearchFiles_FetchAllPages(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"files":[],"totalCount":85,"truncatedAt":10}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive_searchFiles", map[string]any{
		"query":         "mimeType = 'application/vnd.google-apps.folder' and 'root' in parents",
		"rawQuery":      true,
		"fetchAllPages": true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "--all") {
		t.Fatalf("expected --all when fetchAllPages true, got %v", gotArgs)
	}
	if slices.Contains(gotArgs, "--compact") {
		t.Fatalf("expected no --compact when fetchAllPages (--all), got %v", gotArgs)
	}
}

func TestGoogleTools_DriveListFiles_Global_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"files":[]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "drive_listFiles", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_listFiles", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "sheets_valuesUpdate", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "sheets_links", map[string]any{
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

func TestGoogleTools_SheetsValuesGet_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"range":"Sheet1!A1:B2","values":[["a","b"],["c","d"]]}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets_valuesGet", map[string]any{
		"spreadsheetId":     "s1",
		"range":             "Sheet1!A1:B2",
		"majorDimension":    "ROWS",
		"valueRenderOption": "FORMATTED_VALUE",
		"account":           "a@example.com",
		"retryBackoffMs":    float64(700),
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{
		"--json",
		"--retry-backoff", "700ms",
		"--account", "a@example.com",
		"sheets", "get", "s1", "Sheet1!A1:B2",
		"--dimension", "ROWS",
		"--render", "FORMATTED_VALUE",
	}
	for _, a := range want {
		if !slices.Contains(gotArgs, a) {
			t.Fatalf("expected args to contain %q, got %v", a, gotArgs)
		}
	}
}

func TestGoogleTools_SheetsValuesGet_InvalidInput(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) {
		return "{}", "", nil
	})
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{"missing spreadsheetId", map[string]any{"range": "Sheet1!A1"}},
		{"missing range", map[string]any{"spreadsheetId": "s1"}},
		{"both missing", map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := s.ExecuteTool(context.Background(), "sheets_valuesGet", tc.args)
			if env.OK {
				t.Fatal("expected invalid_argument")
			}
			if env.Error == nil || env.Error.Code != "invalid_argument" {
				t.Fatalf("unexpected error: %#v", env.Error)
			}
		})
	}
}

func TestGoogleTools_SheetsSortRange_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets_sortRange", map[string]any{
		"spreadsheetId": "s1",
		"range":         "Sheet1!A2:J200",
		"sortByColumn":  float64(1),
		"desc":          true,
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{"--json", "sheets", "sort", "s1", "Sheet1!A2:J200", "--by-column", "1", "--desc"}
	for _, a := range want {
		if !slices.Contains(gotArgs, a) {
			t.Fatalf("expected args to contain %q, got %v", a, gotArgs)
		}
	}
}

func TestGoogleTools_SheetsDedupeRows_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets_dedupeRows", map[string]any{
		"spreadsheetId": "s1",
		"range":         "Sheet1!A2:J200",
		"keyColumns":    []any{float64(0), float64(2)},
		"keep":          "first",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{"--json", "sheets", "dedupe", "s1", "Sheet1!A2:J200", "--key-columns", "0,2", "--keep", "first"}
	for _, a := range want {
		if !slices.Contains(gotArgs, a) {
			t.Fatalf("expected args to contain %q, got %v", a, gotArgs)
		}
	}
}

func TestGoogleTools_SheetsDedupeRows_MissingRange_ReturnsError(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) {
		return "{}", "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets_dedupeRows", map[string]any{
		"spreadsheetId": "s1",
	})
	if env.OK {
		t.Fatal("expected error for missing range")
	}
	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
	}
}

func TestGoogleTools_SheetsFilterCopyRows_InvalidColumnOrOp(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) { return "{}", "", nil })
	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing column", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A1:B2", "targetSheet": "T", "op": "eq", "value": "x"}},
		{"invalid op", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A1:B2", "targetSheet": "T", "column": float64(0), "op": "invalid", "value": "x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := s.ExecuteTool(context.Background(), "sheets_filterCopyRows", tt.args)
			if env.OK {
				t.Fatal("expected invalid_argument")
			}
			if env.Error == nil || env.Error.Code != "invalid_argument" {
				t.Fatalf("unexpected error: %#v", env.Error)
			}
		})
	}
}

func TestGoogleTools_SheetsUpsertRows_InvalidKeyColumns(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) { return "{}", "", nil })
	env := s.ExecuteTool(context.Background(), "sheets_upsertRows", map[string]any{
		"spreadsheetId": "s1",
		"range":         "Sheet1!A2:B10",
		"keyColumns":    []any{"not", "ints"},
		"rows":          []any{[]any{"a", "b"}},
	})
	if env.OK {
		t.Fatal("expected invalid_argument for non-integer keyColumns")
	}
	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
	}
}

func TestGoogleTools_SheetsSummarize_InvalidGroupBy(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) { return "{}", "", nil })
	env := s.ExecuteTool(context.Background(), "sheets_summarize", map[string]any{
		"spreadsheetId": "s1",
		"range":         "Sheet1!A2:B10",
		"groupBy":       []any{},
		"aggregate":     "count",
	})
	if env.OK {
		t.Fatal("expected invalid_argument for empty groupBy")
	}
	if env.Error == nil || env.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", env.Error)
	}
}

func TestGoogleTools_SheetsFilterCopyRows_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"rowsCopied":5}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets_filterCopyRows", map[string]any{
		"spreadsheetId":   "s1",
		"range":           "Sheet1!A2:J200",
		"targetSheet":     "Filtered",
		"column":          float64(1),
		"op":              "eq",
		"value":           "yes",
		"destinationCell": "A1",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{"--json", "sheets", "filter-copy", "s1", "Sheet1!A2:J200", "Filtered", "--column", "1", "--op", "eq", "--value", "yes", "--destination-cell", "A1"}
	for _, a := range want {
		if !slices.Contains(gotArgs, a) {
			t.Fatalf("expected args to contain %q, got %v", a, gotArgs)
		}
	}
}

func TestGoogleTools_SheetsUpsertRows_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"updated":1,"appended":0}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets_upsertRows", map[string]any{
		"spreadsheetId": "s1",
		"range":         "Sheet1!A2:B10",
		"keyColumns":    []any{float64(0), float64(1)},
		"rows":          []any{[]any{"a", "b"}, []any{"c", "d"}},
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "sheets") || !slices.Contains(gotArgs, "upsert") || !slices.Contains(gotArgs, "--key-columns") {
		t.Fatalf("expected upsert args, got %v", gotArgs)
	}
}

func TestGoogleTools_SheetsMoveRows_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"rowsMoved":3}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets_moveRows", map[string]any{
		"spreadsheetId": "s1",
		"range":         "Sheet1!A2:J200",
		"targetSheet":   "Out",
		"column":        float64(0),
		"op":            "eq",
		"value":         "x",
		"mode":          "move",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	want := []string{"sheets", "move-rows", "s1", "Sheet1!A2:J200", "Out", "--column", "0", "--op", "eq", "--value", "x", "--mode", "move"}
	for _, a := range want {
		if !slices.Contains(gotArgs, a) {
			t.Fatalf("expected args to contain %q, got %v", a, gotArgs)
		}
	}
}

func TestGoogleTools_SheetsApplyFormula_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"rowsUpdated":5}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets_applyFormula", map[string]any{
		"spreadsheetId": "s1",
		"range":         "Sheet1!C2:C10",
		"formula":       "=A{row}+B{row}",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "apply-formula") || !slices.Contains(gotArgs, "--formula") {
		t.Fatalf("expected apply-formula args, got %v", gotArgs)
	}
}

func TestGoogleTools_SheetsSummarize_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"rowCount":3}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "sheets_summarize", map[string]any{
		"spreadsheetId": "s1",
		"range":         "Sheet1!A2:B10",
		"groupBy":       []any{float64(0)},
		"metricColumn":  float64(1),
		"aggregate":     "sum",
	})
	if !env.OK {
		t.Fatalf("expected success, got error: %#v", env.Error)
	}
	if !slices.Contains(gotArgs, "summarize") || !slices.Contains(gotArgs, "--group-by") || !slices.Contains(gotArgs, "--aggregate") {
		t.Fatalf("expected summarize args, got %v", gotArgs)
	}
}

func TestGoogleTools_SlidesReplaceText_MapsArgs(t *testing.T) {
	var gotArgs []string
	s := NewGoogleServer(func(args []string) (string, string, error) {
		gotArgs = append([]string{}, args...)
		return `{"occurrencesChanged":1}`, "", nil
	})
	env := s.ExecuteTool(context.Background(), "slides_replaceText", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_sed", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_sed", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_smartEdit", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_smartEdit", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_create", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_createWithBody", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_createWithBody", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_get", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_get", map[string]any{})
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
	env := s.ExecuteTool(context.Background(), "docs_cat", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_cat", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_listTabs", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_listTabs", map[string]any{})
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
	env := s.ExecuteTool(context.Background(), "docs_positionsEnd", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_positionsSearch", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_positionsSearch", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_positionsHeadings", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "docs_positionsHeadings", map[string]any{})
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
	env := s.ExecuteTool(context.Background(), "drive_moveFile", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_renameFile", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_shareFile", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_unshare", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_createComment", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_copyFile", map[string]any{
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

// expectedSheetsTools is the full list of sheet tool names that must be registered.
// If the deployed binary shows only 6 sheet tools, it was built from an older commit—run deploy.sh on the server.
var expectedSheetsTools = []string{
	"sheets_applyFormula", "sheets_clear", "sheets_dedupeRows", "sheets_executeBatch", "sheets_filterCopyRows",
	"sheets_links", "sheets_metadata", "sheets_moveRows", "sheets_planBatch", "sheets_sortRange", "sheets_summarize",
	"sheets_upsertRows", "sheets_valuesAppend", "sheets_valuesGet", "sheets_valuesRead", "sheets_valuesUpdate",
}

func TestGoogleTools_SheetsToolsRegistered(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) { return "{}", "", nil })
	specs := s.ListToolSpecs()
	names := make(map[string]bool)
	for _, spec := range specs {
		names[spec.Name] = true
	}
	// Regression: expect 64 tools total (60 + gmail_search, gmail_send, calendar_events, contacts_list).
	if len(specs) != 64 {
		t.Errorf("expected 64 registered tools, got %d", len(specs))
	}
	for _, want := range expectedSheetsTools {
		if !names[want] {
			t.Errorf("sheet tool %q not registered; only %d sheet tools in binary (expected %d). Deploy with ./scripts/deploy.sh on the server.", want, len(names), len(expectedSheetsTools))
		}
	}
}

// TestGoogleTools_RegisteredDomainOrderAndCounts asserts the Phase 1 split contract: docs, sheets, slides, drive, gmail, calendar, contacts contribute the expected counts. ListToolSpecs returns tools sorted by name.
func TestGoogleTools_RegisteredDomainOrderAndCounts(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) { return "{}", "", nil })
	specs := s.ListToolSpecs()
	if len(specs) != 64 {
		t.Fatalf("expected 64 tools, got %d", len(specs))
	}
	var docs, sheets, slides, drive, gmail, calendar, contacts int
	for _, spec := range specs {
		switch {
		case strings.HasPrefix(spec.Name, "docs_"):
			docs++
		case strings.HasPrefix(spec.Name, "sheets_"):
			sheets++
		case strings.HasPrefix(spec.Name, "slides_"):
			slides++
		case strings.HasPrefix(spec.Name, "drive_"):
			drive++
		case strings.HasPrefix(spec.Name, "gmail_"):
			gmail++
		case strings.HasPrefix(spec.Name, "calendar_"):
			calendar++
		case strings.HasPrefix(spec.Name, "contacts_"):
			contacts++
		default:
			t.Errorf("unexpected tool prefix: %q", spec.Name)
		}
	}
	if docs != 21 {
		t.Errorf("expected 21 docs tools, got %d", docs)
	}
	if sheets != 16 {
		t.Errorf("expected 16 sheets tools, got %d", sheets)
	}
	if slides != 4 {
		t.Errorf("expected 4 slides tools, got %d", slides)
	}
	if drive != 19 {
		t.Errorf("expected 19 drive tools, got %d", drive)
	}
	if gmail != 2 {
		t.Errorf("expected 2 gmail tools, got %d", gmail)
	}
	if calendar != 1 {
		t.Errorf("expected 1 calendar tool, got %d", calendar)
	}
	if contacts != 1 {
		t.Errorf("expected 1 contacts tool, got %d", contacts)
	}
}

// TestGoogleTools_SuccessEnvelope_HasServiceAndOperation asserts all successful MCP tool responses include service and operation (envelope contract).
func TestGoogleTools_SuccessEnvelope_HasServiceAndOperation(t *testing.T) {
	tools := []struct {
		name string
		tool string
		args map[string]any
	}{
		{"docs_get", "docs_get", map[string]any{"docId": "d1"}},
		{"drive_listFiles", "drive_listFiles", map[string]any{"parentId": "root", "max": float64(5)}},
		{"drive_uploadFile", "drive_uploadFile", map[string]any{"localPath": "/tmp/upload-test.txt"}},
		{"sheets_links", "sheets_links", map[string]any{"spreadsheetId": "s1", "range": "A1"}},
		{"sheets_valuesGet", "sheets_valuesGet", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A1"}},
		{"sheets_sortRange", "sheets_sortRange", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A2:B10"}},
		{"sheets_dedupeRows", "sheets_dedupeRows", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A2:B10"}},
		{"sheets_filterCopyRows", "sheets_filterCopyRows", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A2:B10", "targetSheet": "Out", "column": float64(0), "op": "eq", "value": "x"}},
		{"sheets_upsertRows", "sheets_upsertRows", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A2:B10", "keyColumns": []any{float64(0)}, "rows": []any{[]any{"a", "b"}}}},
		{"sheets_moveRows", "sheets_moveRows", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A2:B10", "targetSheet": "Out", "column": float64(0), "op": "eq", "value": "x"}},
		{"sheets_applyFormula", "sheets_applyFormula", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!C2:C10", "formula": "=A{row}+B{row}"}},
		{"sheets_summarize", "sheets_summarize", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A2:B10", "groupBy": []any{float64(0)}, "aggregate": "count"}},
		{"sheets_clear", "sheets_clear", map[string]any{"spreadsheetId": "s1", "range": "Sheet1!A1:B2"}},
		{"sheets_metadata", "sheets_metadata", map[string]any{"spreadsheetId": "s1"}},
		{"docs_export", "docs_export", map[string]any{"docId": "d1", "format": "pdf"}},
		{"gmail_search", "gmail_search", map[string]any{"query": "from:test@example.com", "max": float64(5)}},
		{"gmail_send", "gmail_send", map[string]any{"to": "x@y.com", "subject": "Test", "body": "Hello"}},
		{"calendar_events", "calendar_events", map[string]any{"from": "2025-01-01T00:00:00Z", "to": "2025-01-02T00:00:00Z", "max": float64(10)}},
		{"contacts_list", "contacts_list", map[string]any{"max": float64(5)}},
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
	env := s.ExecuteTool(context.Background(), "drive_deleteFile", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_unshare", map[string]any{
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
	env := s.ExecuteTool(context.Background(), "drive_deleteComment", map[string]any{
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
