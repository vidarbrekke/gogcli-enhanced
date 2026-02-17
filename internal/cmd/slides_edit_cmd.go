package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/api/slides/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SlidesEditCmd provides edit operations for Google Slides with agentic safety.
type SlidesEditCmd struct {
	Batch SlidesEditBatchCmd `cmd:"" name:"batch" help:"Apply multiple Slides API batch operations from JSON"`
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

// newSlidesService creates a new Slides API service.
// This variable is overridden by the googleapi package to provide actual implementation.
var newSlidesService = func(ctx context.Context, account string) (*slides.Service, error) {
	return nil, errors.New("slides service not initialized - ensure googleapi/slides.go is imported")
}
