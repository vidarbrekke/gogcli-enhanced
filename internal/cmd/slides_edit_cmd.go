package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/slides/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SlidesEditCmd provides edit operations for Google Slides with agentic safety.
type SlidesEditCmd struct {
	Batch        SlidesEditBatchCmd        `cmd:"" name:"batch" help:"Apply multiple Slides API batch operations from JSON"`
	ReplaceText  SlidesEditReplaceTextCmd  `cmd:"" name:"replace-text" help:"Find and replace text across all slides"`
	ReplaceImage SlidesEditReplaceImageCmd `cmd:"" name:"replace-image" help:"Replace an image in a presentation preserving position/size"`
	CreateSlide  SlidesEditCreateSlideCmd  `cmd:"" name:"create-slide" help:"Add a new slide to a presentation"`
	DuplicateSlide SlidesEditDuplicateSlideCmd `cmd:"" name:"duplicate-slide" help:"Duplicate an existing slide"`
	RefreshCharts SlidesEditRefreshChartsCmd `cmd:"" name:"refresh-charts" help:"Refresh embedded Google Sheets charts"`
	UpdateNotes  SlidesEditUpdateNotesCmd  `cmd:"" name:"update-notes" help:"Update speaker notes on a slide"`
	DeleteSlide  SlidesEditDeleteSlideCmd  `cmd:"" name:"delete-slide" help:"Delete a slide by object ID"`
	InsertTable  SlidesEditInsertTableCmd  `cmd:"" name:"insert-table" help:"Insert a data table into a slide"`
	MergeData    SlidesEditMergeDataCmd    `cmd:"" name:"merge-data" help:"Generate presentations from template using JSON data (mail-merge)"`
}

