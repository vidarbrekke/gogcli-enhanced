package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"reflect"
	"strings"

	"google.golang.org/api/docs/v1"
	gapi "google.golang.org/api/googleapi"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// docsEditError provides structured error metadata for JSON error envelopes.
type docsEditError struct {
	Operation    string
	DocID        string
	ErrorCode    string
	Message      string
	HTTPStatus   int
	GoogleReason string
	RequestIndex *int
	Cause        error
}

func (e *docsEditError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "docs edit failed"
}

func (e *docsEditError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *docsEditError) JSONErrorFields() map[string]any {
	if e == nil {
		return map[string]any{}
	}
	fields := map[string]any{
		"error_code": e.ErrorCode,
		"operation":  e.Operation,
		"doc_id":     e.DocID,
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

func newDocsEditError(op, docID, code, msg string, cause error) error {
	e := &docsEditError{
		Operation: op,
		DocID:     strings.TrimSpace(docID),
		ErrorCode: strings.TrimSpace(code),
		Message:   strings.TrimSpace(msg),
		Cause:     cause,
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

func isDocsNotFound(err error) bool {
	var apiErr *gapi.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Code == http.StatusNotFound
}

func docsAppendIndex(doc *docs.Document) int64 {
	if doc == nil || doc.Body == nil || len(doc.Body.Content) == 0 {
		return 1
	}
	last := doc.Body.Content[len(doc.Body.Content)-1]
	if last == nil || last.EndIndex <= 1 {
		return 1
	}
	return last.EndIndex - 1
}

func applyDocsEditSafety(req *docs.BatchUpdateDocumentRequest, safety DocsEditSafetyFlags) {
	if req == nil {
		return
	}
	requiredRevision := strings.TrimSpace(safety.RequireRevision)
	if requiredRevision == "" {
		return
	}
	req.WriteControl = &docs.WriteControl{RequiredRevisionId: requiredRevision}
}

func docsDryRunOutput(ctx context.Context, u *ui.UI, docID string, req *docs.BatchUpdateDocumentRequest, extra map[string]any) error {
	return docsDryRunOutputWithOpts(ctx, u, docID, req, extra, false)
}

func docsDryRunOutputWithOpts(ctx context.Context, u *ui.UI, docID string, req *docs.BatchUpdateDocumentRequest, extra map[string]any, includePretty bool) error {
	payload := map[string]any{
		"dryRun":     true,
		"documentId": docID,
		"request":    req,
	}
	for k, v := range extra {
		payload[k] = v
	}
	if includePretty {
		if hash, err := docsRequestHash(req); err == nil {
			payload["requestHash"] = hash
		}
		if norm, err := docsNormalizedRequestString(req); err == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("dry-run\ttrue")
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("operations\t%d", len(req.Requests))
	if req.WriteControl != nil && strings.TrimSpace(req.WriteControl.RequiredRevisionId) != "" {
		u.Out().Printf("required-revision\t%s", req.WriteControl.RequiredRevisionId)
	}
	raw, err := json.Marshal(req)
	if err == nil {
		u.Out().Printf("request\t%s", string(raw))
	}
	return nil
}

// docsRequestOperationCount returns the number of operation fields set in a docs.Request.
// Used for validation (exactly one operation per request). Reflection is safe here: input
// is our own BatchUpdateDocumentRequest, not user-controlled.
func docsRequestOperationCount(r *docs.Request) int {
	if r == nil {
		return 0
	}
	v := reflect.ValueOf(*r)
	t := reflect.TypeOf(*r)
	count := 0
	for i := range t.NumField() {
		name := t.Field(i).Name
		if name == "ForceSendFields" || name == "NullFields" || name == "ServerResponse" {
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

func docsRequestOperationName(r *docs.Request) string {
	if r == nil {
		return ""
	}
	v := reflect.ValueOf(*r)
	t := reflect.TypeOf(*r)
	for i := range t.NumField() {
		name := t.Field(i).Name
		if name == "ForceSendFields" || name == "NullFields" || name == "ServerResponse" {
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

func docsMaybeWriteNormalizedRequest(path string, req *docs.BatchUpdateDocumentRequest) error {
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

func docsNormalizedRequestString(req *docs.BatchUpdateDocumentRequest) (string, error) {
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

func docsRequestHash(req *docs.BatchUpdateDocumentRequest) (string, error) {
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
