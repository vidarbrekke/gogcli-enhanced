package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func TestExecute_SheetsEditValues_JSON(t *testing.T) {
	origSheets := newSheetsService
	t.Cleanup(func() { newSheetsService = origSheets })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/d1/values/") {
			var vr sheets.ValueRange
			if err := json.NewDecoder(r.Body).Decode(&vr); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(vr.Values) != 1 || len(vr.Values[0]) != 2 {
				t.Fatalf("unexpected values: %#v", vr.Values)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"updatedRange":   "Sheet1!A1:B1",
				"updatedRows":    1,
				"updatedColumns": 2,
				"updatedCells":   2,
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewSheetsService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return svc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "sheets", "edit", "values", "d1", "Sheet1!A1:B1", "a|b"}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["updatedCells"] != float64(2) {
		t.Fatalf("updatedCells=%v", parsed["updatedCells"])
	}
}

func TestExecute_SheetsEditValues_DryRun_Pretty_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "sheets", "edit", "values", "d1", "Sheet1!A1:B1", "a|b", "--dry-run", "--pretty"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["dryRun"] != true {
		t.Fatalf("dryRun=%v", parsed["dryRun"])
	}
	hash, ok := parsed["requestHash"].(string)
	if !ok || len(hash) != 64 {
		t.Fatalf("requestHash=%v", parsed["requestHash"])
	}
}

func TestExecute_SheetsEditValues_OutputRequestFile_JSON(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "values-request.json")
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "sheets", "edit", "values", "d1", "Sheet1!A1:B1", "a|b", "--dry-run", "--output-request-file", outFile}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(b), "\"values\"") {
		t.Fatalf("expected values request in output file, got: %q", string(b))
	}
}

func TestExecute_SheetsEditAppend_JSON(t *testing.T) {
	origSheets := newSheetsService
	t.Cleanup(func() { newSheetsService = origSheets })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, ":append") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"updates": map[string]any{
					"updatedRange":   "Sheet1!A1:B1",
					"updatedRows":    1,
					"updatedColumns": 2,
					"updatedCells":   2,
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewSheetsService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return svc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "sheets", "edit", "append", "d1", "Sheet1!A:C", "a|b"}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["updatedCells"] != float64(2) {
		t.Fatalf("updatedCells=%v", parsed["updatedCells"])
	}
}

func TestExecute_SheetsEditClear_JSON(t *testing.T) {
	origSheets := newSheetsService
	t.Cleanup(func() { newSheetsService = origSheets })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, ":clear") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"clearedRange": "Sheet1!A1:B2",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewSheetsService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return svc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "sheets", "edit", "clear", "d1", "Sheet1!A1:B2"}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["clearedRange"] != "Sheet1!A1:B2" {
		t.Fatalf("clearedRange=%v", parsed["clearedRange"])
	}
}

func TestExecute_SheetsEditClear_RequiresForceOrDryRun(t *testing.T) {
	err := Execute([]string{"--account", "a@b.com", "sheets", "edit", "clear", "d1", "Sheet1!A1:B2"})
	if err == nil || !strings.Contains(err.Error(), "destructive") {
		t.Fatalf("expected destructive guard error, got: %v", err)
	}
}

func TestExecute_SheetsEditBatch_JSONFile(t *testing.T) {
	origSheets := newSheetsService
	t.Cleanup(func() { newSheetsService = origSheets })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v4/spreadsheets/d1:batchUpdate" {
			var req sheets.BatchUpdateSpreadsheetRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(req.Requests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(req.Requests))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spreadsheetId": "d1",
				"replies":       []any{map[string]any{}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewSheetsService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return svc, nil }

	tmp, err := os.CreateTemp(t.TempDir(), "sheets-batch-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	payload := `{"requests":[{"addSheet":{"properties":{"title":"NewSheet"}}}]}`
	if _, err := tmp.WriteString(payload); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "sheets", "edit", "batch", "d1", "--requests-file", tmp.Name()}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["operations"] != float64(1) {
		t.Fatalf("operations=%v", parsed["operations"])
	}
}

