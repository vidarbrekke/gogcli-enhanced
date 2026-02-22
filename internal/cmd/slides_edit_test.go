package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

func setupSlidesEditMockService(t *testing.T) {
	t.Helper()
	origSlides := newSlidesService
	t.Cleanup(func() { newSlidesService = origSlides })

	createCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/v1/presentations"):
			createCount++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"presentationId": "gen-" + string(rune('0'+createCount)),
				"title":          "Generated",
			})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, ":batchUpdate"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"presentationId": "p1",
				"replies": []any{
					map[string]any{
						"replaceAllText": map[string]any{"occurrencesChanged": 2},
						"createSlide":    map[string]any{"objectId": "slide_new"},
						"duplicateObject": map[string]any{
							"objectId": "slide_copy",
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc, err := slides.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("slides.NewService: %v", err)
	}
	newSlidesService = func(context.Context, string) (*slides.Service, error) { return svc, nil }
}

func setupDriveMergeMockService(t *testing.T) {
	t.Helper()
	origDrive := newDriveService
	t.Cleanup(func() { newDriveService = origDrive })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "files/tpl1"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "tpl1", "parents": []string{"template-folder"}})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "files/gen-1"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "gen-1", "parents": []string{"old-folder"}})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "files/gen-1"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "gen-1"})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "files/gen-1/export"):
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.4 mock"))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "upload/") && strings.Contains(r.URL.Path, "files"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "pdf1"})
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "files/gen-1"):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	svc, err := drive.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("drive.NewService: %v", err)
	}
	newDriveService = func(context.Context, string) (*drive.Service, error) { return svc, nil }
}

func TestExecute_SlidesEditBatch_JSON(t *testing.T) {
	setupSlidesEditMockService(t)
	withStdin(t, `{"requests":[{"replaceAllText":{"containsText":{"text":"x"},"replaceText":"y"}}]}`, func() {
		out := captureStdout(t, func() {
			if err := Execute([]string{"--json", "--account", "a@b.com", "slides", "edit", "batch", "p1", "--requests-file", "-"}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
		var parsed map[string]any
		if err := json.Unmarshal([]byte(out), &parsed); err != nil {
			t.Fatalf("parse json: %v", err)
		}
		if parsed["operations"] != float64(1) {
			t.Fatalf("operations=%v", parsed["operations"])
		}
	})
}

func TestExecute_SlidesEditReplaceText_JSON(t *testing.T) {
	setupSlidesEditMockService(t)
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "--account", "a@b.com", "slides", "edit", "replace-text", "p1", "--find", "old", "--replace", "new"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["occurrences"] != float64(2) {
		t.Fatalf("occurrences=%v", parsed["occurrences"])
	}
}

func TestExecute_SlidesEditCreateSlide_JSON(t *testing.T) {
	setupSlidesEditMockService(t)
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "--account", "a@b.com", "slides", "edit", "create-slide", "p1", "--layout", "TITLE", "--index", "0"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["slideId"] != "slide_new" {
		t.Fatalf("slideId=%v", parsed["slideId"])
	}
}

func TestExecute_SlidesEditDuplicateSlide_ValidateOnly_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "slides", "edit", "duplicate-slide", "p1", "--slide-id", "s1", "--count", "2", "--validate-only"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["validateOnly"] != true {
		t.Fatalf("validateOnly=%v", parsed["validateOnly"])
	}
}

func TestExecute_SlidesEditRefreshCharts_ValidateOnly_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "slides", "edit", "refresh-charts", "p1", "--all", "--validate-only"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["all"] != true {
		t.Fatalf("all=%v", parsed["all"])
	}
}

func TestExecute_SlidesEditReplaceImage_DryRun_Pretty_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "slides", "edit", "replace-image", "p1", "--object-id", "img1", "--source-url", "https://example.com/x.png", "--dry-run", "--pretty"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["dryRun"] != true {
		t.Fatalf("dryRun=%v", parsed["dryRun"])
	}
}

func TestExecute_SlidesEditInsertTable_OutputRequestFile_JSON(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "slides-insert-table.json")
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "slides", "edit", "insert-table", "p1", "--slide-id", "s1", "--rows", "2", "--columns", "2", "--validate-only", "--output-request-file", outFile}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(b), "\"createTable\"") {
		t.Fatalf("unexpected request file: %q", string(b))
	}
}

