package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"google.golang.org/api/docs/v1"
	gapi "google.golang.org/api/googleapi"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// AgenticEditSafetyFlags provides common safety flags across all edit commands.
// Docs, Sheets, and Slides edit commands embed this struct.
type AgenticEditSafetyFlags struct {
	DryRun            bool   `name:"dry-run" help:"Build request and print it without executing API call"`
	ValidateOnly      bool   `name:"validate-only" help:"Validate request payload locally without executing API call"`
	Pretty            bool   `name:"pretty" help:"Include normalized pretty-printed request JSON in output"`
	OutputRequestFile string `name:"output-request-file" help:"Write normalized request JSON to this file (use '-' for stdout)"`
	ExecuteFromFile   string `name:"execute-from-file" help:"Execute request JSON from this file (bypasses direct command input)"`
	RequireRevision   string `name:"require-revision" help:"Require specific revision ID to prevent conflicts (Docs only)"`
}

// EditError provides structured error metadata for JSON error envelopes.
// Works across Docs, Sheets, and Slides services.
type EditError struct {
	Service      string // "docs", "sheets", "slides"
	Operation    string
	ResourceID   string // doc_id, spreadsheet_id, presentation_id
	ErrorCode    string
	Message      string
	HTTPStatus   int
	GoogleReason string
	RequestIndex *int
	Cause        error
}

func (e *EditError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return fmt.Sprintf("%s edit failed", e.Service)
}

func (e *EditError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *EditError) JSONErrorFields() map[string]any {
	if e == nil {
		return map[string]any{}
	}
	fields := map[string]any{
		"error_code":  e.ErrorCode,
		"service":     e.Service,
		"operation":   e.Operation,
		"resource_id": e.ResourceID,
	}
	if e.HTTPStatus > 0 {
		fields["http_status"] = e.HTTPStatus
	}
	if strings.TrimSpace(e.GoogleReason) != "" {
		fields["google_reason"] = e.GoogleReason
	}
	if e.RequestIndex != nil {
		fields["request_index"] = *e.RequestIndex
	}
	return fields
}

// NewEditError creates a standardized edit error with service context.
func NewEditError(service, operation, resourceID, code, msg string, cause error) error {
	e := &EditError{
		Service:    service,
		Operation:  operation,
		ResourceID: strings.TrimSpace(resourceID),
		ErrorCode:  strings.TrimSpace(code),
		Message:    strings.TrimSpace(msg),
		Cause:      cause,
	}
	var apiErr *gapi.Error
	if errors.As(cause, &apiErr) {
		e.HTTPStatus = apiErr.Code
		if len(apiErr.Errors) > 0 && strings.TrimSpace(apiErr.Errors[0].Reason) != "" {
			e.GoogleReason = strings.TrimSpace(apiErr.Errors[0].Reason)
		}
	}
	return e
}

// IsNotFound checks if an error is a 404 Not Found from Google API.
func IsNotFound(err error) bool {
	var apiErr *gapi.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Code == http.StatusNotFound
}

// --- Request Hash Utilities ---

// RequestHash generates a deterministic SHA256 hash of any request object.
func RequestHash(req any) (string, error) {
	if req == nil {
		return "", errors.New("nil request")
	}
	b, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// --- Normalized Request Utilities ---

// NormalizedRequestString returns a pretty-printed JSON string of any request.
func NormalizedRequestString(req any) (string, error) {
	if req == nil {
		return "", errors.New("nil request")
	}
	pretty, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return "", err
	}
	pretty = append(pretty, '\n')
	return string(pretty), nil
}

// MaybeWriteNormalizedRequest writes normalized request JSON to file or stdout.
// Returns nil if path is empty or request is nil.
func MaybeWriteNormalizedRequest(path string, req any) error {
	path = strings.TrimSpace(path)
	if path == "" || req == nil {
		return nil
	}
	pretty, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return err
	}
	pretty = append(pretty, '\n')
	if path == "-" {
		_, err = os.Stdout.Write(pretty)
		return err
	}
	return os.WriteFile(path, pretty, 0o600)
}

// NormalizedRequestForOutput handles request file output, respecting JSON context.
// For JSON mode with stdout path, returns string; otherwise writes to file.
func NormalizedRequestForOutput(ctx context.Context, path string, req any) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || req == nil {
		return "", nil
	}
	if path == "-" && outfmt.IsJSON(ctx) {
		return NormalizedRequestString(req)
	}
	if err := MaybeWriteNormalizedRequest(path, req); err != nil {
		return "", err
	}
	return "", nil
}

// --- Request Operation Reflection Utilities ---

// RequestOperationCount returns the number of operation fields set in a request struct.
// Uses reflection - skip internal/GeneratedName fields.
// Used for validation (exactly one operation per request in batch scenarios).
func RequestOperationCount(r any) int {
	if r == nil {
		return 0
	}
	v := reflect.ValueOf(r)
	// Dereference if pointer
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0
	}
	t := v.Type()
	count := 0
	for i := range t.NumField() {
		field := t.Field(i)
		name := field.Name
		// Skip internal Google API fields
		if name == "ForceSendFields" || name == "NullFields" || name == "ServerResponse" ||
			name == "Header" || name == "XXX_NoUnkeyedLiteral" || strings.HasPrefix(name, "XXX_") {
			continue
		}
		fv := v.Field(i)
		switch fv.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
			if !fv.IsNil() {
				count++
			}
		}
	}
	return count
}