func TestExecute_SheetsEditBatch_ValidateOnly_JSON(t *testing.T) {
	withStdin(t, `{"requests":[{"addSheet":{"properties":{"title":"NewSheet"}}}]}`, func() {
		out := captureStdout(t, func() {
			stderr := captureStderr(t, func() {
				if err := Execute([]string{"--json", "sheets", "edit", "batch", "d1", "--requests-file", "-", "--validate-only"}); err != nil {
					t.Fatalf("Execute: %v", err)
				}
			})
			if strings.TrimSpace(stderr) != "" {
				t.Fatalf("unexpected stderr: %q", stderr)
			}
		})
		var parsed map[string]any
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			t.Fatalf("parse json: %v; out=%q", err, out)
		}
		if parsed["validateOnly"] != true || parsed["valid"] != true {
			t.Fatalf("unexpected validate payload: %#v", parsed)
		}
		hash, ok := parsed["requestHash"].(string)
		if !ok || len(hash) != 64 {
			t.Fatalf("requestHash=%v", parsed["requestHash"])
		}
	})
}

func TestExecute_SheetsEditBatch_InvalidRequest_JSONErrorEnvelope(t *testing.T) {
	stderr := captureStderr(t, func() {
		withStdin(t, `{"requests":[{"addSheet":{"properties":{"title":"x"}},"deleteSheet":{"sheetId":1}}]}`, func() {
			err := Execute([]string{"--json", "sheets", "edit", "batch", "d1", "--requests-file", "-"})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &parsed); err != nil {
		t.Fatalf("parse stderr json: %v; stderr=%q", err, stderr)
	}
	errorObj, ok := parsed["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %#v", parsed)
	}
	if errorObj["error_code"] != "invalid_request" {
		t.Fatalf("error_code=%v", errorObj["error_code"])
	}
	if errorObj["request_index"] != float64(0) {
		t.Fatalf("request_index=%v", errorObj["request_index"])
	}
	if errorObj["operation"] != "batch" {
		t.Fatalf("operation=%v", errorObj["operation"])
	}
}

func TestExecute_SheetsEditBatch_OutputRequestFileDash_JSON(t *testing.T) {
	withStdin(t, `{"requests":[{"addSheet":{"properties":{"title":"NewSheet"}}}]}`, func() {
		out := captureStdout(t, func() {
			stderr := captureStderr(t, func() {
				if err := Execute([]string{"--json", "sheets", "edit", "batch", "d1", "--requests-file", "-", "--validate-only", "--output-request-file", "-"}); err != nil {
					t.Fatalf("Execute: %v", err)
				}
			})
			if strings.TrimSpace(stderr) != "" {
				t.Fatalf("unexpected stderr: %q", stderr)
			}
		})
		var parsed map[string]any
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			t.Fatalf("expected single JSON envelope, parse err=%v out=%q", err, out)
		}
		norm, ok := parsed["normalizedRequest"].(string)
		if !ok || !strings.Contains(norm, "\"requests\"") {
			t.Fatalf("normalizedRequest=%v", parsed["normalizedRequest"])
		}
	})
}

func TestExecute_SheetsEditReplaceText_JSON(t *testing.T) {
	origSheets := newSheetsService
	t.Cleanup(func() { newSheetsService = origSheets })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v4/spreadsheets/d1:batchUpdate" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spreadsheetId": "d1",
				"replies": []any{
					map[string]any{
						"findReplace": map[string]any{
							"occurrencesChanged": 3,
						},
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewSheetsService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return svc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "sheets", "edit", "replace-text", "d1", "--find", "old", "--replace", "new"}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["occurrencesChanged"] != float64(3) {
		t.Fatalf("occurrencesChanged=%v", parsed["occurrencesChanged"])
	}
}

func TestExecute_SheetsEditValues_ValidateOnly_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "sheets", "edit", "values", "d1", "Sheet1!A1:B1", "a|b", "--validate-only"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["validateOnly"] != true || parsed["valid"] != true {
		t.Fatalf("unexpected validate payload: %#v", parsed)
	}
}

func TestExecute_SheetsEditAppend_ValidateOnly_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "sheets", "edit", "append", "d1", "Sheet1!A:C", "a|b", "--validate-only"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["validateOnly"] != true || parsed["valid"] != true {
		t.Fatalf("unexpected validate payload: %#v", parsed)
	}
}