func TestExecute_SlidesEditInsertTable_DataFile_ValidateOnly_JSON(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "table-data.json")
	if err := os.WriteFile(dataFile, []byte(`[["A","B"],["C","D"]]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "slides", "edit", "insert-table", "p1", "--slide-id", "s1", "--rows", "2", "--columns", "2", "--data-file", dataFile, "--validate-only", "--pretty"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["filledCells"] != float64(4) {
		t.Fatalf("filledCells=%v", parsed["filledCells"])
	}
	norm, ok := parsed["normalizedRequest"].(string)
	if !ok || !strings.Contains(norm, "\"insertText\"") {
		t.Fatalf("normalizedRequest=%v", parsed["normalizedRequest"])
	}
}

func TestExecute_SlidesEditMergeData_ValidateOnly_JSON(t *testing.T) {
	dataFile := filepath.Join(t.TempDir(), "merge-data.json")
	if err := os.WriteFile(dataFile, []byte(`[{"name":"Alice"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "slides", "edit", "merge-data", "tpl1", "--data-file", dataFile, "--validate-only"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["recordCount"] != float64(1) {
		t.Fatalf("recordCount=%v", parsed["recordCount"])
	}
}

func TestExecute_SlidesEditMergeData_JSON(t *testing.T) {
	setupSlidesEditMockService(t)
	dataFile := filepath.Join(t.TempDir(), "merge-data.json")
	if err := os.WriteFile(dataFile, []byte(`[{"name":"Alice"},{"name":"Bob"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "--account", "a@b.com", "slides", "edit", "merge-data", "tpl1", "--data-file", dataFile}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["generated"] != float64(2) {
		t.Fatalf("generated=%v", parsed["generated"])
	}
}

func TestExecute_SlidesEditMergeData_ExportPDF_OutputFolder_JSON(t *testing.T) {
	setupSlidesEditMockService(t)
	setupDriveMergeMockService(t)
	dataFile := filepath.Join(t.TempDir(), "merge-data-export.json")
	if err := os.WriteFile(dataFile, []byte(`[{"name":"Alice"}]`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	out := captureStdout(t, func() {
		if err := Execute([]string{
			"--json", "--account", "a@b.com",
			"slides", "edit", "merge-data", "tpl1",
			"--data-file", dataFile,
			"--output-folder-id", "dest-folder",
			"--export-pdf",
		}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["generated"] != float64(1) {
		t.Fatalf("generated=%v parsed=%#v", parsed["generated"], parsed)
	}
	results, ok := parsed["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("results=%#v", parsed["results"])
	}
	row, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("result row=%#v", results[0])
	}
	if row["pdfFileId"] != "pdf1" {
		t.Fatalf("pdfFileId=%v", row["pdfFileId"])
	}
}

func TestExecute_SlidesEditRefreshCharts_Invalid_JSONErrorEnvelope(t *testing.T) {
	stderr := captureStderr(t, func() {
		_ = Execute([]string{"--json", "slides", "edit", "refresh-charts", "p1"})
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &parsed); err != nil {
		t.Fatalf("parse stderr json: %v; stderr=%q", err, stderr)
	}
	errObj, ok := parsed["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %#v", parsed)
	}
	if errObj["error_code"] != "invalid_argument" {
		t.Fatalf("error_code=%v", errObj["error_code"])
	}
}

func TestExecute_SlidesEditDeleteSlide_DryRun_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "slides", "edit", "delete-slide", "p1", "s1", "--dry-run"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["dryRun"] != true {
		t.Fatalf("dryRun=%v", parsed["dryRun"])
	}
}

func TestExecute_SlidesEditUpdateNotes_ValidateOnly_JSON(t *testing.T) {
	origSlides := newSlidesService
	t.Cleanup(func() { newSlidesService = origSlides })
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v1/presentations/p1") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"presentationId": "p1",
				"slides": []any{
					map[string]any{
						"objectId": "s1",
						"slideProperties": map[string]any{
							"notesPage": map[string]any{
								"notesProperties": map[string]any{
									"speakerNotesObjectId": "notes1",
								},
							},
						},
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	svc, err := slides.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("slides.NewService: %v", err)
	}
	newSlidesService = func(context.Context, string) (*slides.Service, error) { return svc, nil }
	out := captureStdout(t, func() {
		if err := Execute([]string{"--json", "--account", "a@b.com", "slides", "edit", "update-notes", "p1", "s1", "--notes", "hello", "--validate-only"}); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if parsed["validateOnly"] != true || parsed["slideId"] != "s1" {
		t.Fatalf("payload=%#v", parsed)
	}
}