// SlidesEditBatchCmd applies multiple batch operations to a presentation.
type SlidesEditBatchCmd struct {
	PresentationID string              `arg:"" name:"presentationId" help:"Presentation ID"`
	RequestsFile   string              `name:"requests-file" help:"Path to JSON request body, or '-' for stdin" default:"-"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditBatchCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
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
		if RequestOperationCount(r) != 1 {
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
		requestKinds = append(requestKinds, RequestOperationName(r))
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
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", presentationID)
		u.Out().Printf("operations\t%d", len(req.Requests))
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
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
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", presentationID)
	u.Out().Printf("operations\t%d", len(req.Requests))
	u.Out().Printf("replies\t%d", len(resp.Replies))
	return nil
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
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
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
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("slides", "replace-text", presentationID, "invalid_json", "decode execute-from-file failed", err)
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
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", presentationID)
		u.Out().Printf("find\t%s", find)
		u.Out().Printf("replace\t%s", replace)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
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
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
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
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
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
			return outfmt.WriteJSON(ctx, os.Stdout, dryRunPayload)
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
	useDrive := strings.TrimSpace(c.OutputFolderID) != "" || c.ExportAsPDF
	var driveSvc *drive.Service
	if useDrive {
		driveSvc, err = newDriveService(ctx, account)
		if err != nil {
			return NewEditError("slides", "merge-data", templateID, "service_init_failed", "create drive service failed", err)
		}
	}
	outputFolderID := strings.TrimSpace(c.OutputFolderID)
	if outputFolderID == "" && c.ExportAsPDF && driveSvc != nil {
		templateMeta, metaErr := driveSvc.Files.Get(templateID).Fields("parents").Context(ctx).Do()
		if metaErr == nil && len(templateMeta.Parents) > 0 {
			outputFolderID = strings.TrimSpace(templateMeta.Parents[0])
		}
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
		if outputFolderID != "" && driveSvc != nil {
			fileMeta, getErr := driveSvc.Files.Get(newPres.PresentationId).Fields("parents").Context(ctx).Do()
			if getErr != nil {
				results = append(results, map[string]any{
					"index":          i,
					"status":         "failed",
					"error":          getErr.Error(),
					"stage":          "get-parents",
					"presentationId": newPres.PresentationId,
				})
				failedCount++
				continue
			}
			removeParents := strings.Join(fileMeta.Parents, ",")
			moveCall := driveSvc.Files.Update(newPres.PresentationId, &drive.File{}).AddParents(outputFolderID)
			if strings.TrimSpace(removeParents) != "" {
				moveCall = moveCall.RemoveParents(removeParents)
			}
			if _, moveErr := moveCall.Context(ctx).Do(); moveErr != nil {
				results = append(results, map[string]any{
					"index":          i,
					"status":         "failed",
					"error":          moveErr.Error(),
					"stage":          "move-output",
					"presentationId": newPres.PresentationId,
				})
				failedCount++
				continue
			}
		}

		var exportedPDFID string
		// 4. Export as PDF if requested
		if c.ExportAsPDF {
			if driveSvc == nil {
				results = append(results, map[string]any{
					"index":          i,
					"status":         "failed",
					"error":          "drive service unavailable",
					"stage":          "export-pdf",
					"presentationId": newPres.PresentationId,
				})
				failedCount++
				continue
			}
			exportResp, exportErr := driveSvc.Files.Export(newPres.PresentationId, "application/pdf").Context(ctx).Download()
			if exportErr != nil {
				results = append(results, map[string]any{
					"index":          i,
					"status":         "failed",
					"error":          exportErr.Error(),
					"stage":          "export-pdf",
					"presentationId": newPres.PresentationId,
				})
				failedCount++
				continue
			}
			pdfBytes, readErr := io.ReadAll(exportResp.Body)
			_ = exportResp.Body.Close()
			if readErr != nil {
				results = append(results, map[string]any{
					"index":          i,
					"status":         "failed",
					"error":          readErr.Error(),
					"stage":          "read-exported-pdf",
					"presentationId": newPres.PresentationId,
				})
				failedCount++
				continue
			}
			pdfName := filename + ".pdf"
			pdfFile := &drive.File{Name: pdfName, MimeType: "application/pdf"}
			if outputFolderID != "" {
				pdfFile.Parents = []string{outputFolderID}
			}
			createdPDF, createErr := driveSvc.Files.Create(pdfFile).Media(bytes.NewReader(pdfBytes)).Context(ctx).Do()
			if createErr != nil {
				results = append(results, map[string]any{
					"index":          i,
					"status":         "failed",
					"error":          createErr.Error(),
					"stage":          "create-pdf-file",
					"presentationId": newPres.PresentationId,
				})
				failedCount++
				continue
			}
			exportedPDFID = createdPDF.Id
			_ = driveSvc.Files.Delete(newPres.PresentationId).Context(ctx).Do()
		}

		result := map[string]any{
			"index":          i,
			"status":         "success",
			"presentationId": newPres.PresentationId,
			"title":          newPres.Title,
		}
		if outputFolderID != "" {
			result["outputFolderId"] = outputFolderID
		}
		if c.ExportAsPDF {
			result["pdfFileId"] = exportedPDFID
		}
		results = append(results, result)
		generatedCount++
	}

	payload := map[string]any{
		"templateId":     templateID,
		"recordCount":    len(dataRecords),
		"generated":      generatedCount,
		"failed":         failedCount,
		"exportAsPDF":    c.ExportAsPDF,
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

// SlidesEditReplaceImageCmd replaces an image in a presentation while preserving position and size.
type SlidesEditReplaceImageCmd struct {
	PresentationID string `arg:"" name:"presentationId" help:"Presentation ID"`
	ObjectID       string `name:"object-id" help:"ID of the image object to replace (e.g., 'image1')"`
	SourceURL      string `name:"source-url" help:"URL of replacement image (publicly accessible)"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditReplaceImageCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
	presentationID := strings.TrimSpace(c.PresentationID)
	objectID := strings.TrimSpace(c.ObjectID)
	sourceURL := strings.TrimSpace(c.SourceURL)

	if presentationID == "" {
		return NewEditError("slides", "replace-image", presentationID, "invalid_argument", "empty presentationId", nil)
	}
	if objectID == "" {
		return NewEditError("slides", "replace-image", presentationID, "invalid_argument", "empty object-id", nil)
	}
	if sourceURL == "" {
		return NewEditError("slides", "replace-image", presentationID, "invalid_argument", "empty source-url", nil)
	}

	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				ReplaceImage: &slides.ReplaceImageRequest{
					ImageObjectId:              objectID,
					ImageReplaceMethod:         "CENTER_INSIDE",
					Url:                        sourceURL,
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("slides", "replace-image", presentationID, "invalid_json", "decode execute-from-file failed", err)
	}

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("slides", "replace-image", presentationID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("slides", "replace-image", presentationID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"presentationId": presentationID,
			"objectId":       objectID,
			"sourceUrl":      sourceURL,
			"requestHash":    requestHash,
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
		u.Out().Printf("id\t%s", presentationID)
		u.Out().Printf("object\t%s", objectID)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return SlidesDryRunOutput(ctx, u, presentationID, req, map[string]any{
			"objectId":          objectID,
			"sourceUrl":         sourceURL,
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
		return NewEditError("slides", "replace-image", presentationID, "service_init_failed", "create slides service failed", err)
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, req).Do()
	if err != nil {
		if IsNotFound(err) {
			return NewEditError("slides", "replace-image", presentationID, "presentation_not_found",
				fmt.Sprintf("presentation not found (id=%s)", presentationID), err)
		}
		return NewEditError("slides", "replace-image", presentationID, "api_error", "replace image failed", err)
	}

	payload := map[string]any{
		"presentationId": presentationID,
		"objectId":       objectID,
		"replies":        len(resp.Replies),
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", presentationID)
	u.Out().Printf("object\t%s", objectID)
	u.Out().Printf("replies\t%d", len(resp.Replies))
	return nil
}

// SlidesEditCreateSlideCmd adds a new slide to a presentation.
type SlidesEditCreateSlideCmd struct {
	PresentationID string `arg:"" name:"presentationId" help:"Presentation ID"`
	Layout         string `name:"layout" help:"Slide layout type (BLANK, CAPTION_ONLY, TITLE, TITLE_AND_BODY, etc.)" default:"BLANK"`
	Index          int    `name:"index" help:"Insert position (0-based, -1 for end)" default:"-1"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditCreateSlideCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
	presentationID := strings.TrimSpace(c.PresentationID)
	layout := strings.TrimSpace(c.Layout)

	if presentationID == "" {
		return NewEditError("slides", "create-slide", presentationID, "invalid_argument", "empty presentationId", nil)
	}

	insertionIndex := int64(c.Index)
	if c.Index < 0 {
		insertionIndex = -1 // End of presentation
	}

	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateSlide: &slides.CreateSlideRequest{
					SlideLayoutReference: &slides.LayoutReference{
						PredefinedLayout: layout,
					},
					InsertionIndex: insertionIndex,
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("slides", "create-slide", presentationID, "invalid_json", "decode execute-from-file failed", err)
	}

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("slides", "create-slide", presentationID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("slides", "create-slide", presentationID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"presentationId": presentationID,
			"layout":         layout,
			"index":          c.Index,
			"requestHash":    requestHash,
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
		u.Out().Printf("id\t%s", presentationID)
		u.Out().Printf("layout\t%s", layout)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return SlidesDryRunOutput(ctx, u, presentationID, req, map[string]any{
			"layout":            layout,
			"index":             c.Index,
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
		return NewEditError("slides", "create-slide", presentationID, "service_init_failed", "create slides service failed", err)
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, req).Do()
	if err != nil {
		if IsNotFound(err) {
			return NewEditError("slides", "create-slide", presentationID, "presentation_not_found",
				fmt.Sprintf("presentation not found (id=%s)", presentationID), err)
		}
		return NewEditError("slides", "create-slide", presentationID, "api_error", "create slide failed", err)
	}

	newSlideID := ""
	if len(resp.Replies) > 0 && resp.Replies[0].CreateSlide != nil {
		newSlideID = resp.Replies[0].CreateSlide.ObjectId
	}

	payload := map[string]any{
		"presentationId": presentationID,
		"layout":         layout,
		"slideId":        newSlideID,
		"index":          c.Index,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", presentationID)
	u.Out().Printf("layout\t%s", layout)
	u.Out().Printf("slide-id\t%s", newSlideID)
	return nil
}

// SlidesEditDuplicateSlideCmd duplicates an existing slide.
type SlidesEditDuplicateSlideCmd struct {
	PresentationID string `arg:"" name:"presentationId" help:"Presentation ID"`
	SlideID        string `name:"slide-id" help:"ID of slide to duplicate"`
	Count          int    `name:"count" help:"Number of copies (default 1)" default:"1"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditDuplicateSlideCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
	presentationID := strings.TrimSpace(c.PresentationID)
	slideID := strings.TrimSpace(c.SlideID)

	if presentationID == "" {
		return NewEditError("slides", "duplicate-slide", presentationID, "invalid_argument", "empty presentationId", nil)
	}
	if slideID == "" {
		return NewEditError("slides", "duplicate-slide", presentationID, "invalid_argument", "empty slide-id", nil)
	}
	if c.Count < 1 {
		c.Count = 1
	}

	// Build requests for each duplicate
	req := &slides.BatchUpdatePresentationRequest{
		Requests: make([]*slides.Request, 0, c.Count),
	}
	for i := 0; i < c.Count; i++ {
		req.Requests = append(req.Requests, &slides.Request{
			DuplicateObject: &slides.DuplicateObjectRequest{
				ObjectId: slideID,
			},
		})
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("slides", "duplicate-slide", presentationID, "invalid_json", "decode execute-from-file failed", err)
	}

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("slides", "duplicate-slide", presentationID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("slides", "duplicate-slide", presentationID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"presentationId": presentationID,
			"slideId":        slideID,
			"count":          c.Count,
			"requestHash":    requestHash,
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
		u.Out().Printf("id\t%s", presentationID)
		u.Out().Printf("slide\t%s", slideID)
		u.Out().Printf("copies\t%d", c.Count)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return SlidesDryRunOutput(ctx, u, presentationID, req, map[string]any{
			"slideId":           slideID,
			"count":             c.Count,
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
		return NewEditError("slides", "duplicate-slide", presentationID, "service_init_failed", "create slides service failed", err)
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, req).Do()
	if err != nil {
		if IsNotFound(err) {
			return NewEditError("slides", "duplicate-slide", presentationID, "presentation_not_found",
				fmt.Sprintf("presentation not found (id=%s)", presentationID), err)
		}
		return NewEditError("slides", "duplicate-slide", presentationID, "api_error", "duplicate slide failed", err)
	}

	newSlideIDs := make([]string, 0, c.Count)
	for _, reply := range resp.Replies {
		if reply.DuplicateObject != nil {
			newSlideIDs = append(newSlideIDs, reply.DuplicateObject.ObjectId)
		}
	}

	payload := map[string]any{
		"presentationId": presentationID,
		"slideId":        slideID,
		"count":          c.Count,
		"newSlideIds":    newSlideIDs,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", presentationID)
	u.Out().Printf("slide\t%s", slideID)
	u.Out().Printf("copies\t%d", c.Count)
	u.Out().Printf("created\t%d", len(newSlideIDs))
	return nil
}

// SlidesEditRefreshChartsCmd refreshes embedded Google Sheets charts.
type SlidesEditRefreshChartsCmd struct {
	PresentationID string `arg:"" name:"presentationId" help:"Presentation ID"`
	ChartID        string `name:"chart-id" help:"Specific chart object ID to refresh"`
	All            bool   `name:"all" help:"Refresh all linked charts in presentation"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditRefreshChartsCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
	presentationID := strings.TrimSpace(c.PresentationID)
	chartID := strings.TrimSpace(c.ChartID)

	if presentationID == "" {
		return NewEditError("slides", "refresh-charts", presentationID, "invalid_argument", "empty presentationId", nil)
	}
	if !c.All && chartID == "" {
		return NewEditError("slides", "refresh-charts", presentationID, "invalid_argument", "specify --chart-id or --all", nil)
	}

	// Build request
	var batchReq *slides.BatchUpdatePresentationRequest
	if c.All {
		// For --all, we send a single refresh request (API refreshes all linked charts)
		batchReq = &slides.BatchUpdatePresentationRequest{
			Requests: []*slides.Request{
				{
					RefreshSheetsChart: &slides.RefreshSheetsChartRequest{},
				},
			},
		}
	} else {
		batchReq = &slides.BatchUpdatePresentationRequest{
			Requests: []*slides.Request{
				{
					RefreshSheetsChart: &slides.RefreshSheetsChartRequest{
						ObjectId: chartID,
					},
				},
			},
		}
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, batchReq); err != nil {
		return NewEditError("slides", "refresh-charts", presentationID, "invalid_json", "decode execute-from-file failed", err)
	}

	requestHash, hashErr := RequestHash(batchReq)
	if hashErr != nil {
		return NewEditError("slides", "refresh-charts", presentationID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, batchReq)
	if normErr != nil {
		return NewEditError("slides", "refresh-charts", presentationID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"presentationId": presentationID,
			"all":            c.All,
			"requestHash":    requestHash,
		}
		if !c.All {
			payload["chartId"] = chartID
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, err := NormalizedRequestString(batchReq); err == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", presentationID)
		if c.All {
			u.Out().Printf("mode\tall")
		} else {
			u.Out().Printf("chart\t%s", chartID)
		}
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		extra := map[string]any{
			"all":               c.All,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}
		if !c.All {
			extra["chartId"] = chartID
		}
		return SlidesDryRunOutput(ctx, u, presentationID, batchReq, extra, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSlidesService(ctx, account)
	if err != nil {
		return NewEditError("slides", "refresh-charts", presentationID, "service_init_failed", "create slides service failed", err)
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, batchReq).Do()
	if err != nil {
		if IsNotFound(err) {
			return NewEditError("slides", "refresh-charts", presentationID, "presentation_not_found",
				fmt.Sprintf("presentation not found (id=%s)", presentationID), err)
		}
		return NewEditError("slides", "refresh-charts", presentationID, "api_error", "refresh charts failed", err)
	}

	payload := map[string]any{
		"presentationId": presentationID,
		"all":            c.All,
		"replies":        len(resp.Replies),
	}
	if !c.All {
		payload["chartId"] = chartID
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", presentationID)
	if c.All {
		u.Out().Printf("mode\tall")
	} else {
		u.Out().Printf("chart\t%s", chartID)
	}
	u.Out().Printf("replies\t%d", len(resp.Replies))
	return nil
}

type SlidesEditUpdateNotesCmd struct {
	PresentationID string  `arg:"" name:"presentationId" help:"Presentation ID"`
	SlideID        string  `arg:"" name:"slideId" help:"Slide object ID"`
	Notes          *string `name:"notes" help:"Speaker notes text (use --notes '' to clear notes)"`
	NotesFile      string  `name:"notes-file" help:"Path to file containing speaker notes"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditUpdateNotesCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
	presentationID := strings.TrimSpace(c.PresentationID)
	slideID := strings.TrimSpace(c.SlideID)
	if presentationID == "" {
		return newSlidesEditError("update-notes", presentationID, "invalid_argument", "empty presentationId", nil)
	}
	if slideID == "" {
		return newSlidesEditError("update-notes", presentationID, "invalid_argument", "empty slideId", nil)
	}
	notes := ""
	updateNotes := false
	if strings.TrimSpace(c.NotesFile) != "" {
		data, err := os.ReadFile(c.NotesFile)
		if err != nil {
			return newSlidesEditError("update-notes", presentationID, "input_open_failed", "read notes-file failed", err)
		}
		notes = string(data)
		updateNotes = true
	} else if c.Notes != nil {
		notes = *c.Notes
		updateNotes = true
	}
	if !updateNotes {
		return newSlidesEditError("update-notes", presentationID, "invalid_argument", "provide --notes or --notes-file", nil)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSlidesService(ctx, account)
	if err != nil {
		return newSlidesEditError("update-notes", presentationID, "service_init_failed", "create slides service failed", err)
	}
	pres, err := svc.Presentations.Get(presentationID).Context(ctx).Do()
	if err != nil {
		return newSlidesEditError("update-notes", presentationID, "api_error", "get presentation failed", err)
	}

	notesObjectID := ""
	foundSlide := false
	for _, s := range pres.Slides {
		if s.ObjectId != slideID {
			continue
		}
		foundSlide = true
		if s.SlideProperties != nil && s.SlideProperties.NotesPage != nil {
			np := s.SlideProperties.NotesPage
			if np.NotesProperties != nil {
				notesObjectID = np.NotesProperties.SpeakerNotesObjectId
			}
			if notesObjectID == "" {
				for _, el := range np.PageElements {
					if el.Shape != nil && el.Shape.Placeholder != nil && el.Shape.Placeholder.Type == placeholderTypeBody {
						notesObjectID = el.ObjectId
						break
					}
				}
			}
		}
		break
	}
	if !foundSlide {
		return newSlidesEditError("update-notes", presentationID, "not_found", "slide not found", nil)
	}
	if notesObjectID == "" {
		return newSlidesEditError("update-notes", presentationID, "invalid_response", "speaker notes placeholder not found", nil)
	}

	requests := []*slides.Request{
		{
			DeleteText: &slides.DeleteTextRequest{
				ObjectId: notesObjectID,
				TextRange: &slides.Range{
					Type: "ALL",
				},
			},
		},
	}
	if notes != "" {
		requests = append(requests, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId: notesObjectID,
				Text:     notes,
			},
		})
	}
	req := &slides.BatchUpdatePresentationRequest{Requests: requests}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("slides", "update-notes", presentationID, "invalid_json", "decode execute-from-file failed", err)
	}
	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return newSlidesEditError("update-notes", presentationID, "invalid_request", "failed to hash request", hashErr)
	}
	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newSlidesEditError("update-notes", presentationID, "output_write_failed", "write normalized request failed", normErr)
	}
	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"presentationId": presentationID,
			"slideId":        slideID,
			"requestHash":    requestHash,
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
		u.Out().Printf("id\t%s", presentationID)
		u.Out().Printf("slide\t%s", slideID)
		return nil
	}
	if isEditDryRun(flags, c.Safety) {
		return SlidesDryRunOutput(ctx, u, presentationID, req, map[string]any{
			"slideId":           slideID,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}
	if _, err := svc.Presentations.BatchUpdate(presentationID, req).Context(ctx).Do(); err != nil {
		return newSlidesEditError("update-notes", presentationID, "api_error", "update notes failed", err)
	}
	payload := map[string]any{
		"presentationId": presentationID,
		"slideId":        slideID,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("Updated notes on slide %s", slideID)
	return nil
}

type SlidesEditDeleteSlideCmd struct {
	PresentationID string `arg:"" name:"presentationId" help:"Presentation ID"`
	SlideID        string `arg:"" name:"slideId" help:"Slide object ID to delete"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func (c *SlidesEditDeleteSlideCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
	presentationID := strings.TrimSpace(c.PresentationID)
	slideID := strings.TrimSpace(c.SlideID)
	if presentationID == "" {
		return newSlidesEditError("delete-slide", presentationID, "invalid_argument", "empty presentationId", nil)
	}
	if slideID == "" {
		return newSlidesEditError("delete-slide", presentationID, "invalid_argument", "empty slideId", nil)
	}
	if !isEditDryRun(flags, c.Safety) && !outfmt.IsJSON(ctx) && (flags == nil || !flags.Force) {
		return newSlidesEditError("delete-slide", presentationID, "confirmation_required", "delete-slide is destructive; rerun with --force or use --dry-run", nil)
	}
	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DeleteObject: &slides.DeleteObjectRequest{ObjectId: slideID},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("slides", "delete-slide", presentationID, "invalid_json", "decode execute-from-file failed", err)
	}
	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return newSlidesEditError("delete-slide", presentationID, "invalid_request", "failed to hash request", hashErr)
	}
	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newSlidesEditError("delete-slide", presentationID, "output_write_failed", "write normalized request failed", normErr)
	}
	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"presentationId": presentationID,
			"slideId":        slideID,
			"requestHash":    requestHash,
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
		u.Out().Printf("id\t%s", presentationID)
		return nil
	}
	if isEditDryRun(flags, c.Safety) {
		return SlidesDryRunOutput(ctx, u, presentationID, req, map[string]any{
			"slideId":           slideID,
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
		return newSlidesEditError("delete-slide", presentationID, "service_init_failed", "create slides service failed", err)
	}
	if _, err := svc.Presentations.BatchUpdate(presentationID, req).Context(ctx).Do(); err != nil {
		return newSlidesEditError("delete-slide", presentationID, "api_error", "delete slide failed", err)
	}
	payload := map[string]any{
		"presentationId": presentationID,
		"slideId":        slideID,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("Deleted slide %s", slideID)
	return nil
}

// SlidesEditInsertTableCmd inserts a data table into a slide.
type SlidesEditInsertTableCmd struct {
	PresentationID string `arg:"" name:"presentationId" help:"Presentation ID"`
	SlideID        string `name:"slide-id" help:"Slide ID to insert table into"`
	Rows           int    `name:"rows" help:"Number of rows" default:"3"`
	Columns        int    `name:"columns" help:"Number of columns" default:"3"`
	DataFile       string `name:"data-file" help:"Path to JSON array for table data (optional)"`
	Safety         AgenticEditSafetyFlags `embed:""`
}

func parseSlidesTableData(path string) ([][]string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw [][]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(raw))
	for _, rawRow := range raw {
		row := make([]string, 0, len(rawRow))
		for _, cell := range rawRow {
			row = append(row, fmt.Sprintf("%v", cell))
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (c *SlidesEditInsertTableCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "slides")
	presentationID := strings.TrimSpace(c.PresentationID)
	slideID := strings.TrimSpace(c.SlideID)

	if presentationID == "" {
		return NewEditError("slides", "insert-table", presentationID, "invalid_argument", "empty presentationId", nil)
	}
	if slideID == "" {
		return NewEditError("slides", "insert-table", presentationID, "invalid_argument", "empty slide-id", nil)
	}
	if c.Rows < 1 || c.Rows > 20 {
		return NewEditError("slides", "insert-table", presentationID, "invalid_argument", "rows must be 1-20", nil)
	}
	if c.Columns < 1 || c.Columns > 20 {
		return NewEditError("slides", "insert-table", presentationID, "invalid_argument", "columns must be 1-20", nil)
	}

	// Generate unique object ID for the table
	tableID := fmt.Sprintf("table_%d", time.Now().Unix())

	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateTable: &slides.CreateTableRequest{
					ObjectId: tableID,
					ElementProperties: &slides.PageElementProperties{
						PageObjectId: slideID,
						Size: &slides.Size{
							Height: &slides.Dimension{Magnitude: 300, Unit: "PT"},
							Width:  &slides.Dimension{Magnitude: 400, Unit: "PT"},
						},
						Transform: &slides.AffineTransform{
							TranslateX: 50,
							TranslateY: 50,
							ScaleX:     1,
							ScaleY:     1,
						},
					},
					Rows:    int64(c.Rows),
					Columns: int64(c.Columns),
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return NewEditError("slides", "insert-table", presentationID, "invalid_json", "decode execute-from-file failed", err)
	}
	dataRows, dataErr := parseSlidesTableData(c.DataFile)
	if dataErr != nil {
		return NewEditError("slides", "insert-table", presentationID, "invalid_json", "parse data-file failed", dataErr)
	}
	filledCells := 0
	if len(dataRows) > 0 {
		for rowIdx, row := range dataRows {
			if rowIdx >= c.Rows {
				break
			}
			for colIdx, text := range row {
				if colIdx >= c.Columns {
					break
				}
				if strings.TrimSpace(text) == "" {
					continue
				}
				req.Requests = append(req.Requests, &slides.Request{
					InsertText: &slides.InsertTextRequest{
						ObjectId:       tableID,
						InsertionIndex: 0,
						CellLocation: &slides.TableCellLocation{
							RowIndex:    int64(rowIdx),
							ColumnIndex: int64(colIdx),
						},
						Text: text,
					},
				})
				filledCells++
			}
		}
	}

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return NewEditError("slides", "insert-table", presentationID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return NewEditError("slides", "insert-table", presentationID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":   true,
			"valid":          true,
			"presentationId": presentationID,
			"slideId":        slideID,
			"rows":           c.Rows,
			"columns":        c.Columns,
			"tableId":        tableID,
			"requestHash":    requestHash,
			"filledCells":    filledCells,
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
		u.Out().Printf("id\t%s", presentationID)
		u.Out().Printf("slide\t%s", slideID)
		u.Out().Printf("table\t%dx%d", c.Rows, c.Columns)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return SlidesDryRunOutput(ctx, u, presentationID, req, map[string]any{
			"slideId":           slideID,
			"rows":              c.Rows,
			"columns":           c.Columns,
			"tableId":           tableID,
			"filledCells":       filledCells,
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
		return NewEditError("slides", "insert-table", presentationID, "service_init_failed", "create slides service failed", err)
	}

	resp, err := svc.Presentations.BatchUpdate(presentationID, req).Do()
	if err != nil {
		if IsNotFound(err) {
			return NewEditError("slides", "insert-table", presentationID, "presentation_not_found",
				fmt.Sprintf("presentation not found (id=%s)", presentationID), err)
		}
		return NewEditError("slides", "insert-table", presentationID, "api_error", "insert table failed", err)
	}

	payload := map[string]any{
		"presentationId": presentationID,
		"slideId":        slideID,
		"tableId":        tableID,
		"rows":           c.Rows,
		"columns":        c.Columns,
		"filledCells":    filledCells,
		"replies":        len(resp.Replies),
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", presentationID)
	u.Out().Printf("slide\t%s", slideID)
	u.Out().Printf("table\t%s", tableID)
	u.Out().Printf("size\t%dx%d", c.Rows, c.Columns)
	return nil
}

// newSlidesService creates a new Slides API service.
// This variable is overridden by the googleapi package to provide actual implementation.
var newSlidesService = func(ctx context.Context, account string) (*slides.Service, error) {
	return nil, errors.New("slides service not initialized - ensure googleapi/slides.go is imported")
}