// RequestOperationName returns the name of the first set operation field in a request struct.
func RequestOperationName(r any) string {
	if r == nil {
		return ""
	}
	v := reflect.ValueOf(r)
	// Dereference if pointer
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		name := field.Name
		// Skip internal Google API fields
		if name == "ForceSendFields" || name == "NullFields" || name == "ServerResponse" ||
			name == "Header" || name == "XXX_NoUnkeyedLiteral" || strings.HasPrefix(name, "XXX_") {
			continue
		}
		fv := v.Field(i)
		switch fv.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
			if !fv.IsNil() {
				return name
			}
		}
	}
	return ""
}

// --- Dry Run Output Utilities ---

// DryRunOutput builds and outputs a dry-run payload for any service.
// Returns JSON output if in JSON mode, otherwise prints key-value pairs.
func DryRunOutput(ctx context.Context, u *ui.UI, service, resourceID string, req any, extra map[string]any, includePretty bool) error {
	payload := map[string]any{
		"dryRun":     true,
		"service":    service,
		"resourceId": resourceID,
		"request":    req,
	}
	for k, v := range extra {
		payload[k] = v
	}
	if includePretty {
		if hash, err := RequestHash(req); err == nil {
			payload["requestHash"] = hash
		}
		if norm, err := NormalizedRequestString(req); err == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	// Human-readable output
	u.Out().Printf("dry-run\ttrue")
	u.Out().Printf("service\t%s", service)
	u.Out().Printf("id\t%s", resourceID)
	return nil
}

// DocsDryRunOutput is a backward-compatible wrapper for Docs dry-run output.
// Uses the new shared DryRunOutput under the hood.
func DocsDryRunOutput(ctx context.Context, u *ui.UI, docID string, req any, extra map[string]any) error {
	return DryRunOutput(ctx, u, "docs", docID, req, extra, false)
}

// DocsDryRunOutputWithOpts includes pretty-printed request info.
func DocsDryRunOutputWithOpts(ctx context.Context, u *ui.UI, docID string, req any, extra map[string]any, includePretty bool) error {
	return DryRunOutput(ctx, u, "docs", docID, req, extra, includePretty)
}

// SheetsDryRunOutput is a wrapper for Sheets dry-run output.
func SheetsDryRunOutput(ctx context.Context, u *ui.UI, spreadsheetID string, req any, extra map[string]any, includePretty bool) error {
	return DryRunOutput(ctx, u, "sheets", spreadsheetID, req, extra, includePretty)
}

// ==== BACKWARD-COMPATIBILITY HELPERS FOR LEGACY DOCS COMMANDS ====

// docsRequestHash hashes a normalized BatchUpdateDocumentRequest for change detection.
// Used by legacy docs edit commands; new commands use RequestHash directly.
func docsRequestHash(req any) (string, error) {
	return RequestHash(req)
}

// docsNormalizedRequestString returns a normalized, pretty-printed JSON string of the request.
// Used by legacy docs edit commands; new commands use NormalizedRequestString directly.
func docsNormalizedRequestString(req any) (string, error) {
	return NormalizedRequestString(req)
}

// docsMaybeWriteNormalizedRequest writes normalized request JSON to file or stdout.
// Used by legacy docs edit commands; new commands use NormalizedRequestForOutput directly.
func docsMaybeWriteNormalizedRequest(path string, req any) error {
	path = strings.TrimSpace(path)
	if path == "" || req == nil {
		return nil
	}
	pretty, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return err
	}
	pretty = append(pretty, '\n')
	if path == "-" {
		_, err = os.Stdout.Write(pretty)
		return err
	}
	return os.WriteFile(path, pretty, 0o600)
}

// docsNormalizedRequestForOutput handles conditional output of normalized requests.
// Used by legacy docs edit commands; new commands use NormalizedRequestForOutput directly.
func docsNormalizedRequestForOutput(ctx context.Context, path string, req any) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || req == nil {
		return "", nil
	}
	if path == "-" && outfmt.IsJSON(ctx) {
		return NormalizedRequestString(req)
	}
	if err := docsMaybeWriteNormalizedRequest(path, req); err != nil {
		return "", err
	}
	return "", nil
}

// applyDocsEditSafety applies safety flags (like required revision ID) to a BatchUpdateDocumentRequest.
// Used by legacy docs edit commands; new commands should use the shared pattern directly.
func applyDocsEditSafety(req any, safety AgenticEditSafetyFlags) {
	if req == nil {
		return
	}
	// Handle Docs BatchUpdateDocumentRequest
	if docReq, ok := req.(*docs.BatchUpdateDocumentRequest); ok {
		requiredRevision := strings.TrimSpace(safety.RequireRevision)
		if requiredRevision == "" {
			return
		}
		docReq.WriteControl = &docs.WriteControl{RequiredRevisionId: requiredRevision}
	}
}

// docsDryRunOutput outputs dry-run results with minimal details.
// Used by legacy docs edit commands; new commands use DryRunOutput directly.
func docsDryRunOutput(ctx context.Context, u *ui.UI, docID string, req any, extra map[string]any) error {
	return DryRunOutput(ctx, u, "docs", docID, req, extra, false)
}

// docsDryRunOutputWithOpts outputs dry-run results with optional pretty-printing.
// Used by legacy docs edit commands; new commands use DryRunOutput with includePretty directly.
func docsDryRunOutputWithOpts(ctx context.Context, u *ui.UI, docID string, req any, extra map[string]any, includePretty bool) error {
	return DryRunOutput(ctx, u, "docs", docID, req, extra, includePretty)
}
