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

	gapi "google.golang.org/api/googleapi"
	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type SheetsEditSafetyFlags struct {
	DryRun            bool   `name:"dry-run" help:"Build request and print it without executing API call"`
	ValidateOnly      bool   `name:"validate-only" help:"Validate request payload locally without executing API call"`
	Pretty            bool   `name:"pretty" help:"Include normalized pretty-printed request JSON in output"`
	OutputRequestFile string `name:"output-request-file" help:"Write normalized request JSON to this file (use '-' for stdout)"`
	ExecuteFromFile   string `name:"execute-from-file" help:"Execute request JSON from this file (bypasses direct command input)"`
}

type sheetsEditError struct {
	Operation     string
	SpreadsheetID string
	ErrorCode     string
	Message       string
	HTTPStatus    int
	GoogleReason  string
	RequestIndex  *int
	Cause         error
}

func (e *sheetsEditError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "sheets edit failed"
}

func (e *sheetsEditError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *sheetsEditError) JSONErrorFields() map[string]any {
	if e == nil {
		return map[string]any{}
	}
	fields := map[string]any{
		"error_code":     e.ErrorCode,
		"operation":      e.Operation,
		"spreadsheet_id": e.SpreadsheetID,
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

func newSheetsEditError(op, spreadsheetID, code, msg string, cause error) error {
	e := &sheetsEditError{
		Operation:     op,
		SpreadsheetID: strings.TrimSpace(spreadsheetID),
		ErrorCode:     strings.TrimSpace(code),
		Message:       strings.TrimSpace(msg),
		Cause:         cause,
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

func isSheetsNotFound(err error) bool {
	var apiErr *gapi.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Code == http.StatusNotFound
}

func sheetsDryRunOutput(ctx context.Context, u *ui.UI, spreadsheetID string, req any, extra map[string]any, includePretty bool) error {
	payload := map[string]any{
		"dryRun":        true,
		"spreadsheetId": spreadsheetID,
		"request":       req,
	}
	for k, v := range extra {
		payload[k] = v
	}
	if includePretty {
		if hash, err := sheetsRequestHash(req); err == nil {
			payload["requestHash"] = hash
		}
		if norm, err := sheetsNormalizedRequestString(req); err == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("dry-run\ttrue")
	u.Out().Printf("id\t%s", spreadsheetID)
	return nil
}

func sheetsRequestOperationCount(r *sheets.Request) int {
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

func sheetsRequestOperationName(r *sheets.Request) string {
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

func sheetsMaybeWriteNormalizedRequest(path string, req any) error {
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

func sheetsNormalizedRequestForOutput(ctx context.Context, path string, req any) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || req == nil {
		return "", nil
	}
	if path == "-" && outfmt.IsJSON(ctx) {
		return sheetsNormalizedRequestString(req)
	}
	if err := sheetsMaybeWriteNormalizedRequest(path, req); err != nil {
		return "", err
	}
	return "", nil
}

func sheetsNormalizedRequestString(req any) (string, error) {
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

func sheetsRequestHash(req any) (string, error) {
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
