package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type DocsBatchCmd struct {
	DocID        string              `arg:"" name:"docId" help:"Doc ID"`
	RequestsFile string              `name:"requests-file" help:"Path to JSON request body, or '-' for stdin" default:"-"`
	Safety       DocsEditSafetyFlags `embed:""`
}

func (c *DocsBatchCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return newDocsEditError("batch", docID, "invalid_argument", "empty docId", usage("empty docId"))
	}
	requestsFile := strings.TrimSpace(c.RequestsFile)
	executeFromFile := strings.TrimSpace(c.Safety.ExecuteFromFile)
	if executeFromFile != "" && requestsFile != "-" && requestsFile != "" {
		return newDocsEditError("batch", docID, "invalid_argument", "cannot combine --execute-from-file with --requests-file", usage("cannot combine --execute-from-file with --requests-file"))
	}
	if executeFromFile != "" {
		requestsFile = executeFromFile
	}
	if requestsFile == "" {
		return newDocsEditError("batch", docID, "invalid_argument", "empty requests-file", usage("empty requests-file"))
	}

	var reader io.Reader = os.Stdin
	if requestsFile != "-" {
		f, openErr := os.Open(requestsFile) //nolint:gosec // user-provided path
		if openErr != nil {
			return newDocsEditError("batch", docID, "input_open_failed", "open requests-file failed", openErr)
		}
		defer f.Close()
		reader = f
	}

	var req docs.BatchUpdateDocumentRequest
	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		return newDocsEditError("batch", docID, "invalid_json", "decode requests JSON failed", err)
	}
	if len(req.Requests) == 0 {
		return newDocsEditError("batch", docID, "invalid_argument", "batch request has no operations", usage("batch request has no operations"))
	}
	for i, r := range req.Requests {
		if RequestOperationCount(r) != 1 {
			idx := i
			err := newDocsEditError("batch", docID, "invalid_request", fmt.Sprintf("request[%d] must set exactly one operation field", i), usage(fmt.Sprintf("request[%d] must set exactly one operation field", i)))
			if de, ok := err.(*EditError); ok {
				de.RequestIndex = &idx
			}
			return err
		}
	}
	applyDocsEditSafety(&req, c.Safety)
	requestHash, hashErr := RequestHash(&req)
	if hashErr != nil {
		return newDocsEditError("batch", docID, "invalid_request", "failed to hash normalized request", hashErr)
	}
	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, &req)
	if normErr != nil {
		return newDocsEditError("batch", docID, "output_write_failed", "write normalized request failed", normErr)
	}
	requestKinds := make([]string, 0, len(req.Requests))
	for _, r := range req.Requests {
		requestKinds = append(requestKinds, RequestOperationName(r))
	}
	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly": true,
			"valid":        true,
			"documentId":   docID,
			"operations":   len(req.Requests),
			"requestKinds": requestKinds,
			"requestHash":  requestHash,
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(req); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if req.WriteControl != nil && strings.TrimSpace(req.WriteControl.RequiredRevisionId) != "" {
			payload["requiredRevisionId"] = req.WriteControl.RequiredRevisionId
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", docID)
		u.Out().Printf("operations\t%d", len(req.Requests))
		if c.Safety.Pretty {
			pretty, prettyErr := json.MarshalIndent(req, "", "  ")
			if prettyErr == nil {
				u.Out().Printf("pretty-request\t%s", string(pretty))
			}
		}
		return nil
	}
	if isEditDryRun(flags, c.Safety) {
		return DryRunOutput(ctx, u, "docs", docID, &req, map[string]any{
			"operations":        len(req.Requests),
			"requestKinds":      requestKinds,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, false)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return newDocsEditError("batch", docID, "service_init_failed", "create docs service failed", err)
	}
	resp, err := svc.Documents.BatchUpdate(docID, &req).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return newDocsEditError("batch", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return newDocsEditError("batch", docID, "api_error", "batch update failed", err)
	}

	operations := len(req.Requests)
	if outfmt.IsJSON(ctx) {
		payload := map[string]any{
			"documentId": docID,
			"operations": operations,
			"replies":    len(resp.Replies),
		}
		if normalizedForJSON != "" {
			payload["normalizedRequest"] = normalizedForJSON
		}
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("operations\t%d", operations)
	u.Out().Printf("replies\t%d", len(resp.Replies))
	return nil
}

type DocsDeleteCmd struct {
	DocID      string                 `arg:"" name:"docId" help:"Doc ID"`
	StartIndex int64                  `arg:"" name:"start" help:"Start index (inclusive, 1-based)"`
	EndIndex   int64                  `arg:"" name:"end" help:"End index (exclusive)"`
	Safety     AgenticEditSafetyFlags `embed:""`
}

func (c *DocsDeleteCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return NewEditError("docs", "delete", docID, "invalid_argument", "empty docId", nil)
	}
	if c.StartIndex < 1 {
		return NewEditError("docs", "delete", docID, "invalid_argument", "start must be >= 1", nil)
	}
	if c.EndIndex <= c.StartIndex {
		return NewEditError("docs", "delete", docID, "invalid_argument", "end must be > start", nil)
	}
	if !isEditDryRun(flags, c.Safety) && !outfmt.IsJSON(ctx) && (flags == nil || !flags.Force) {
		return NewEditError("docs", "delete", docID, "confirmation_required", "delete is destructive; rerun with --force or use --dry-run", nil)
	}

	// Build the batch request
	req := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				DeleteContentRange: &docs.DeleteContentRangeRequest{
					Range: &docs.Range{
						StartIndex: c.StartIndex,
						EndIndex:   c.EndIndex,
					},
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("docs", "delete", docID, "invalid_json", "decode execute-from-file failed", err)
	}
	applyDocsEditSafety(req, c.Safety)

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("docs", "delete", docID, "invalid_request", "failed to hash request", hashErr)
	}

	deletedChars := c.EndIndex - c.StartIndex
	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("docs", "delete", docID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly": true,
			"valid":        true,
			"documentId":   docID,
			"deletedChars": deletedChars,
			"requestHash":  requestHash,
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(req); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", docID)
		u.Out().Printf("deleted\t%d", deletedChars)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return DryRunOutput(ctx, u, "docs", docID, req, map[string]any{
			"deletedChars":      deletedChars,
			"startIndex":        c.StartIndex,
			"endIndex":          c.EndIndex,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return NewEditError("docs", "delete", docID, "service_init_failed", "create docs service failed", err)
	}

	_, err = svc.Documents.BatchUpdate(docID, req).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return NewEditError("docs", "delete", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return NewEditError("docs", "delete", docID, "api_error", "delete failed", err)
	}

	payload := map[string]any{
		"documentId":   docID,
		"deletedChars": deletedChars,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		payload["requestHash"] = requestHash
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("deleted\t%d", deletedChars)
	return nil
}

type DocsInsertCmd struct {
	DocID  string                 `arg:"" name:"docId" help:"Doc ID"`
	Text   string                 `arg:"" name:"text" help:"Text to insert"`
	Index  int64                  `name:"index" help:"Insertion index (1-based)" default:"1"`
	Safety AgenticEditSafetyFlags `embed:""`
}

func (c *DocsInsertCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return NewEditError("docs", "insert", docID, "invalid_argument", "empty docId", nil)
	}
	text := strings.TrimSpace(c.Text)
	if text == "" {
		return NewEditError("docs", "insert", docID, "invalid_argument", "empty text", nil)
	}
	if c.Index < 1 {
		return NewEditError("docs", "insert", docID, "invalid_argument", "index must be >= 1", nil)
	}

	// Build the batch request
	req := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: c.Index},
					Text:     text,
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("docs", "insert", docID, "invalid_json", "decode execute-from-file failed", err)
	}
	applyDocsEditSafety(req, c.Safety)

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("docs", "insert", docID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("docs", "insert", docID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":  true,
			"valid":         true,
			"documentId":    docID,
			"insertedChars": len(text),
			"index":         c.Index,
			"requestHash":   requestHash,
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(req); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", docID)
		u.Out().Printf("inserted\t%d", len(text))
		u.Out().Printf("index\t%d", c.Index)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return DryRunOutput(ctx, u, "docs", docID, req, map[string]any{
			"insertedChars":     len(text),
			"index":             c.Index,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return NewEditError("docs", "insert", docID, "service_init_failed", "create docs service failed", err)
	}

	_, err = svc.Documents.BatchUpdate(docID, req).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return NewEditError("docs", "insert", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return NewEditError("docs", "insert", docID, "api_error", "insert failed", err)
	}

	payload := map[string]any{
		"documentId":    docID,
		"insertedChars": len(text),
		"index":         c.Index,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		payload["requestHash"] = requestHash
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("inserted\t%d", len(text))
	u.Out().Printf("index\t%d", c.Index)
	return nil
}

type DocsAppendCmd struct {
	DocID  string              `arg:"" name:"docId" help:"Doc ID"`
	Text   string              `arg:"" name:"text" help:"Text to append"`
	Safety DocsEditSafetyFlags `embed:""`
}

func (c *DocsAppendCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return newDocsEditError("append", docID, "invalid_argument", "empty docId", usage("empty docId"))
	}
	text := strings.TrimSpace(c.Text)
	if text == "" {
		return newDocsEditError("append", docID, "invalid_argument", "empty text", usage("empty text"))
	}

	svc, err := newDocsService(ctx, account)
	if err != nil {
		return newDocsEditError("append", docID, "service_init_failed", "create docs service failed", err)
	}

	doc, err := svc.Documents.Get(docID).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return newDocsEditError("append", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return newDocsEditError("append", docID, "api_error", "fetch document failed", err)
	}
	index := docsAppendIndex(doc)

	req := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: index},
					Text:     text,
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return newDocsEditError("append", docID, "invalid_json", "decode execute-from-file failed", err)
	}
	applyDocsEditSafety(req, c.Safety)
	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newDocsEditError("append", docID, "output_write_failed", "write normalized request failed", normErr)
	}
	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return newDocsEditError("append", docID, "invalid_request", "failed to hash request", hashErr)
	}
	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":  true,
			"valid":         true,
			"documentId":    docID,
			"insertedChars": len(text),
			"index":         index,
			"requestHash":   requestHash,
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(req); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", docID)
		u.Out().Printf("appended\t%d", len(text))
		u.Out().Printf("index\t%d", index)
		return nil
	}
	if isEditDryRun(flags, c.Safety) {
		return DryRunOutput(ctx, u, "docs", docID, req, map[string]any{
			"insertedChars":     len(text),
			"index":             index,
			"normalizedRequest": normalizedForJSON,
			"requestHash":       requestHash,
		}, c.Safety.Pretty)
	}
	if _, err := svc.Documents.BatchUpdate(docID, req).Context(ctx).Do(); err != nil {
		if isDocsNotFound(err) {
			return newDocsEditError("append", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return newDocsEditError("append", docID, "api_error", "append failed", err)
	}

	payload := map[string]any{
		"documentId":    docID,
		"insertedChars": len(text),
		"index":         index,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		payload["requestHash"] = requestHash
		if norm, normErr := NormalizedRequestString(req); normErr == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("appended\t%d", len(text))
	u.Out().Printf("index\t%d", index)
	return nil
}

type DocsReplaceCmd struct {
	DocID     string                 `arg:"" name:"docId" help:"Doc ID"`
	Find      string                 `name:"find" help:"Text to find"`
	Replace   string                 `name:"replace" help:"Replacement text"`
	MatchCase bool                   `name:"match-case" help:"Case-sensitive matching"`
	Safety    AgenticEditSafetyFlags `embed:""`
}

func (c *DocsReplaceCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return NewEditError("docs", "replace-text", docID, "invalid_argument", "empty docId", nil)
	}
	find := strings.TrimSpace(c.Find)
	if find == "" {
		return NewEditError("docs", "replace-text", docID, "invalid_argument", "empty find", nil)
	}

	// Build the batch request
	req := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				ReplaceAllText: &docs.ReplaceAllTextRequest{
					ContainsText: &docs.SubstringMatchCriteria{
						Text:      find,
						MatchCase: c.MatchCase,
					},
					ReplaceText: c.Replace,
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("docs", "replace-text", docID, "invalid_json", "decode execute-from-file failed", err)
	}
	applyDocsEditSafety(req, c.Safety)

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("docs", "replace-text", docID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("docs", "replace-text", docID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly": true,
			"valid":        true,
			"documentId":   docID,
			"find":         find,
			"replace":      c.Replace,
			"matchCase":    c.MatchCase,
			"requestHash":  requestHash,
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(req); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", docID)
		u.Out().Printf("find\t%s", find)
		u.Out().Printf("replace\t%s", c.Replace)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return DryRunOutput(ctx, u, "docs", docID, req, map[string]any{
			"find":              find,
			"replace":           c.Replace,
			"matchCase":         c.MatchCase,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return NewEditError("docs", "replace-text", docID, "service_init_failed", "create docs service failed", err)
	}

	resp, err := svc.Documents.BatchUpdate(docID, req).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return NewEditError("docs", "replace-text", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return NewEditError("docs", "replace-text", docID, "api_error", "replace failed", err)
	}

	var occurrences int64
	if resp != nil && len(resp.Replies) > 0 && resp.Replies[0] != nil && resp.Replies[0].ReplaceAllText != nil {
		occurrences = resp.Replies[0].ReplaceAllText.OccurrencesChanged
	}

	payload := map[string]any{
		"documentId":         docID,
		"occurrencesChanged": occurrences,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		payload["requestHash"] = requestHash
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("replaced\t%d", occurrences)
	return nil
}

type DocsInsertTableCmd struct {
	DocID  string                 `arg:"" name:"docId" help:"Doc ID"`
	Rows   int64                  `name:"rows" help:"Number of rows in the table" default:"2"`
	Cols   int64                  `name:"cols" help:"Number of columns in the table" default:"2"`
	Index  int64                  `name:"index" help:"Index where table should be inserted (1-based); omit to insert at end"`
	Safety AgenticEditSafetyFlags `embed:""`
}

func (c *DocsInsertTableCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return NewEditError("docs", "insert-table", docID, "invalid_argument", "empty docId", nil)
	}
	if c.Rows < 1 {
		return NewEditError("docs", "insert-table", docID, "invalid_argument", "rows must be >= 1", nil)
	}
	if c.Cols < 1 {
		return NewEditError("docs", "insert-table", docID, "invalid_argument", "cols must be >= 1", nil)
	}

	// Build the batch request with InsertTable operation
	req := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertTable: &docs.InsertTableRequest{
					Rows:    c.Rows,
					Columns: c.Cols,
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("docs", "insert-table", docID, "invalid_json", "decode execute-from-file failed", err)
	}

	// Set location: either at index or at end of document
	if c.Index > 0 {
		req.Requests[0].InsertTable.Location = &docs.Location{Index: c.Index}
	} else {
		req.Requests[0].InsertTable.EndOfSegmentLocation = &docs.EndOfSegmentLocation{}
	}

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("docs", "insert-table", docID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("docs", "insert-table", docID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly": true,
			"valid":        true,
			"documentId":   docID,
			"rows":         c.Rows,
			"cols":         c.Cols,
			"requestHash":  requestHash,
		}
		if c.Index > 0 {
			payload["index"] = c.Index
		} else {
			payload["position"] = "end"
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(req); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", docID)
		u.Out().Printf("rows\t%d", c.Rows)
		u.Out().Printf("cols\t%d", c.Cols)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		position := "end"
		if c.Index > 0 {
			position = fmt.Sprintf("index %d", c.Index)
		}
		return DryRunOutput(ctx, u, "docs", docID, req, map[string]any{
			"rows":              c.Rows,
			"cols":              c.Cols,
			"position":          position,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return NewEditError("docs", "insert-table", docID, "service_init_failed", "create docs service failed", err)
	}

	_, err = svc.Documents.BatchUpdate(docID, req).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return NewEditError("docs", "insert-table", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return NewEditError("docs", "insert-table", docID, "api_error", "insert table failed", err)
	}

	payload := map[string]any{
		"documentId": docID,
		"rows":       c.Rows,
		"cols":       c.Cols,
	}
	if c.Index > 0 {
		payload["index"] = c.Index
	} else {
		payload["position"] = "end"
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		payload["requestHash"] = requestHash
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("table-inserted\ttrue")
	u.Out().Printf("rows\t%d", c.Rows)
	u.Out().Printf("cols\t%d", c.Cols)
	return nil
}

type DocsReplaceImageCmd struct {
	DocID         string                 `arg:"" name:"docId" help:"Doc ID"`
	ImageID       string                 `name:"image-id" help:"ID of existing image to replace"`
	URI           string                 `name:"uri" help:"URI of new image"`
	ReplaceMethod string                 `name:"replace-method" help:"Replace method: CENTER_CROP or UNSPECIFIED" default:"UNSPECIFIED"`
	TabID         string                 `name:"tab-id" help:"Tab ID containing the image (omit for first tab)"`
	Safety        AgenticEditSafetyFlags `embed:""`
}

func (c *DocsReplaceImageCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return NewEditError("docs", "replace-image", docID, "invalid_argument", "empty docId", nil)
	}
	imageID := strings.TrimSpace(c.ImageID)
	if imageID == "" {
		return NewEditError("docs", "replace-image", docID, "invalid_argument", "empty image-id", nil)
	}
	uri := strings.TrimSpace(c.URI)
	if uri == "" {
		return NewEditError("docs", "replace-image", docID, "invalid_argument", "empty uri", nil)
	}

	// Build the batch request
	req := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				ReplaceImage: &docs.ReplaceImageRequest{
					ImageObjectId:      imageID,
					Uri:                uri,
					ImageReplaceMethod: strings.TrimSpace(c.ReplaceMethod),
					TabId:              strings.TrimSpace(c.TabID),
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("docs", "replace-image", docID, "invalid_json", "decode execute-from-file failed", err)
	}

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("docs", "replace-image", docID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("docs", "replace-image", docID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":  true,
			"valid":         true,
			"documentId":    docID,
			"imageId":       imageID,
			"uri":           uri,
			"replaceMethod": c.ReplaceMethod,
			"requestHash":   requestHash,
		}
		if strings.TrimSpace(c.TabID) != "" {
			payload["tabId"] = c.TabID
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(req); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", docID)
		u.Out().Printf("image-id\t%s", imageID)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return DryRunOutput(ctx, u, "docs", docID, req, map[string]any{
			"imageId":           imageID,
			"uri":               uri,
			"replaceMethod":     c.ReplaceMethod,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return NewEditError("docs", "replace-image", docID, "service_init_failed", "create docs service failed", err)
	}

	_, err = svc.Documents.BatchUpdate(docID, req).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return NewEditError("docs", "replace-image", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return NewEditError("docs", "replace-image", docID, "api_error", "replace image failed", err)
	}

	payload := map[string]any{
		"documentId": docID,
		"imageId":    imageID,
		"uri":        uri,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		payload["requestHash"] = requestHash
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("image-replaced\ttrue")
	u.Out().Printf("image-id\t%s", imageID)
	return nil
}

// DocsInsertImageCmd inserts an inline image at a specific index (VID-112).
type DocsInsertImageCmd struct {
	DocID    string                 `arg:"" name:"docId" help:"Doc ID"`
	URI      string                 `name:"uri" help:"Image URI (public URL, PNG/JPEG/GIF, max 50MB)"`
	Index    int64                  `name:"index" help:"Insertion index (1-based)" default:"1"`
	WidthPt  float64                `name:"width-pt" help:"Width in points (optional)"`
	HeightPt float64                `name:"height-pt" help:"Height in points (optional)"`
	Safety   AgenticEditSafetyFlags `embed:""`
}

func (c *DocsInsertImageCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return NewEditError("docs", "insert-image", docID, "invalid_argument", "empty docId", nil)
	}
	uri := strings.TrimSpace(c.URI)
	if uri == "" {
		return NewEditError("docs", "insert-image", docID, "invalid_argument", "empty uri", nil)
	}
	if c.Index < 1 {
		return NewEditError("docs", "insert-image", docID, "invalid_argument", "index must be >= 1", nil)
	}

	insertReq := &docs.InsertInlineImageRequest{
		Uri:      uri,
		Location: &docs.Location{Index: c.Index},
	}
	if c.WidthPt > 0 || c.HeightPt > 0 {
		insertReq.ObjectSize = &docs.Size{}
		if c.WidthPt > 0 {
			insertReq.ObjectSize.Width = &docs.Dimension{Magnitude: c.WidthPt, Unit: "PT"}
		}
		if c.HeightPt > 0 {
			insertReq.ObjectSize.Height = &docs.Dimension{Magnitude: c.HeightPt, Unit: "PT"}
		}
	}

	req := &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{InsertInlineImage: insertReq},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("docs", "insert-image", docID, "invalid_json", "decode execute-from-file failed", err)
	}
	applyDocsEditSafety(req, c.Safety)

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("docs", "insert-image", docID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("docs", "insert-image", docID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly": true,
			"valid":        true,
			"documentId":   docID,
			"uri":          uri,
			"index":        c.Index,
			"requestHash":  requestHash,
		}
		if c.WidthPt > 0 {
			payload["widthPt"] = c.WidthPt
		}
		if c.HeightPt > 0 {
			payload["heightPt"] = c.HeightPt
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(req); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", docID)
		u.Out().Printf("uri\t%s", uri)
		u.Out().Printf("index\t%d", c.Index)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return DryRunOutput(ctx, u, "docs", docID, req, map[string]any{
			"uri":               uri,
			"index":             c.Index,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return NewEditError("docs", "insert-image", docID, "service_init_failed", "create docs service failed", err)
	}

	_, err = svc.Documents.BatchUpdate(docID, req).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return NewEditError("docs", "insert-image", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return NewEditError("docs", "insert-image", docID, "api_error", "insert image failed", err)
	}

	payload := map[string]any{
		"documentId":    docID,
		"uri":           uri,
		"index":         c.Index,
		"imageInserted": true,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		payload["requestHash"] = requestHash
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("image-inserted\ttrue")
	u.Out().Printf("index\t%d", c.Index)
	return nil
}

// DocsEditMergeDataCmd generates Google Docs from a template using JSON data (mail-merge).
type DocsEditMergeDataCmd struct {
	TemplateID       string                 `arg:"" name:"templateId" help:"Template document ID"`
	DataFile         string                 `name:"data-file" help:"Path to JSON array of data objects"`
	OutputFolderID   string                 `name:"output-folder-id" help:"Drive folder ID for output (default: same as template)"`
	FilenameFormat   string                 `name:"filename-format" help:"Format for output filenames using {{placeholder}} syntax (default: 'Generated - {{name}}')"`
	IncludeTimestamp bool                   `name:"include-timestamp" help:"Append timestamp to filename for uniqueness"`
	Safety           AgenticEditSafetyFlags `embed:""`
}

func (c *DocsEditMergeDataCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "docs")
	templateID := strings.TrimSpace(normalizeGoogleID(c.TemplateID))
	dataFile := strings.TrimSpace(c.DataFile)

	if templateID == "" {
		return NewEditError("docs", "merge-data", templateID, "invalid_argument", "empty templateId", nil)
	}
	if dataFile == "" {
		return NewEditError("docs", "merge-data", templateID, "invalid_argument", "empty data-file", nil)
	}

	dataBytes, err := os.ReadFile(dataFile) //nolint:gosec // user-provided path
	if err != nil {
		return NewEditError("docs", "merge-data", templateID, "input_open_failed", "read data-file failed", err)
	}

	var dataRecords []map[string]any
	if jsonErr := json.Unmarshal(dataBytes, &dataRecords); jsonErr != nil {
		return NewEditError("docs", "merge-data", templateID, "invalid_json", "parse data-file failed", jsonErr)
	}
	if len(dataRecords) == 0 {
		return NewEditError("docs", "merge-data", templateID, "invalid_argument", "data-file contains no records", nil)
	}

	sampleRecord := dataRecords[0]
	if len(sampleRecord) == 0 {
		return NewEditError("docs", "merge-data", templateID, "invalid_argument", "data records are empty", nil)
	}

	// Build preview of operations (first 3 records)
	previewRecords := dataRecords
	if len(previewRecords) > 3 {
		previewRecords = previewRecords[:3]
	}
	operations := make([]map[string]any, 0, len(previewRecords))
	for _, record := range previewRecords {
		filename := FormatMergeFilename(c.FilenameFormat, record, c.IncludeTimestamp)
		ops := make([]map[string]any, 0)
		for key, value := range record {
			ops = append(ops, map[string]any{
				"operation": "ReplaceAllText",
				"find":      fmt.Sprintf("{{%s}}", key),
				"replace":   fmt.Sprintf("%v", value),
			})
		}
		operations = append(operations, map[string]any{
			"filename":   filename,
			"operations": ops,
		})
	}

	requestHash, hashErr := RequestHash(dataRecords)
	if hashErr != nil {
		return NewEditError("docs", "merge-data", templateID, "invalid_request", "failed to hash data", hashErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"templateId":     templateID,
			"recordCount":    len(dataRecords),
			"sampleFilename": FormatMergeFilename(c.FilenameFormat, sampleRecord, c.IncludeTimestamp),
			"requestHash":    requestHash,
			"operations":     operations,
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("template\t%s", templateID)
		u.Out().Printf("records\t%d", len(dataRecords))
		u.Out().Printf("sample-filename\t%s", payload["sampleFilename"])
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		dryRunPayload := map[string]any{
			"dryRun":      true,
			"service":     "docs",
			"templateId":  templateID,
			"recordCount": len(dataRecords),
			"requestHash": requestHash,
			"operations":  operations,
		}
		if c.Safety.Pretty {
			if norm, normErr := NormalizedRequestString(dataRecords); normErr == nil {
				dryRunPayload["normalizedData"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, dryRunPayload)
		}
		u.Out().Printf("dry-run\ttrue")
		u.Out().Printf("service\tdocs")
		u.Out().Printf("template\t%s", templateID)
		u.Out().Printf("records\t%d", len(dataRecords))
		return nil
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	driveSvc, err := newDriveService(ctx, account)
	if err != nil {
		return NewEditError("docs", "merge-data", templateID, "service_init_failed", "create drive service failed", err)
	}
	docsSvc, err := newDocsService(ctx, account)
	if err != nil {
		return NewEditError("docs", "merge-data", templateID, "service_init_failed", "create docs service failed", err)
	}

	outputFolderID := strings.TrimSpace(c.OutputFolderID)
	if outputFolderID == "" {
		templateMeta, metaErr := driveSvc.Files.Get(templateID).Fields("parents").Context(ctx).Do()
		if metaErr == nil && templateMeta != nil && len(templateMeta.Parents) > 0 {
			outputFolderID = strings.TrimSpace(templateMeta.Parents[0])
		}
	}

	results := make([]map[string]any, 0, len(dataRecords))
	generatedCount := 0
	failedCount := 0

	for i, record := range dataRecords {
		filename := FormatMergeFilename(c.FilenameFormat, record, c.IncludeTimestamp)

		// 1. Copy template via Drive
		copyFile := &drive.File{Name: filename}
		if outputFolderID != "" {
			copyFile.Parents = []string{outputFolderID}
		}
		copied, copyErr := driveSvc.Files.Copy(templateID, copyFile).Context(ctx).Do()
		if copyErr != nil {
			if IsNotFound(copyErr) {
				results = append(results, map[string]any{
					"index": i, "status": "failed", "error": copyErr.Error(),
					"stage": "copy", "error_code": "template_not_found",
				})
			} else {
				results = append(results, map[string]any{
					"index": i, "status": "failed", "error": copyErr.Error(), "stage": "copy",
				})
			}
			failedCount++
			continue
		}
		newDocID := copied.Id

		// 2. ReplaceAllText for each placeholder
		req := &docs.BatchUpdateDocumentRequest{
			Requests: make([]*docs.Request, 0, len(record)),
		}
		for key, value := range record {
			textValue := fmt.Sprintf("%v", value)
			req.Requests = append(req.Requests, &docs.Request{
				ReplaceAllText: &docs.ReplaceAllTextRequest{
					ContainsText: &docs.SubstringMatchCriteria{
						Text:      fmt.Sprintf("{{%s}}", key),
						MatchCase: false,
					},
					ReplaceText: textValue,
				},
			})
		}

		_, batchErr := docsSvc.Documents.BatchUpdate(newDocID, req).Context(ctx).Do()
		if batchErr != nil {
			results = append(results, map[string]any{
				"index": i, "status": "failed", "error": batchErr.Error(),
				"stage": "batch-update", "documentId": newDocID,
			})
			failedCount++
			continue
		}

		// 3. Move to output folder if different from copy parent
		if outputFolderID != "" && copied.Parents != nil {
			alreadyInFolder := false
			for _, p := range copied.Parents {
				if strings.TrimSpace(p) == outputFolderID {
					alreadyInFolder = true
					break
				}
			}
			if !alreadyInFolder {
				fileMeta, getErr := driveSvc.Files.Get(newDocID).Fields("parents").Context(ctx).Do()
				if getErr != nil {
					results = append(results, map[string]any{
						"index": i, "status": "failed", "error": getErr.Error(),
						"stage": "get-parents", "documentId": newDocID,
					})
					failedCount++
					continue
				}
				removeParents := strings.Join(fileMeta.Parents, ",")
				moveCall := driveSvc.Files.Update(newDocID, &drive.File{}).AddParents(outputFolderID)
				if strings.TrimSpace(removeParents) != "" {
					moveCall = moveCall.RemoveParents(removeParents)
				}
				if _, moveErr := moveCall.Context(ctx).Do(); moveErr != nil {
					results = append(results, map[string]any{
						"index": i, "status": "failed", "error": moveErr.Error(),
						"stage": "move-output", "documentId": newDocID,
					})
					failedCount++
					continue
				}
			}
		}

		results = append(results, map[string]any{
			"index":      i,
			"status":     "success",
			"documentId": newDocID,
			"title":      filename,
		})
		generatedCount++
	}

	payload := map[string]any{
		"templateId":     templateID,
		"recordCount":    len(dataRecords),
		"generated":      generatedCount,
		"failed":         failedCount,
		"outputFolderId": outputFolderID,
		"results":        results,
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("template\t%s", templateID)
	u.Out().Printf("records\t%d", len(dataRecords))
	u.Out().Printf("generated\t%d", generatedCount)
	u.Out().Printf("failed\t%d", failedCount)
	return nil
}