func TestExecute_SheetsEditReplaceText_Invalid_JSONErrorEnvelope(t *testing.T) {
	stderr := captureStderr(t, func() {
		err := Execute([]string{"--json", "sheets", "edit", "replace-text", "d1", "--replace", "new"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &parsed); err != nil {
		t.Fatalf("parse stderr json: %v; stderr=%q", err, stderr)
	}
	errorObj, ok := parsed["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %#v", parsed)
	}
	if errorObj["error_code"] != "invalid_argument" {
		t.Fatalf("error_code=%v", errorObj["error_code"])
	}
	if errorObj["operation"] != "replace-text" {
		t.Fatalf("operation=%v", errorObj["operation"])
	}
}

func TestExecute_SheetsEditFormat_ValidateOnly_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{
				"--json", "sheets", "edit", "format", "d1", "Sheet1!A1:B1",
				"--format-json", `{"textFormat":{"bold":true}}`,
				"--format-fields", "textFormat.bold",
				"--validate-only",
			}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["validateOnly"] != true || parsed["valid"] != true {
		t.Fatalf("unexpected validate payload: %#v", parsed)
	}
}

func TestExecute_SheetsEditInsert_ValidateOnly_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "sheets", "edit", "insert", "d1", "Sheet1", "rows", "2", "--count", "3", "--validate-only"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["dimension"] != "ROWS" {
		t.Fatalf("dimension=%v", parsed["dimension"])
	}
}

func TestExecute_SheetsEditDeleteRange_ValidateOnly_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "sheets", "edit", "delete-range", "d1", "Sheet1!A1:C10", "--shift-dimension", "ROWS", "--validate-only"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["validateOnly"] != true || parsed["valid"] != true {
		t.Fatalf("unexpected validate payload: %#v", parsed)
	}
	hash, ok := parsed["requestHash"].(string)
	if !ok || len(hash) != 64 {
		t.Fatalf("requestHash=%v", parsed["requestHash"])
	}
}

func TestExecute_SheetsEditDeleteRange_DryRun_Pretty_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{"--json", "sheets", "edit", "delete-range", "d1", "Sheet1!A1:C10", "--shift-dimension", "COLUMNS", "--dry-run", "--pretty"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["dryRun"] != true {
		t.Fatalf("dryRun=%v", parsed["dryRun"])
	}
	if parsed["service"] != "sheets" {
		t.Fatalf("service=%v", parsed["service"])
	}
}

func TestExecute_SheetsEditDeleteRange_RequiresForceOrDryRun(t *testing.T) {
	err := Execute([]string{"--account", "a@b.com", "sheets", "edit", "delete-range", "d1", "Sheet1!A1:B2", "--shift-dimension", "ROWS"})
	if err == nil || !strings.Contains(err.Error(), "destructive") {
		t.Fatalf("expected destructive guard error, got: %v", err)
	}
}

