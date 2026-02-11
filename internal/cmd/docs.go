package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	gapi "google.golang.org/api/googleapi"

	"github.com/steipete/gogcli/internal/googleapi"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

var newDocsService = googleapi.NewDocs

type DocsCmd struct {
	Export DocsExportCmd `cmd:"" name:"export" help:"Export a Google Doc (pdf|docx|txt)"`
	Info   DocsInfoCmd   `cmd:"" name:"info" help:"Get Google Doc metadata"`
	Create DocsCreateCmd `cmd:"" name:"create" help:"Create a Google Doc"`
	Copy   DocsCopyCmd   `cmd:"" name:"copy" help:"Copy a Google Doc"`
	Cat    DocsCatCmd    `cmd:"" name:"cat" help:"Print a Google Doc as plain text"`
	Edit   DocsEditCmd   `cmd:"" name:"edit" help:"Edit Google Doc content"`
}

type DocsEditCmd struct {
	Append  DocsAppendCmd  `cmd:"" name:"append" help:"Append text to the end of a Google Doc"`
	Batch   DocsBatchCmd   `cmd:"" name:"batch" help:"Apply multiple Docs API edit operations from JSON"`
	Delete  DocsDeleteCmd  `cmd:"" name:"delete" help:"Delete a text range in a Google Doc"`
	Insert  DocsInsertCmd  `cmd:"" name:"insert" help:"Insert text at a specific index in a Google Doc"`
	Replace DocsReplaceCmd `cmd:"" name:"replace" help:"Replace text throughout a Google Doc"`
}

type DocsEditSafetyFlags struct {
	DryRun          bool   `name:"dry-run" help:"Build request and print it without executing API call"`
	RequireRevision string `name:"require-revision" help:"Require this document revision ID for update (optimistic concurrency guard)"`
}

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

type DocsBatchCmd struct {
	DocID        string `arg:"" name:"docId" help:"Doc ID"`
	RequestsFile string `name:"requests-file" help:"Path to JSON request body, or '-' for stdin" default:"-"`
	ExecuteFromFile string `name:"execute-from-file" help:"Execute request JSON from this file (bypasses --requests-file input)"`
	ValidateOnly bool   `name:"validate-only" help:"Validate request payload locally without executing API call"`
	Pretty       bool   `name:"pretty" help:"Include normalized pretty-printed request JSON in output"`
	OutputRequestFile string `name:"output-request-file" help:"Write normalized request JSON to this file (use '-' for stdout)"`
	Safety       DocsEditSafetyFlags `embed:""`
}

