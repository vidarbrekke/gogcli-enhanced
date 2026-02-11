package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)

func TestExecute_DocsEditReplace_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/documents/d1:batchUpdate" {
			var req docs.BatchUpdateDocumentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(req.Requests) != 1 || req.Requests[0] == nil || req.Requests[0].ReplaceAllText == nil {
				t.Fatalf("expected one ReplaceAllText request")
			}
			got := req.Requests[0].ReplaceAllText
			if got.ContainsText == nil || got.ContainsText.Text != "hello" || !got.ContainsText.MatchCase {
				t.Fatalf("unexpected containsText: %#v", got.ContainsText)
			}
			if got.ReplaceText != "world" {
				t.Fatalf("unexpected replaceText: %q", got.ReplaceText)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "d1",
				"replies": []any{
					map[string]any{
						"replaceAllText": map[string]any{
							"occurrencesChanged": 2,
						},
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "docs", "edit", "replace", "d1", "hello", "world", "--match-case"}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["documentId"] != "d1" {
		t.Fatalf("documentId=%v", parsed["documentId"])
	}
	if parsed["occurrencesChanged"] != float64(2) {
		t.Fatalf("occurrencesChanged=%v", parsed["occurrencesChanged"])
	}
}

func TestExecute_DocsEditReplace_NotFound(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	err = Execute([]string{"--account", "a@b.com", "docs", "edit", "replace", "missing", "a", "b"})
	if err == nil || !strings.Contains(err.Error(), "doc not found or not a Google Doc") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestExecute_DocsEditAppend_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/documents/d1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "d1",
				"body": map[string]any{
					"content": []any{
						map[string]any{"startIndex": 0, "endIndex": 6},
					},
				},
			})
			return
		case r.Method == http.MethodPost && r.URL.Path == "/v1/documents/d1:batchUpdate":
			var req docs.BatchUpdateDocumentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(req.Requests) != 1 || req.Requests[0] == nil || req.Requests[0].InsertText == nil {
				t.Fatalf("expected one InsertText request")
			}
			got := req.Requests[0].InsertText
			if got.Location == nil || got.Location.Index != 5 {
				t.Fatalf("expected index=5, got=%#v", got.Location)
			}
			if got.Text != "tail" {
				t.Fatalf("unexpected text: %q", got.Text)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"documentId": "d1", "replies": []any{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "docs", "edit", "append", "d1", "tail"}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["documentId"] != "d1" {
		t.Fatalf("documentId=%v", parsed["documentId"])
	}
	if parsed["insertedChars"] != float64(4) {
		t.Fatalf("insertedChars=%v", parsed["insertedChars"])
	}
	if parsed["index"] != float64(5) {
		t.Fatalf("index=%v", parsed["index"])
	}
}

func TestExecute_DocsEditAppend_EmptyText(t *testing.T) {
	err := Execute([]string{"--account", "a@b.com", "docs", "edit", "append", "d1", "   "})
	if err == nil || !strings.Contains(err.Error(), "empty text") {
		t.Fatalf("expected empty text error, got: %v", err)
	}
}

func TestExecute_DocsEditInsert_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/documents/d1:batchUpdate" {
			var req docs.BatchUpdateDocumentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(req.Requests) != 1 || req.Requests[0] == nil || req.Requests[0].InsertText == nil {
				t.Fatalf("expected one InsertText request")
			}
			got := req.Requests[0].InsertText
			if got.Location == nil || got.Location.Index != 3 {
				t.Fatalf("expected index=3, got=%#v", got.Location)
			}
			if got.Text != "abc" {
				t.Fatalf("unexpected text: %q", got.Text)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"documentId": "d1", "replies": []any{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "docs", "edit", "insert", "d1", "abc", "--index", "3"}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["documentId"] != "d1" {
		t.Fatalf("documentId=%v", parsed["documentId"])
	}
	if parsed["insertedChars"] != float64(3) {
		t.Fatalf("insertedChars=%v", parsed["insertedChars"])
	}
	if parsed["index"] != float64(3) {
		t.Fatalf("index=%v", parsed["index"])
	}
}

