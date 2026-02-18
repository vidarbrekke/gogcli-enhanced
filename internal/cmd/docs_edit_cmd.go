package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/api/docs/v1"

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
		if docsRequestOperationCount(r) != 1 {
			idx := i
			err := newDocsEditError("batch", docID, "invalid_request", fmt.Sprintf("request[%d] must set exactly one operation field", i), usage(fmt.Sprintf("request[%d] must set exactly one operation field", i)))
			if de, ok := err.(*EditError); ok {
				de.RequestIndex = &idx
			}
			return err
		}
	}
	applyDocsEditSafety(&req, c.Safety)
	requestHash, hashErr := docsRequestHash(&req)
	if hashErr != nil {
		return newDocsEditError("batch", docID, "invalid_request", "failed to hash normalized request", hashErr)
	}
	normalizedForJSON := ""
	if strings.TrimSpace(c.Safety.OutputRequestFile) == "-" && outfmt.IsJSON(ctx) {
		norm, normErr := docsNormalizedRequestString(&req)
		if normErr != nil {
			return newDocsEditError("batch", docID, "invalid_request", "failed to normalize request", normErr)
		}
		normalizedForJSON = norm
	} else if err := docsMaybeWriteNormalizedRequest(c.Safety.OutputRequestFile, &req); err != nil {
		return newDocsEditError("batch", docID, "output_write_failed", "write normalized request failed", err)
	}
	requestKinds := make([]string, 0, len(req.Requests))
	for _, r := range req.Requests {
		requestKinds = append(requestKinds, docsRequestOperationName(r))
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
		if c.Safety.Pretty {
			pretty, prettyErr := json.MarshalIndent(req, "", "  ")
			if prettyErr == nil {
				payload["prettyRequest"] = string(pretty)
			}
		}
		if normalizedForJSON != "" {
			payload["normalizedRequest"] = normalizedForJSON
		}
		if req.WriteControl != nil && strings.TrimSpace(req.WriteControl.RequiredRevisionId) != "" {
			payload["requiredRevisionId"] = req.WriteControl.RequiredRevisionId
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(os.Stdout, payload)
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
	if c.Safety.DryRun {
		return docsDryRunOutput(ctx, u, docID, &req, map[string]any{
			"operations":        len(req.Requests),
			"requestKinds":      requestKinds,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		})
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
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("operations\t%d", operations)
	u.Out().Printf("replies\t%d", len(resp.Replies))
	return nil
}

type DocsDeleteCmd struct {
	DocID      string              `arg:"" name:"docId" help:"Doc ID"`
	StartIndex int64               `arg:"" name:"start" help:"Start index (inclusive, 1-based)"`
	EndIndex   int64               `arg:"" name:"end" help:"End index (exclusive)"`
	Safety     DocsEditSafetyFlags `embed:""`
}

func (c *DocsDeleteCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return newDocsEditError("delete", docID, "invalid_argument", "empty docId", usage("empty docId"))
	}
	if c.StartIndex < 1 {
		return newDocsEditError("delete", docID, "invalid_argument", "start must be >= 1", usage("start must be >= 1"))
	}
	if c.EndIndex <= c.StartIndex {
		return newDocsEditError("delete", docID, "invalid_argument", "end must be > start", usage("end must be > start"))
	}
	if !c.Safety.DryRun && !outfmt.IsJSON(ctx) && (flags == nil || !flags.Force) {
		return newDocsEditError("delete", docID, "confirmation_required", "delete is destructive; rerun with --force or use --dry-run", usage("delete is destructive; rerun with --force or use --dry-run"))
	}

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
	applyDocsEditSafety(req, c.Safety)
	normalizedForJSON, normErr := docsNormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newDocsEditError("delete", docID, "output_write_failed", "write normalized request failed", normErr)
	}
	if c.Safety.DryRun {
		return docsDryRunOutputWithOpts(ctx, u, docID, req, map[string]any{
			"deletedChars":      c.EndIndex - c.StartIndex,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return newDocsEditError("delete", docID, "service_init_failed", "create docs service failed", err)
	}
	if _, err := svc.Documents.BatchUpdate(docID, req).Context(ctx).Do(); err != nil {
		if isDocsNotFound(err) {
			return newDocsEditError("delete", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return newDocsEditError("delete", docID, "api_error", "delete failed", err)
	}

	deletedChars := c.EndIndex - c.StartIndex
	payload := map[string]any{
		"documentId":   docID,
		"deletedChars": deletedChars,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		if hash, hashErr := docsRequestHash(req); hashErr == nil {
			payload["requestHash"] = hash
		}
		if norm, normErr := docsNormalizedRequestString(req); normErr == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("deleted\t%d", deletedChars)
	return nil
}

type DocsInsertCmd struct {
	DocID  string              `arg:"" name:"docId" help:"Doc ID"`
	Text   string              `arg:"" name:"text" help:"Text to insert"`
	Index  int64               `name:"index" help:"Insertion index (1-based)" default:"1"`
	Safety DocsEditSafetyFlags `embed:""`
}

func (c *DocsInsertCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return newDocsEditError("insert", docID, "invalid_argument", "empty docId", usage("empty docId"))
	}
	text := strings.TrimSpace(c.Text)
	if text == "" {
		return newDocsEditError("insert", docID, "invalid_argument", "empty text", usage("empty text"))
	}
	if c.Index < 1 {
		return newDocsEditError("insert", docID, "invalid_argument", "index must be >= 1", usage("index must be >= 1"))
	}

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
	applyDocsEditSafety(req, c.Safety)
	normalizedForJSON, normErr := docsNormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newDocsEditError("insert", docID, "output_write_failed", "write normalized request failed", normErr)
	}
	if c.Safety.DryRun {
		return docsDryRunOutputWithOpts(ctx, u, docID, req, map[string]any{
			"insertedChars":     len(text),
			"index":             c.Index,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newDocsService(ctx, account)
	if err != nil {
		return newDocsEditError("insert", docID, "service_init_failed", "create docs service failed", err)
	}
	if _, err := svc.Documents.BatchUpdate(docID, req).Context(ctx).Do(); err != nil {
		if isDocsNotFound(err) {
			return newDocsEditError("insert", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return newDocsEditError("insert", docID, "api_error", "insert failed", err)
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
		if hash, hashErr := docsRequestHash(req); hashErr == nil {
			payload["requestHash"] = hash
		}
		if norm, normErr := docsNormalizedRequestString(req); normErr == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, payload)
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
	applyDocsEditSafety(req, c.Safety)
	normalizedForJSON, normErr := docsNormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newDocsEditError("append", docID, "output_write_failed", "write normalized request failed", normErr)
	}
	if c.Safety.DryRun {
		return docsDryRunOutputWithOpts(ctx, u, docID, req, map[string]any{
			"insertedChars":     len(text),
			"index":             index,
			"normalizedRequest": normalizedForJSON,
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
		if hash, hashErr := docsRequestHash(req); hashErr == nil {
			payload["requestHash"] = hash
		}
		if norm, normErr := docsNormalizedRequestString(req); normErr == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("appended\t%d", len(text))
	u.Out().Printf("index\t%d", index)
	return nil
}

type DocsReplaceCmd struct {
	DocID  string                 `arg:"" name:"docId" help:"Doc ID"`
	Find   string                 `name:"find" help:"Text to find"`
	Replace string                `name:"replace" help:"Replacement text"`
	MatchCase bool                `name:"match-case" help:"Case-sensitive matching"`
	Safety AgenticEditSafetyFlags `embed:""`
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
			return outfmt.WriteJSON(os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", docID)
		u.Out().Printf("find\t%s", find)
		u.Out().Printf("replace\t%s", c.Replace)
		return nil
	}

	if c.Safety.DryRun {
		return DryRunOutput(ctx, u, "docs", docID, req, map[string]any{
			"find":               find,
			"replace":            c.Replace,
			"matchCase":          c.MatchCase,
			"requestHash":        requestHash,
			"normalizedRequest":  normalizedForJSON,
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
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("replaced\t%d", occurrences)
	return nil
}
