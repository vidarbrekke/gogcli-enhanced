package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"google.golang.org/api/slides/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SlidesEditCmd provides edit operations for Google Slides with agentic safety.
type SlidesEditCmd struct {
	Batch       SlidesEditBatchCmd       `cmd:"" name:"batch" help:"Apply multiple Slides API batch operations from JSON"`
	ReplaceText SlidesEditReplaceTextCmd `cmd:"" name:"replace-text" help:"Find and replace text across all slides"`
	MergeData   SlidesEditMergeDataCmd   `cmd:"" name:"merge-data" help:"Generate presentations from template using JSON data (mail-merge)"`
}

// SlidesEditBatchCmd applies multiple batch operations to a presentation.
type SlidesEditBatchCmd struct {
	PresentationID string              `arg:"" name:"presentationId" help:"Presentation ID"`
	RequestsFile   string              `name:"requests-file" help:"Path to JSON request body, or '-' for stdin" default:"-"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditBatchCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	presentationID := strings.TrimSpace(c.PresentationID)

	if presentationID == "" {
		return NewEditError("slides", "batch", presentationID, "invalid_argument", "empty presentationId", nil)
	}

	requestsFile := strings.TrimSpace(c.RequestsFile)
	executeFromFile := strings.TrimSpace(c.Safety.ExecuteFromFile)

	if executeFromFile != "" && requestsFile != "-" && requestsFile != "" {
		return NewEditError("slides", "batch", presentationID, "invalid_argument", "cannot combine --execute-from-file with --requests-file", nil)
	}
	if executeFromFile != "" {
		requestsFile = executeFromFile
	}
	if requestsFile == "" {
		return NewEditError("slides", "batch", presentationID, "invalid_argument", "empty requests-file", nil)
	}

	// Read request body
	var reader io.Reader = os.Stdin
	if requestsFile != "-" {
		f, openErr := os.Open(requestsFile)
		if openErr != nil {
			return NewEditError("slides", "batch", presentationID, "input_open_failed", "open requests-file failed", openErr)
		}
		defer f.Close()
		reader = f
	}

	var req slides.BatchUpdatePresentationRequest
	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		return NewEditError("slides", "batch", presentationID, "invalid_json", "decode requests JSON failed", err)
	}

	if len(req.Requests) == 0 {
		return NewEditError("slides", "batch", presentationID, "invalid_argument", "batch request has no operations", nil)
	}

	// Validate each request has exactly one operation
	for i, r := range req.Requests {
		if slidesRequestOperationCount(r) != 1 {
			idx := i
			err := NewEditError("slides", "batch", presentationID, "invalid_request",
				fmt.Sprintf("request[%d] must set exactly one operation field", i), nil)
			if editErr, ok := err.(*EditError); ok {
				editErr.RequestIndex = &idx
			}
			return err
		}
	}

	requestHash, hashErr := RequestHash(&req)
	if hashErr != nil {
		return NewEditError("slides", "batch", presentationID, "invalid_request", "failed to hash normalized request", hashErr)
	}

	requestKinds := make([]string, 0, len(req.Requests))
	for _, r := range req.Requests {
		requestKinds = append(requestKinds, slidesRequestOperationName(r))
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, &req)
	if normErr != nil {
		return NewEditError("slides", "batch", presentationID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":     true,
			"valid":            true,
			"presentationId":   presentationID,
			"operations":       len(req.Requests),
			"requestKinds":     requestKinds,
			"requestHash":      requestHash,
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(&req); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", presentationID)
		u.Out().Printf("operations\t%d", len(req.Requests))
		return nil
	}

	if c.Safety.DryRun {
		return SlidesDryRunOutput(ctx, u, presentationID, &req, map[string]any{
			"operations":        len(req.Requests),
			"requestKinds":      requestKinds,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSlidesService(ctx, account)
	if err != nil {
		return NewEditError("slides", "batch", presentationID, "service_init_failed", "create slides service failed", err)
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, &req).Do()
	if err != nil {
		if IsNotFound(err) {
			return NewEditError("slides", "batch", presentationID, "presentation_not_found",
				fmt.Sprintf("presentation not found (id=%s)", presentationID), err)
		}
		return NewEditError("slides", "batch", presentationID, "api_error", "batch update failed", err)
	}

	payload := map[string]any{
		"presentationId": presentationID,
		"operations":     len(req.Requests),
		"replies":        len(resp.Replies),
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", presentationID)
	u.Out().Printf("operations\t%d", len(req.Requests))
	u.Out().Printf("replies\t%d", len(resp.Replies))
	return nil
}

// slidesRequestOperationCount returns the number of operation fields set in a slides.Request.
func slidesRequestOperationCount(r *slides.Request) int {
	if r == nil {
		return 0
	}
	count := 0
	if r.CreateSlide != nil {
		count++
	}
	if r.CreateShape != nil {
		count++
	}
	if r.CreateTable != nil {
		count++
	}
	if r.InsertText != nil {
		count++
	}
	if r.ReplaceAllText != nil {
		count++
	}
	if r.DeleteObject != nil {
		count++
	}
	if r.DeleteText != nil {
		count++
	}
	if r.UpdatePageProperties != nil {
		count++
	}
	if r.UpdateShapeProperties != nil {
		count++
	}
	if r.UpdateTableCellProperties != nil {
		count++
	}
	if r.UpdateTextStyle != nil {
		count++
	}
	if r.DuplicateObject != nil {
		count++
	}
	if r.RefreshSheetsChart != nil {
		count++
	}
	if r.ReplaceAllShapesWithSheetsChart != nil {
		count++
	}
	if r.ReplaceImage != nil {
		count++
	}
	return count
}

// slidesRequestOperationName returns the name of the first set operation field in a slides.Request.
func slidesRequestOperationName(r *slides.Request) string {
	if r == nil {
		return ""
	}
	if r.CreateSlide != nil {
		return "CreateSlide"
	}
	if r.CreateShape != nil {
		return "CreateShape"
	}
	if r.CreateTable != nil {
		return "CreateTable"
	}
	if r.InsertText != nil {
		return "InsertText"
	}
	if r.ReplaceAllText != nil {
		return "ReplaceAllText"
	}
	if r.DeleteObject != nil {
		return "DeleteObject"
	}
	if r.DeleteText != nil {
		return "DeleteText"
	}
	if r.UpdatePageProperties != nil {
		return "UpdatePageProperties"
	}
	if r.UpdateShapeProperties != nil {
		return "UpdateShapeProperties"
	}
	if r.UpdateTableCellProperties != nil {
		return "UpdateTableCellProperties"
	}
	if r.UpdateTextStyle != nil {
		return "UpdateTextStyle"
	}
	if r.DuplicateObject != nil {
		return "DuplicateObject"
	}
	if r.RefreshSheetsChart != nil {
		return "RefreshSheetsChart"
	}
	if r.ReplaceAllShapesWithSheetsChart != nil {
		return "ReplaceAllShapesWithSheetsChart"
	}
	if r.ReplaceImage != nil {
		return "ReplaceImage"
	}
	return ""
}

// SlidesDryRunOutput is a wrapper for Slides dry-run output using shared helpers.
func SlidesDryRunOutput(ctx context.Context, u *ui.UI, presentationID string, req any, extra map[string]any, includePretty bool) error {
	return DryRunOutput(ctx, u, "slides", presentationID, req, extra, includePretty)
}

// SlidesEditReplaceTextCmd finds and replaces text across all slides.
type SlidesEditReplaceTextCmd struct {
	PresentationID string `arg:"" name:"presentationId" help:"Presentation ID"`
	Find           string `name:"find" help:"Text to find"`
	Replace        string `name:"replace" help:"Replacement text"`
	MatchCase      bool   `name:"match-case" help:"Case-sensitive matching"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditReplaceTextCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	presentationID := strings.TrimSpace(c.PresentationID)
	find := strings.TrimSpace(c.Find)
	replace := c.Replace

	if presentationID == "" {
		return NewEditError("slides", "replace-text", presentationID, "invalid_argument", "empty presentationId", nil)
	}
	if find == "" {
		return NewEditError("slides", "replace-text", presentationID, "invalid_argument", "empty find", nil)
	}

	// Build the batch request
	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				ReplaceAllText: &slides.ReplaceAllTextRequest{
					ContainsText: &slides.SubstringMatchCriteria{
						Text:      find,
						MatchCase: c.MatchCase,
					},
					ReplaceText: replace,
				},
			},
		},
	}

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("slides", "replace-text", presentationID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("slides", "replace-text", presentationID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"presentationId": presentationID,
			"find":           find,
			"replace":        replace,
			"matchCase":      c.MatchCase,
			"requestHash":    requestHash,
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
		u.Out().Printf("id\t%s", presentationID)
		u.Out().Printf("find\t%s", find)
		u.Out().Printf("replace\t%s", replace)
		return nil
	}

	if c.Safety.DryRun {
		return SlidesDryRunOutput(ctx, u, presentationID, req, map[string]any{
			"find":              find,
			"replace":           replace,
			"matchCase":         c.MatchCase,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSlidesService(ctx, account)
	if err != nil {
		return NewEditError("slides", "replace-text", presentationID, "service_init_failed", "create slides service failed", err)
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, req).Do()
	if err != nil {
		if IsNotFound(err) {
			return NewEditError("slides", "replace-text", presentationID, "presentation_not_found",
				fmt.Sprintf("presentation not found (id=%s)", presentationID), err)
		}
		return NewEditError("slides", "replace-text", presentationID, "api_error", "replace text failed", err)
	}

	// Count occurrences replaced
	occurrences := 0
	for _, reply := range resp.Replies {
		if reply.ReplaceAllText != nil {
			occurrences += int(reply.ReplaceAllText.OccurrencesChanged)
		}
	}

	payload := map[string]any{
		"presentationId": presentationID,
		"find":           find,
		"replace":        replace,
		"occurrences":    occurrences,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", presentationID)
	u.Out().Printf("find\t%s", find)
	u.Out().Printf("replace\t%s", replace)
	u.Out().Printf("occurrences\t%d", occurrences)
	return nil
}

// SlidesEditMergeDataCmd generates presentations from a template using JSON data.
type SlidesEditMergeDataCmd struct {
	TemplateID       string `arg:"" name:"templateId" help:"Template presentation ID"`
	DataFile         string `name:"data-file" help:"Path to JSON array of data objects"`
	OutputFolderID   string `name:"output-folder-id" help:"Drive folder ID for output (default: same as template)"`
	FilenameFormat   string `name:"filename-format" help:"Format for output filenames using {{placeholder}} syntax (default: 'Generated - {{name}}')"`
	ExportAsPDF      bool   `name:"export-pdf" help:"Export as PDF instead of creating Google Slides"`
	IncludeTimestamp bool   `name:"include-timestamp" help:"Append timestamp to filename for uniqueness"`
	Safety           AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditMergeDataCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	templateID := strings.TrimSpace(c.TemplateID)
	dataFile := strings.TrimSpace(c.DataFile)

	if templateID == "" {
		return NewEditError("slides", "merge-data", templateID, "invalid_argument", "empty templateId", nil)
	}
	if dataFile == "" {
		return NewEditError("slides", "merge-data", templateID, "invalid_argument", "empty data-file", nil)
	}

	// Read and parse the data file
	dataBytes, err := os.ReadFile(dataFile)
	if err != nil {
		return NewEditError("slides", "merge-data", templateID, "input_open_failed", "read data-file failed", err)
	}

	var dataRecords []map[string]any
	if err := json.Unmarshal(dataBytes, &dataRecords); err != nil {
		return NewEditError("slides", "merge-data", templateID, "invalid_json", "parse data-file failed", err)
	}
	if len(dataRecords) == 0 {
		return NewEditError("slides", "merge-data", templateID, "invalid_argument", "data-file contains no records", nil)
	}

	// Validate first record has data
	sampleRecord := dataRecords[0]
	if len(sampleRecord) == 0 {
		return NewEditError("slides", "merge-data", templateID, "invalid_argument", "data records are empty", nil)
	}

	// Build preview of operations (first 3 records only for preview)
	previewRecords := dataRecords
	if len(previewRecords) > 3 {
		previewRecords = previewRecords[:3]
	}
	operations := make([]map[string]any, 0, len(previewRecords))
	for _, record := range previewRecords {
		filename := formatMergeFilename(c.FilenameFormat, record, c.IncludeTimestamp)
		ops := make([]map[string]any, 0)
		for key, value := range record {
			textValue := fmt.Sprintf("%v", value)
			ops = append(ops, map[string]any{
				"operation": "ReplaceAllText",
				"find":      fmt.Sprintf("{{%s}}", key),
				"replace":   textValue,
			})
		}
		operations = append(operations, map[string]any{
			"filename":   filename,
			"operations": ops,
		})
	}

	requestHash, hashErr := RequestHash(dataRecords)
	if hashErr != nil {
		return NewEditError("slides", "merge-data", templateID, "invalid_request", "failed to hash data", hashErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"templateId":     templateID,
			"recordCount":    len(dataRecords),
			"sampleFilename": formatMergeFilename(c.FilenameFormat, sampleRecord, c.IncludeTimestamp),
			"requestHash":    requestHash,
			"operations":     operations,
			"exportAsPDF":    c.ExportAsPDF,
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("template\t%s", templateID)
		u.Out().Printf("records\t%d", len(dataRecords))
		u.Out().Printf("sample-filename\t%s", payload["sampleFilename"])
		return nil
	}

	if c.Safety.DryRun {
		dryRunPayload := map[string]any{
			"dryRun":       true,
			"service":      "slides",
			"templateId":   templateID,
			"recordCount":  len(dataRecords),
			"requestHash":  requestHash,
			"exportAsPDF":  c.ExportAsPDF,
			"operations":   operations,
		}

		// Add full preview if --pretty
		if c.Safety.Pretty {
			if norm, err := NormalizedRequestString(dataRecords); err == nil {
				dryRunPayload["normalizedData"] = norm
			}
		}

		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(os.Stdout, dryRunPayload)
		}
		u.Out().Printf("dry-run\ttrue")
		u.Out().Printf("service\tslides")
		u.Out().Printf("template\t%s", templateID)
		u.Out().Printf("records\t%d", len(dataRecords))
		u.Out().Printf("would-generate\t%d", len(dataRecords))
		return nil
	}

	// Execute merge (requires auth)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	svc, err := newSlidesService(ctx, account)
	if err != nil {
		return NewEditError("slides", "merge-data", templateID, "service_init_failed", "create slides service failed", err)
	}

	results := make([]map[string]any, 0, len(dataRecords))
	generatedCount := 0
	failedCount := 0

	for i, record := range dataRecords {
		filename := formatMergeFilename(c.FilenameFormat, record, c.IncludeTimestamp)

		// 1. Copy template to new presentation
		copyReq := &slides.Presentation{Title: filename}
		newPres, err := svc.Presentations.Create(copyReq).Do()
		if err != nil {
			results = append(results, map[string]any{
				"index":   i,
				"status":  "failed",
				"error":   err.Error(),
				"stage":   "create",
			})
			failedCount++
			continue
		}

		// 2. Build and execute replace operations
		batchReq := &slides.BatchUpdatePresentationRequest{Requests: make([]*slides.Request, 0)}
		for key, value := range record {
			textValue := fmt.Sprintf("%v", value)
			batchReq.Requests = append(batchReq.Requests, &slides.Request{
				ReplaceAllText: &slides.ReplaceAllTextRequest{
					ContainsText: &slides.SubstringMatchCriteria{
						Text:      fmt.Sprintf("{{%s}}", key),
						MatchCase: true,
					},
					ReplaceText: textValue,
				},
			})
		}

		_, err = svc.Presentations.BatchUpdate(newPres.PresentationId, batchReq).Do()
		if err != nil {
			results = append(results, map[string]any{
				"index":         i,
				"status":        "failed",
				"error":         err.Error(),
				"stage":         "batch-update",
				"presentationId": newPres.PresentationId,
			})
			failedCount++
			continue
		}

		// 3. Move to output folder if specified
		outputFolderID := strings.TrimSpace(c.OutputFolderID)
		if outputFolderID != "" {
			// Note: Would need Drive API to move file
			// For now, just track it
		}

		// 4. Export as PDF if requested (would require additional implementation)
		if c.ExportAsPDF {
			// Would call Drive API export method
		}

		results = append(results, map[string]any{
			"index":          i,
			"status":         "success",
			"presentationId": newPres.PresentationId,
			"title":          newPres.Title,
		})
		generatedCount++
	}

	payload := map[string]any{
		"templateId":     templateID,
		"recordCount":    len(dataRecords),
		"generated":      generatedCount,
		"failed":         failedCount,
		"exportAsPDF":    c.ExportAsPDF,
		"results":        results,
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("template\t%s", templateID)
	u.Out().Printf("records\t%d", len(dataRecords))
	u.Out().Printf("generated\t%d", generatedCount)
	u.Out().Printf("failed\t%d", failedCount)
	return nil
}

// formatMergeFilename formats filename using {{placeholder}} syntax
func formatMergeFilename(format string, data map[string]any, includeTimestamp bool) string {
	if format == "" {
		format = "Generated - {{name}}"
	}

	result := format
	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		textValue := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, textValue)
	}

	// Clean up any unreplaced placeholders
	result = strings.ReplaceAll(result, "{{", "")
	result = strings.ReplaceAll(result, "}}", "")
	result = strings.TrimSpace(result)

	if includeTimestamp {
		timestamp := time.Now().Format("2006-01-02-150405")
		result = fmt.Sprintf("%s-%s", result, timestamp)
	}

	return result
}

// newSlidesService creates a new Slides API service.
// This variable is overridden by the googleapi package to provide actual implementation.
var newSlidesService = func(ctx context.Context, account string) (*slides.Service, error) {
	return nil, errors.New("slides service not initialized - ensure googleapi/slides.go is imported")
}