func TestExecute_DocsEditInsert_InvalidIndex(t *testing.T) {
	err := Execute([]string{"--account", "a@b.com", "docs", "edit", "insert", "d1", "abc", "--index", "0"})
	if err == nil || !strings.Contains(err.Error(), "index must be >= 1") {
		t.Fatalf("expected invalid index error, got: %v", err)
	}
}

func TestExecute_DocsEditDelete_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/documents/d1:batchUpdate" {
			var req docs.BatchUpdateDocumentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(req.Requests) != 1 || req.Requests[0] == nil || req.Requests[0].DeleteContentRange == nil {
				t.Fatalf("expected one DeleteContentRange request")
			}
			got := req.Requests[0].DeleteContentRange
			if got.Range == nil || got.Range.StartIndex != 2 || got.Range.EndIndex != 6 {
				t.Fatalf("unexpected range: %#v", got.Range)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"documentId": "d1", "replies": []any{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "docs", "edit", "delete", "d1", "2", "6"}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["documentId"] != "d1" {
		t.Fatalf("documentId=%v", parsed["documentId"])
	}
	if parsed["deletedChars"] != float64(4) {
		t.Fatalf("deletedChars=%v", parsed["deletedChars"])
	}
}

func TestExecute_DocsEditDelete_InvalidRange(t *testing.T) {
	err := Execute([]string{"--account", "a@b.com", "docs", "edit", "delete", "d1", "5", "5"})
	if err == nil || !strings.Contains(err.Error(), "end must be > start") {
		t.Fatalf("expected invalid range error, got: %v", err)
	}
}

func TestExecute_DocsEditBatch_JSONFile(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/documents/d1:batchUpdate" {
			var req docs.BatchUpdateDocumentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(req.Requests) != 2 {
				t.Fatalf("expected 2 requests, got %d", len(req.Requests))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "d1",
				"replies":    []any{map[string]any{}, map[string]any{}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	tmp, err := os.CreateTemp(t.TempDir(), "docs-batch-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	payload := `{"requests":[{"insertText":{"location":{"index":1},"text":"Hello"}},{"replaceAllText":{"containsText":{"text":"Hello"},"replaceText":"Hi"}}]}`
	if _, err := tmp.WriteString(payload); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if execErr := Execute([]string{"--json", "--account", "a@b.com", "docs", "edit", "batch", "d1", "--requests-file", tmp.Name()}); execErr != nil {
				t.Fatalf("Execute: %v", execErr)
			}
		})
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("parse json: %v; out=%q", err, out)
	}
	if parsed["documentId"] != "d1" {
		t.Fatalf("documentId=%v", parsed["documentId"])
	}
	if parsed["operations"] != float64(2) {
		t.Fatalf("operations=%v", parsed["operations"])
	}
}

func TestExecute_DocsEditBatch_StdinNoRequests(t *testing.T) {
	withStdin(t, `{"requests":[]}`, func() {
		err := Execute([]string{"--account", "a@b.com", "docs", "edit", "batch", "d1", "--requests-file", "-"})
		if err == nil || !strings.Contains(err.Error(), "batch request has no operations") {
			t.Fatalf("expected no-operations error, got: %v", err)
		}
	})
}

func TestExecute_DocsEditBatch_Stdin_JSON(t *testing.T) {
	origDocs := newDocsService
	t.Cleanup(func() { newDocsService = origDocs })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/documents/d1:batchUpdate" {
			var req docs.BatchUpdateDocumentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if len(req.Requests) != 1 {
				t.Fatalf("expected 1 request, got %d", len(req.Requests))
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"documentId": "d1",
				"replies":    []any{map[string]any{}},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	docSvc, err := docs.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewDocsService: %v", err)
	}
	newDocsService = func(context.Context, string) (*docs.Service, error) { return docSvc, nil }

	withStdin(t, `{"requests":[{"insertText":{"location":{"index":1},"text":"Hi"}}]}`, func() {
		out := captureStdout(t, func() {
			_ = captureStderr(t, func() {
				if execErr := Execute([]string{"--json", "--account", "a@b.com", "docs", "edit", "batch", "d1", "--requests-file", "-"}); execErr != nil {
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
	})
}