func TestExecute_SheetsEditDeleteRange_JSON(t *testing.T) {
	origSheets := newSheetsService
	t.Cleanup(func() { newSheetsService = origSheets })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v4/spreadsheets/d1:batchUpdate" {
			var req sheets.BatchUpdateSpreadsheetRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(req.Requests) != 1 || req.Requests[0].DeleteRange == nil {
				t.Fatalf("expected one DeleteRange request, got %d requests", len(req.Requests))
			}
			dr := req.Requests[0].DeleteRange
			if dr.ShiftDimension != "ROWS" {
				t.Fatalf("shiftDimension=%q", dr.ShiftDimension)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spreadsheetId": "d1",
				"replies":       []any{map[string]any{}},
			})
			return
		}
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v4/spreadsheets/d1") && strings.Contains(r.URL.RawQuery, "fields=") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sheets": []any{
					map[string]any{"properties": map[string]any{"sheetId": int64(0), "title": "Sheet1"}},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewSheetsService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return svc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "sheets", "edit", "delete-range", "d1", "Sheet1!A1:C10", "--shift-dimension", "ROWS", "--force"}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["deletedRange"] != "Sheet1!A1:C10" {
		t.Fatalf("deletedRange=%v", parsed["deletedRange"])
	}
	if parsed["shiftDimension"] != "ROWS" {
		t.Fatalf("shiftDimension=%v", parsed["shiftDimension"])
	}
}