func (c *DocsBatchCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return newDocsEditError("batch", docID, "invalid_argument", "empty docId", usage("empty docId"))
	}
	requestsFile := strings.TrimSpace(c.RequestsFile)
	executeFromFile := strings.TrimSpace(c.ExecuteFromFile)
	if executeFromFile != "" && strings.TrimSpace(c.RequestsFile) != "-" && strings.TrimSpace(c.RequestsFile) != "" {
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
			if de, ok := err.(*docsEditError); ok {
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
	if strings.TrimSpace(c.OutputRequestFile) == "-" && outfmt.IsJSON(ctx) {
		norm, normErr := docsNormalizedRequestString(&req)
		if normErr != nil {
			return newDocsEditError("batch", docID, "invalid_request", "failed to normalize request", normErr)
		}
		normalizedForJSON = norm
	} else if err := docsMaybeWriteNormalizedRequest(c.OutputRequestFile, &req); err != nil {
		return newDocsEditError("batch", docID, "output_write_failed", "write normalized request failed", err)
	}
	requestKinds := make([]string, 0, len(req.Requests))
	for _, r := range req.Requests {
		requestKinds = append(requestKinds, docsRequestOperationName(r))
	}
	if c.ValidateOnly {
		payload := map[string]any{
			"validateOnly": true,
			"valid":        true,
			"documentId":   docID,
			"operations":   len(req.Requests),
			"requestKinds": requestKinds,
			"requestHash":  requestHash,
		}
		if c.Pretty {
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
		if c.Pretty {
			pretty, prettyErr := json.MarshalIndent(req, "", "  ")
			if prettyErr == nil {
				u.Out().Printf("pretty-request\t%s", string(pretty))
			}
		}
		return nil
	}
	if c.Safety.DryRun {
		return docsDryRunOutput(ctx, u, docID, &req, map[string]any{
			"operations":   len(req.Requests),
			"requestKinds": requestKinds,
			"requestHash":  requestHash,
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
	DocID      string `arg:"" name:"docId" help:"Doc ID"`
	StartIndex int64  `arg:"" name:"start" help:"Start index (inclusive, 1-based)"`
	EndIndex   int64  `arg:"" name:"end" help:"End index (exclusive)"`
	Safety     DocsEditSafetyFlags `embed:""`
}

func (c *DocsDeleteCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

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

	svc, err := newDocsService(ctx, account)
	if err != nil {
		return newDocsEditError("delete", docID, "service_init_failed", "create docs service failed", err)
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
	if c.Safety.DryRun {
		return docsDryRunOutput(ctx, u, docID, req, map[string]any{
			"deletedChars": c.EndIndex - c.StartIndex,
		})
	}
	if _, err := svc.Documents.BatchUpdate(docID, req).Context(ctx).Do(); err != nil {
		if isDocsNotFound(err) {
			return newDocsEditError("delete", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return newDocsEditError("delete", docID, "api_error", "delete failed", err)
	}

	deletedChars := c.EndIndex - c.StartIndex
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{
			"documentId":   docID,
			"deletedChars": deletedChars,
		})
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("deleted\t%d", deletedChars)
	return nil
}

type DocsInsertCmd struct {
	DocID string `arg:"" name:"docId" help:"Doc ID"`
	Text  string `arg:"" name:"text" help:"Text to insert"`
	Index int64  `name:"index" help:"Insertion index (1-based)" default:"1"`
	Safety DocsEditSafetyFlags `embed:""`
}

func (c *DocsInsertCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

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

	svc, err := newDocsService(ctx, account)
	if err != nil {
		return newDocsEditError("insert", docID, "service_init_failed", "create docs service failed", err)
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
	if c.Safety.DryRun {
		return docsDryRunOutput(ctx, u, docID, req, map[string]any{
			"insertedChars": len(text),
			"index":         c.Index,
		})
	}
	if _, err := svc.Documents.BatchUpdate(docID, req).Context(ctx).Do(); err != nil {
		if isDocsNotFound(err) {
			return newDocsEditError("insert", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return newDocsEditError("insert", docID, "api_error", "insert failed", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{
			"documentId":    docID,
			"insertedChars": len(text),
			"index":         c.Index,
		})
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("inserted\t%d", len(text))
	u.Out().Printf("index\t%d", c.Index)
	return nil
}

type DocsAppendCmd struct {
	DocID string `arg:"" name:"docId" help:"Doc ID"`
	Text  string `arg:"" name:"text" help:"Text to append"`
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
	if c.Safety.DryRun {
		return docsDryRunOutput(ctx, u, docID, req, map[string]any{
			"insertedChars": len(text),
			"index":         index,
		})
	}
	if _, err := svc.Documents.BatchUpdate(docID, req).Context(ctx).Do(); err != nil {
		if isDocsNotFound(err) {
			return newDocsEditError("append", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return newDocsEditError("append", docID, "api_error", "append failed", err)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{
			"documentId":    docID,
			"insertedChars": len(text),
			"index":         index,
		})
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("appended\t%d", len(text))
	u.Out().Printf("index\t%d", index)
	return nil
}

type DocsReplaceCmd struct {
	DocID     string `arg:"" name:"docId" help:"Doc ID"`
	Find      string `arg:"" name:"find" help:"Text to find"`
	Replace   string `arg:"" name:"replace" help:"Replacement text"`
	MatchCase bool   `name:"match-case" help:"Case-sensitive matching"`
	Safety    DocsEditSafetyFlags `embed:""`
}

func (c *DocsReplaceCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	docID := strings.TrimSpace(c.DocID)
	if docID == "" {
		return newDocsEditError("replace", docID, "invalid_argument", "empty docId", usage("empty docId"))
	}
	find := strings.TrimSpace(c.Find)
	if find == "" {
		return newDocsEditError("replace", docID, "invalid_argument", "empty find", usage("empty find"))
	}

	svc, err := newDocsService(ctx, account)
	if err != nil {
		return newDocsEditError("replace", docID, "service_init_failed", "create docs service failed", err)
	}

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
	applyDocsEditSafety(req, c.Safety)
	if c.Safety.DryRun {
		return docsDryRunOutput(ctx, u, docID, req, map[string]any{
			"operation": "replace",
		})
	}
	resp, err := svc.Documents.BatchUpdate(docID, req).Context(ctx).Do()
	if err != nil {
		if isDocsNotFound(err) {
			return newDocsEditError("replace", docID, "doc_not_found", fmt.Sprintf("doc not found or not a Google Doc (id=%s)", docID), err)
		}
		return newDocsEditError("replace", docID, "api_error", "replace failed", err)
	}

	var occurrences int64
	if resp != nil && len(resp.Replies) > 0 && resp.Replies[0] != nil && resp.Replies[0].ReplaceAllText != nil {
		occurrences = resp.Replies[0].ReplaceAllText.OccurrencesChanged
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{
			"documentId":         docID,
			"occurrencesChanged": occurrences,
		})
	}
	u.Out().Printf("id\t%s", docID)
	u.Out().Printf("replaced\t%d", occurrences)
	return nil
}

type DocsExportCmd struct {
	DocID  string         `arg:"" name:"docId" help:"Doc ID"`
	Output OutputPathFlag `embed:""`
	Format string         `name:"format" help:"Export format: pdf|docx|txt" default:"pdf"`
}

func (c *DocsExportCmd) Run(ctx context.Context, flags *RootFlags) error {
	return exportViaDrive(ctx, flags, exportViaDriveOptions{
		ArgName:       "docId",
		ExpectedMime:  "application/vnd.google-apps.document",
		KindLabel:     "Google Doc",
		DefaultFormat: "pdf",
	}, c.DocID, c.Output.Path, c.Format)
}

type DocsInfoCmd struct {
	DocID string `arg:"" name:"docId" help:"Doc ID"`
}

func (c *DocsInfoCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	id := strings.TrimSpace(c.DocID)
	if id == "" {
		return usage("empty docId")
	}

	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}

	doc, err := svc.Documents.Get(id).
		Fields("documentId,title,revisionId").
		Context(ctx).
		Do()
	if err != nil {
		if isDocsNotFound(err) {
			return fmt.Errorf("doc not found or not a Google Doc (id=%s)", id)
		}
		return err
	}
	if doc == nil {
		return errors.New("doc not found")
	}

	file := map[string]any{
		"id":       doc.DocumentId,
		"name":     doc.Title,
		"mimeType": driveMimeGoogleDoc,
	}
	if link := docsWebViewLink(doc.DocumentId); link != "" {
		file["webViewLink"] = link
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{
			strFile:    file,
			"document": doc,
		})
	}

	u.Out().Printf("id\t%s", doc.DocumentId)
	u.Out().Printf("name\t%s", doc.Title)
	u.Out().Printf("mime\t%s", driveMimeGoogleDoc)
	if link := docsWebViewLink(doc.DocumentId); link != "" {
		u.Out().Printf("link\t%s", link)
	}
	if doc.RevisionId != "" {
		u.Out().Printf("revision\t%s", doc.RevisionId)
	}
	return nil
}

type DocsCreateCmd struct {
	Title  string `arg:"" name:"title" help:"Doc title"`
	Parent string `name:"parent" help:"Destination folder ID"`
}

func (c *DocsCreateCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	title := strings.TrimSpace(c.Title)
	if title == "" {
		return usage("empty title")
	}

	svc, err := newDriveService(ctx, account)
	if err != nil {
		return err
	}

	f := &drive.File{
		Name:     title,
		MimeType: "application/vnd.google-apps.document",
	}
	parent := strings.TrimSpace(c.Parent)
	if parent != "" {
		f.Parents = []string{parent}
	}

	created, err := svc.Files.Create(f).
		SupportsAllDrives(true).
		Fields("id, name, mimeType, webViewLink").
		Context(ctx).
		Do()
	if err != nil {
		return err
	}
	if created == nil {
		return errors.New("create failed")
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{strFile: created})
	}

	u.Out().Printf("id\t%s", created.Id)
	u.Out().Printf("name\t%s", created.Name)
	u.Out().Printf("mime\t%s", created.MimeType)
	if created.WebViewLink != "" {
		u.Out().Printf("link\t%s", created.WebViewLink)
	}
	return nil
}

type DocsCopyCmd struct {
	DocID  string `arg:"" name:"docId" help:"Doc ID"`
	Title  string `arg:"" name:"title" help:"New title"`
	Parent string `name:"parent" help:"Destination folder ID"`
}

func (c *DocsCopyCmd) Run(ctx context.Context, flags *RootFlags) error {
	return copyViaDrive(ctx, flags, copyViaDriveOptions{
		ArgName:      "docId",
		ExpectedMime: "application/vnd.google-apps.document",
		KindLabel:    "Google Doc",
	}, c.DocID, c.Title, c.Parent)
}

type DocsCatCmd struct {
	DocID    string `arg:"" name:"docId" help:"Doc ID"`
	MaxBytes int64  `name:"max-bytes" help:"Max bytes to read (0 = unlimited)" default:"2000000"`
}

func (c *DocsCatCmd) Run(ctx context.Context, flags *RootFlags) error {
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	id := strings.TrimSpace(c.DocID)
	if id == "" {
		return usage("empty docId")
	}

	svc, err := newDocsService(ctx, account)
	if err != nil {
		return err
	}

	doc, err := svc.Documents.Get(id).
		Context(ctx).
		Do()
	if err != nil {
		if isDocsNotFound(err) {
			return fmt.Errorf("doc not found or not a Google Doc (id=%s)", id)
		}
		return err
	}
	if doc == nil {
		return errors.New("doc not found")
	}

	text := docsPlainText(doc, c.MaxBytes)

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]any{"text": text})
	}
	_, err = io.WriteString(os.Stdout, text)
	return err
}

func docsWebViewLink(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	return "https://docs.google.com/document/d/" + id + "/edit"
}

func docsPlainText(doc *docs.Document, maxBytes int64) string {
	if doc == nil || doc.Body == nil {
		return ""
	}

	var buf bytes.Buffer
	for _, el := range doc.Body.Content {
		if !appendDocsElementText(&buf, maxBytes, el) {
			break
		}
	}

	return buf.String()
}

func appendDocsElementText(buf *bytes.Buffer, maxBytes int64, el *docs.StructuralElement) bool {
	if el == nil {
		return true
	}

	switch {
	case el.Paragraph != nil:
		for _, p := range el.Paragraph.Elements {
			if p.TextRun == nil {
				continue
			}
			if !appendLimited(buf, maxBytes, p.TextRun.Content) {
				return false
			}
		}
	case el.Table != nil:
		for rowIdx, row := range el.Table.TableRows {
			if rowIdx > 0 {
				if !appendLimited(buf, maxBytes, "\n") {
					return false
				}
			}
			for cellIdx, cell := range row.TableCells {
				if cellIdx > 0 {
					if !appendLimited(buf, maxBytes, "\t") {
						return false
					}
				}
				for _, content := range cell.Content {
					if !appendDocsElementText(buf, maxBytes, content) {
						return false
					}
				}
			}
		}
	case el.TableOfContents != nil:
		for _, content := range el.TableOfContents.Content {
			if !appendDocsElementText(buf, maxBytes, content) {
				return false
			}
		}
	}

	return true
}

func appendLimited(buf *bytes.Buffer, maxBytes int64, s string) bool {
	if maxBytes <= 0 {
		_, _ = buf.WriteString(s)
		return true
	}

	remaining := int(maxBytes) - buf.Len()
	if remaining <= 0 {
		return false
	}
	if len(s) > remaining {
		_, _ = buf.WriteString(s[:remaining])
		return false
	}
	_, _ = buf.WriteString(s)
	return true
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
	payload := map[string]any{
		"dryRun":     true,
		"documentId": docID,
		"request":    req,
	}
	for k, v := range extra {
		payload[k] = v
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