func TestExecute_SheetsEditMergeData_ValidateOnly_JSON(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "merge-data.json")
	if err := os.WriteFile(dataFile, []byte(`[{"quarter":"Q1","year":"2026"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	out := captureStdout(t, func() {
		if err := Execute([]string{
			"--json", "sheets", "edit", "merge-data", "tpl1",
			"--data-file", dataFile,
			"--filename-format", "Report - {{quarter}} {{year}}",
			"--validate-only",
		}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["validateOnly"] != true || parsed["valid"] != true {
		t.Fatalf("validateOnly=%v valid=%v", parsed["validateOnly"], parsed["valid"])
	}
	if parsed["templateId"] != "tpl1" || parsed["recordCount"] != float64(1) {
		t.Fatalf("templateId=%v recordCount=%v", parsed["templateId"], parsed["recordCount"])
	}
	if !strings.Contains(fmt.Sprint(parsed["sampleFilename"]), "Q1") {
		t.Fatalf("sampleFilename should contain Q1: %v", parsed["sampleFilename"])
	}
}

func TestExecute_SheetsEditMergeData_DryRun_JSON(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "merge-data.json")
	if err := os.WriteFile(dataFile, []byte(`[{"a":"1"},{"a":"2"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	out := captureStdout(t, func() {
		if err := Execute([]string{
			"--json", "sheets", "edit", "merge-data", "tpl1",
			"--data-file", dataFile,
			"--dry-run",
		}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["dryRun"] != true || parsed["service"] != "sheets" {
		t.Fatalf("dryRun=%v service=%v", parsed["dryRun"], parsed["service"])
	}
	if parsed["recordCount"] != float64(2) {
		t.Fatalf("recordCount=%v", parsed["recordCount"])
	}
}

func TestExecute_SheetsEditMergeData_Success_JSON(t *testing.T) {
	origSheets := newSheetsService
	origDrive := newDriveService
	t.Cleanup(func() {
		newSheetsService = origSheets
		newDriveService = origDrive
	})

	path := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(path, "files/tpl1") && strings.Contains(r.URL.RawQuery, "parents"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "tpl1", "parents": []string{"folder1"}})
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/copy"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "new-sheet-1", "parents": []string{"folder1"}})
		case r.Method == http.MethodPost && strings.Contains(path, "spreadsheets/new-sheet-1:batchUpdate"):
			var req sheets.BatchUpdateSpreadsheetRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode batchUpdate: %v", err)
			}
			if len(req.Requests) < 1 {
				t.Fatalf("expected FindReplace requests")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"spreadsheetId": "new-sheet-1",
				"replies":       []any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sheetSvc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("sheets.NewService: %v", err)
	}
	driveSvc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("drive.NewService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return sheetSvc, nil }
	newDriveService = func(context.Context, string) (*drive.Service, error) { return driveSvc, nil }

	dataFile := filepath.Join(t.TempDir(), "merge-data.json")
	if err := os.WriteFile(dataFile, []byte(`[{"quarter":"Q1","revenue":"$1.2M"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{
				"--json", "--account", "a@b.com",
				"sheets", "edit", "merge-data", "tpl1",
				"--data-file", dataFile,
			}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["generated"] != float64(1) || parsed["failed"] != float64(0) {
		t.Fatalf("generated=%v failed=%v", parsed["generated"], parsed["failed"])
	}
	results, _ := parsed["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results len=%d", len(results))
	}
	row, _ := results[0].(map[string]any)
	if row["status"] != "success" || row["spreadsheetId"] != "new-sheet-1" {
		t.Fatalf("result=%#v", row)
	}
}

func TestExecute_SheetsEditMergeData_CopyTemplateNotFound_JSON(t *testing.T) {
	origSheets := newSheetsService
	origDrive := newDriveService
	t.Cleanup(func() {
		newSheetsService = origSheets
		newDriveService = origDrive
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "files/tpl1") && strings.Contains(r.URL.RawQuery, "parents"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "tpl1", "parents": []string{"folder1"}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/copy"):
			http.Error(w, `{"error":{"code":404,"message":"not found"}}`, http.StatusNotFound)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sheetSvc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("sheets.NewService: %v", err)
	}
	driveSvc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("drive.NewService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return sheetSvc, nil }
	newDriveService = func(context.Context, string) (*drive.Service, error) { return driveSvc, nil }

	dataFile := filepath.Join(t.TempDir(), "merge-data.json")
	if err := os.WriteFile(dataFile, []byte(`[{"quarter":"Q1"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{
				"--json", "--account", "a@b.com",
				"sheets", "edit", "merge-data", "tpl1",
				"--data-file", dataFile,
			}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["generated"] != float64(0) || parsed["failed"] != float64(1) {
		t.Fatalf("generated=%v failed=%v", parsed["generated"], parsed["failed"])
	}
	results, _ := parsed["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results len=%d", len(results))
	}
	row, _ := results[0].(map[string]any)
	if row["stage"] != "copy" || row["error_code"] != "template_not_found" {
		t.Fatalf("result=%#v", row)
	}
}

func TestExecute_SheetsEditMergeData_BatchUpdateFailure_JSON(t *testing.T) {
	origSheets := newSheetsService
	origDrive := newDriveService
	t.Cleanup(func() {
		newSheetsService = origSheets
		newDriveService = origDrive
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "files/tpl1") && strings.Contains(r.URL.RawQuery, "parents"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "tpl1", "parents": []string{"folder1"}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/copy"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "new-sheet-1", "parents": []string{"folder1"}})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "spreadsheets/new-sheet-1:batchUpdate"):
			http.Error(w, `{"error":{"code":500,"message":"boom"}}`, http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	sheetSvc, err := sheets.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("sheets.NewService: %v", err)
	}
	driveSvc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("drive.NewService: %v", err)
	}
	newSheetsService = func(context.Context, string) (*sheets.Service, error) { return sheetSvc, nil }
	newDriveService = func(context.Context, string) (*drive.Service, error) { return driveSvc, nil }

	dataFile := filepath.Join(t.TempDir(), "merge-data.json")
	if err := os.WriteFile(dataFile, []byte(`[{"quarter":"Q1"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{
				"--json", "--account", "a@b.com",
				"sheets", "edit", "merge-data", "tpl1",
				"--data-file", dataFile,
			}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["generated"] != float64(0) || parsed["failed"] != float64(1) {
		t.Fatalf("generated=%v failed=%v", parsed["generated"], parsed["failed"])
	}
	results, _ := parsed["results"].([]any)
	if len(results) != 1 {
		t.Fatalf("results len=%d", len(results))
	}
	row, _ := results[0].(map[string]any)
	if row["stage"] != "batch-update" || row["spreadsheetId"] != "new-sheet-1" {
		t.Fatalf("result=%#v", row)
	}
}
