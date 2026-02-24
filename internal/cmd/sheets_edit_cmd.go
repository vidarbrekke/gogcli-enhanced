package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

const (
	sheetsShiftRows    = "ROWS"
	sheetsShiftColumns = "COLUMNS"
)

type SheetsEditCmd struct {
	Values      SheetsEditValuesCmd      `cmd:"" name:"values" help:"Update cell values in a range"`
	Append      SheetsEditAppendCmd      `cmd:"" name:"append" help:"Append rows to a sheet"`
	Clear       SheetsEditClearCmd       `cmd:"" name:"clear" help:"Clear values in a range"`
	DeleteRange SheetsEditDeleteRangeCmd `cmd:"" name:"delete-range" help:"Delete a range and shift cells (ROWS or COLUMNS)"`
	MergeData   SheetsEditMergeDataCmd   `cmd:"" name:"merge-data" help:"Generate sheets from template using JSON data (mail-merge)"`
	ReplaceText SheetsEditReplaceTextCmd `cmd:"" name:"replace-text" help:"Find and replace text across sheet cells"`
	Format      SheetsEditFormatCmd      `cmd:"" name:"format" help:"Apply cell formatting in a range"`
	Insert      SheetsEditInsertCmd      `cmd:"" name:"insert" help:"Insert rows/columns in a sheet"`
	Batch       SheetsEditBatchCmd       `cmd:"" name:"batch" help:"Apply multiple Sheets API batch operations from JSON"`
}

type SheetsEditValuesCmd struct {
	SpreadsheetID      string                `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range              string                `arg:"" name:"range" help:"Range (eg. Sheet1!A1:B2)"`
	Values             []string              `arg:"" optional:"" name:"values" help:"Values (comma-separated rows, pipe-separated cells)"`
	ValueInput         string                `name:"input" help:"Value input option: RAW or USER_ENTERED" default:"USER_ENTERED"`
	ValuesJSON         string                `name:"values-json" help:"Values as JSON 2D array"`
	CopyValidationFrom string                `name:"copy-validation-from" help:"Copy data validation from an A1 range (eg. 'Sheet1!A2:D2') to the updated cells"`
	Safety             SheetsEditSafetyFlags `embed:""`
}

func (c *SheetsEditValuesCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "sheets")
	spreadsheetID := strings.TrimSpace(c.SpreadsheetID)
	rangeSpec := cleanRange(c.Range)
	if spreadsheetID == "" {
		return newSheetsEditError("values", spreadsheetID, "invalid_argument", "empty spreadsheetId", usage("empty spreadsheetId"))
	}
	if strings.TrimSpace(rangeSpec) == "" {
		return newSheetsEditError("values", spreadsheetID, "invalid_argument", "empty range", usage("empty range"))
	}

	values, err := sheetsParseValues(c.ValuesJSON, c.Values)
	if err != nil {
		return newSheetsEditError("values", spreadsheetID, "invalid_argument", "invalid values", err)
	}

	valueInputOption := strings.TrimSpace(c.ValueInput)
	if valueInputOption == "" {
		valueInputOption = sheetsValueInputUserEntered
	}

	req := &sheetsEditValuesRequest{
		SpreadsheetID:      spreadsheetID,
		Range:              rangeSpec,
		Values:             values,
		ValueInputOption:   valueInputOption,
		CopyValidationFrom: strings.TrimSpace(c.CopyValidationFrom),
	}
	if _, decodeErr := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); decodeErr != nil {
		return newSheetsEditError("values", spreadsheetID, "invalid_json", "decode execute-from-file failed", decodeErr)
	}
	spreadsheetID = normalizeGoogleID(strings.TrimSpace(req.SpreadsheetID))
	rangeSpec = cleanRange(req.Range)
	values = req.Values
	valueInputOption = strings.TrimSpace(req.ValueInputOption)
	if valueInputOption == "" {
		valueInputOption = sheetsValueInputUserEntered
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newSheetsEditError("values", spreadsheetID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		hash, _ := RequestHash(req)
		payload := map[string]any{
			"validateOnly":  true,
			"valid":         true,
			"spreadsheetId": spreadsheetID,
			"requestHash":   hash,
		}
		if normalizedForJSON != "" {
			payload["normalizedRequest"] = normalizedForJSON
		}
		if c.Safety.Pretty {
			if pretty, prettyErr := NormalizedRequestString(req); prettyErr == nil {
				payload["normalizedRequest"] = pretty
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return SheetsDryRunOutput(ctx, u, spreadsheetID, req, map[string]any{
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return newSheetsEditError("values", spreadsheetID, "service_init_failed", "create sheets service failed", err)
	}

	vr := &sheets.ValueRange{Values: values}
	call := svc.Spreadsheets.Values.Update(spreadsheetID, rangeSpec, vr).ValueInputOption(valueInputOption)
	resp, err := call.Do()
	if err != nil {
		if isSheetsNotFound(err) {
			return newSheetsEditError("values", spreadsheetID, "sheet_not_found", fmt.Sprintf("spreadsheet not found (id=%s)", spreadsheetID), err)
		}
		return newSheetsEditError("values", spreadsheetID, "api_error", "values update failed", err)
	}
	if strings.TrimSpace(c.CopyValidationFrom) != "" {
		if strings.TrimSpace(resp.UpdatedRange) == "" {
			return newSheetsEditError("values", spreadsheetID, "invalid_response", "update response missing updated range for validation copy", errors.New("missing updated range"))
		}
		if err := copyDataValidation(ctx, svc, spreadsheetID, c.CopyValidationFrom, resp.UpdatedRange); err != nil {
			return newSheetsEditError("values", spreadsheetID, "api_error", "copy data validation failed", err)
		}
	}

	payload := map[string]any{
		"updatedRange":   resp.UpdatedRange,
		"updatedRows":    resp.UpdatedRows,
		"updatedColumns": resp.UpdatedColumns,
		"updatedCells":   resp.UpdatedCells,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		if hash, err := RequestHash(req); err == nil {
			payload["requestHash"] = hash
		}
		if norm, err := NormalizedRequestString(req); err == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("Updated %d cells in %s", resp.UpdatedCells, resp.UpdatedRange)
	return nil
}

type SheetsEditAppendCmd struct {
	SpreadsheetID      string                `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range              string                `arg:"" name:"range" help:"Range (eg. Sheet1!A:C)"`
	Values             []string              `arg:"" optional:"" name:"values" help:"Values (comma-separated rows, pipe-separated cells)"`
	ValueInput         string                `name:"input" help:"Value input option: RAW or USER_ENTERED" default:"USER_ENTERED"`
	Insert             string                `name:"insert" help:"Insert data option: OVERWRITE or INSERT_ROWS"`
	ValuesJSON         string                `name:"values-json" help:"Values as JSON 2D array"`
	CopyValidationFrom string                `name:"copy-validation-from" help:"Copy data validation from an A1 range (eg. 'Sheet1!A2:D2') to the appended cells"`
	Safety             SheetsEditSafetyFlags `embed:""`
}

func (c *SheetsEditAppendCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "sheets")
	spreadsheetID := strings.TrimSpace(c.SpreadsheetID)
	rangeSpec := cleanRange(c.Range)
	if spreadsheetID == "" {
		return newSheetsEditError("append", spreadsheetID, "invalid_argument", "empty spreadsheetId", usage("empty spreadsheetId"))
	}
	if strings.TrimSpace(rangeSpec) == "" {
		return newSheetsEditError("append", spreadsheetID, "invalid_argument", "empty range", usage("empty range"))
	}

	values, err := sheetsParseValues(c.ValuesJSON, c.Values)
	if err != nil {
		return newSheetsEditError("append", spreadsheetID, "invalid_argument", "invalid values", err)
	}

	valueInputOption := strings.TrimSpace(c.ValueInput)
	if valueInputOption == "" {
		valueInputOption = "USER_ENTERED"
	}
	insertDataOption := strings.TrimSpace(c.Insert)

	req := &sheetsEditAppendRequest{
		SpreadsheetID:      spreadsheetID,
		Range:              rangeSpec,
		Values:             values,
		ValueInputOption:   valueInputOption,
		InsertDataOption:   insertDataOption,
		CopyValidationFrom: strings.TrimSpace(c.CopyValidationFrom),
	}
	if _, decodeErr := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); decodeErr != nil {
		return newSheetsEditError("append", spreadsheetID, "invalid_json", "decode execute-from-file failed", decodeErr)
	}
	spreadsheetID = normalizeGoogleID(strings.TrimSpace(req.SpreadsheetID))
	rangeSpec = cleanRange(req.Range)
	values = req.Values
	valueInputOption = strings.TrimSpace(req.ValueInputOption)
	if valueInputOption == "" {
		valueInputOption = "USER_ENTERED"
	}
	insertDataOption = strings.TrimSpace(req.InsertDataOption)
	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newSheetsEditError("append", spreadsheetID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		hash, _ := RequestHash(req)
		payload := map[string]any{
			"validateOnly":  true,
			"valid":         true,
			"spreadsheetId": spreadsheetID,
			"requestHash":   hash,
		}
		if normalizedForJSON != "" {
			payload["normalizedRequest"] = normalizedForJSON
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return SheetsDryRunOutput(ctx, u, spreadsheetID, req, map[string]any{
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return newSheetsEditError("append", spreadsheetID, "service_init_failed", "create sheets service failed", err)
	}

	vr := &sheets.ValueRange{Values: values}
	call := svc.Spreadsheets.Values.Append(spreadsheetID, rangeSpec, vr).ValueInputOption(valueInputOption)
	if insertDataOption != "" {
		call = call.InsertDataOption(insertDataOption)
	}
	resp, err := call.Do()
	if err != nil {
		if isSheetsNotFound(err) {
			return newSheetsEditError("append", spreadsheetID, "sheet_not_found", fmt.Sprintf("spreadsheet not found (id=%s)", spreadsheetID), err)
		}
		return newSheetsEditError("append", spreadsheetID, "api_error", "append failed", err)
	}
	if strings.TrimSpace(c.CopyValidationFrom) != "" {
		if resp.Updates == nil || strings.TrimSpace(resp.Updates.UpdatedRange) == "" {
			return newSheetsEditError("append", spreadsheetID, "invalid_response", "append response missing updated range for validation copy", errors.New("missing updated range"))
		}
		if err := copyDataValidation(ctx, svc, spreadsheetID, c.CopyValidationFrom, resp.Updates.UpdatedRange); err != nil {
			return newSheetsEditError("append", spreadsheetID, "api_error", "copy data validation failed", err)
		}
	}

	payload := map[string]any{
		"updatedRange":   resp.Updates.UpdatedRange,
		"updatedRows":    resp.Updates.UpdatedRows,
		"updatedColumns": resp.Updates.UpdatedColumns,
		"updatedCells":   resp.Updates.UpdatedCells,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		if hash, err := RequestHash(req); err == nil {
			payload["requestHash"] = hash
		}
		if norm, err := NormalizedRequestString(req); err == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("Appended %d cells to %s", resp.Updates.UpdatedCells, resp.Updates.UpdatedRange)
	return nil
}

type SheetsEditClearCmd struct {
	SpreadsheetID string                `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range         string                `arg:"" name:"range" help:"Range (eg. Sheet1!A1:B2)"`
	Safety        SheetsEditSafetyFlags `embed:""`
}

func (c *SheetsEditClearCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "sheets")
	spreadsheetID := strings.TrimSpace(c.SpreadsheetID)
	rangeSpec := cleanRange(c.Range)
	if spreadsheetID == "" {
		return newSheetsEditError("clear", spreadsheetID, "invalid_argument", "empty spreadsheetId", usage("empty spreadsheetId"))
	}
	if strings.TrimSpace(rangeSpec) == "" {
		return newSheetsEditError("clear", spreadsheetID, "invalid_argument", "empty range", usage("empty range"))
	}
	if !isEditDryRun(flags, c.Safety) && !outfmt.IsJSON(ctx) && (flags == nil || !flags.Force) {
		return newSheetsEditError("clear", spreadsheetID, "confirmation_required", "clear is destructive; rerun with --force or use --dry-run", usage("clear is destructive; rerun with --force or use --dry-run"))
	}

	req := &sheetsEditClearRequest{
		SpreadsheetID: spreadsheetID,
		Range:         rangeSpec,
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return newSheetsEditError("clear", spreadsheetID, "invalid_json", "decode execute-from-file failed", err)
	}
	spreadsheetID = normalizeGoogleID(strings.TrimSpace(req.SpreadsheetID))
	rangeSpec = cleanRange(req.Range)
	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newSheetsEditError("clear", spreadsheetID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		hash, _ := RequestHash(req)
		payload := map[string]any{
			"validateOnly":  true,
			"valid":         true,
			"spreadsheetId": spreadsheetID,
			"requestHash":   hash,
		}
		if normalizedForJSON != "" {
			payload["normalizedRequest"] = normalizedForJSON
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return SheetsDryRunOutput(ctx, u, spreadsheetID, req, map[string]any{
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return newSheetsEditError("clear", spreadsheetID, "service_init_failed", "create sheets service failed", err)
	}

	resp, err := svc.Spreadsheets.Values.Clear(spreadsheetID, rangeSpec, &sheets.ClearValuesRequest{}).Do()
	if err != nil {
		if isSheetsNotFound(err) {
			return newSheetsEditError("clear", spreadsheetID, "sheet_not_found", fmt.Sprintf("spreadsheet not found (id=%s)", spreadsheetID), err)
		}
		return newSheetsEditError("clear", spreadsheetID, "api_error", "clear failed", err)
	}

	payload := map[string]any{
		"clearedRange": resp.ClearedRange,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		if hash, err := RequestHash(req); err == nil {
			payload["requestHash"] = hash
		}
		if norm, err := NormalizedRequestString(req); err == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("Cleared %s", resp.ClearedRange)
	return nil
}

// SheetsEditDeleteRangeCmd deletes a range and shifts cells (ROWS or COLUMNS). VID-111.
type SheetsEditDeleteRangeCmd struct {
	SpreadsheetID  string                `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range          string                `arg:"" name:"range" help:"Range to delete (eg. Sheet1!A1:C10)"`
	ShiftDimension string                `name:"shift-dimension" help:"How to shift: ROWS or COLUMNS" enum:"ROWS,COLUMNS" default:"ROWS"`
	Safety         SheetsEditSafetyFlags `embed:""`
}

type sheetsEditDeleteRangeRequest struct {
	SpreadsheetID  string `json:"spreadsheetId"`
	Range          string `json:"range"`
	ShiftDimension string `json:"shiftDimension"`
}

func (c *SheetsEditDeleteRangeCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "sheets")
	spreadsheetID := normalizeGoogleID(strings.TrimSpace(c.SpreadsheetID))
	rangeSpec := cleanRange(c.Range)
	shiftDim := strings.TrimSpace(strings.ToUpper(c.ShiftDimension))
	if spreadsheetID == "" {
		return newSheetsEditError("delete-range", spreadsheetID, "invalid_argument", "empty spreadsheetId", usage("empty spreadsheetId"))
	}
	if strings.TrimSpace(rangeSpec) == "" {
		return newSheetsEditError("delete-range", spreadsheetID, "invalid_argument", "empty range", usage("empty range"))
	}
	if shiftDim != sheetsShiftRows && shiftDim != sheetsShiftColumns {
		return newSheetsEditError("delete-range", spreadsheetID, "invalid_argument", "shift-dimension must be ROWS or COLUMNS", usage("shift-dimension must be ROWS or COLUMNS"))
	}
	if _, err := parseSheetRange(rangeSpec, "delete-range"); err != nil {
		return newSheetsEditError("delete-range", spreadsheetID, "invalid_argument", "range must include sheet name (eg. Sheet1!A1:C10)", err)
	}
	if !isEditDryRun(flags, c.Safety) && !outfmt.IsJSON(ctx) && (flags == nil || !flags.Force) {
		return newSheetsEditError("delete-range", spreadsheetID, "confirmation_required", "delete-range is destructive; rerun with --force or use --dry-run", usage("delete-range is destructive; rerun with --force or use --dry-run"))
	}

	req := &sheetsEditDeleteRangeRequest{
		SpreadsheetID:  spreadsheetID,
		Range:          rangeSpec,
		ShiftDimension: shiftDim,
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return newSheetsEditError("delete-range", spreadsheetID, "invalid_json", "decode execute-from-file failed", err)
	}
	spreadsheetID = normalizeGoogleID(strings.TrimSpace(req.SpreadsheetID))
	rangeSpec = cleanRange(req.Range)
	shiftDim = strings.TrimSpace(strings.ToUpper(req.ShiftDimension))
	if shiftDim == "" {
		shiftDim = sheetsShiftRows
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newSheetsEditError("delete-range", spreadsheetID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		hash, _ := RequestHash(req)
		payload := map[string]any{
			"validateOnly":  true,
			"valid":         true,
			"spreadsheetId": spreadsheetID,
			"requestHash":   hash,
		}
		if normalizedForJSON != "" {
			payload["normalizedRequest"] = normalizedForJSON
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return SheetsDryRunOutput(ctx, u, spreadsheetID, req, map[string]any{
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return newSheetsEditError("delete-range", spreadsheetID, "service_init_failed", "create sheets service failed", err)
	}
	rangeInfo, err := parseSheetRange(rangeSpec, "delete-range")
	if err != nil {
		return newSheetsEditError("delete-range", spreadsheetID, "invalid_argument", "invalid range", err)
	}
	sheetIDs, err := fetchSheetIDMap(ctx, svc, spreadsheetID)
	if err != nil {
		return newSheetsEditError("delete-range", spreadsheetID, "api_error", "load sheet metadata failed", err)
	}
	gridRange, err := gridRangeFromMap(rangeInfo, sheetIDs, "delete-range")
	if err != nil {
		return newSheetsEditError("delete-range", spreadsheetID, "invalid_argument", "invalid grid range", err)
	}
	batchReq := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteRange: &sheets.DeleteRangeRequest{
					Range:          gridRange,
					ShiftDimension: shiftDim,
				},
			},
		},
	}
	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, batchReq).Do()
	if err != nil {
		if isSheetsNotFound(err) {
			return newSheetsEditError("delete-range", spreadsheetID, "sheet_not_found", fmt.Sprintf("spreadsheet not found (id=%s)", spreadsheetID), err)
		}
		return newSheetsEditError("delete-range", spreadsheetID, "api_error", "delete-range failed", err)
	}

	payload := map[string]any{
		"deletedRange":   rangeSpec,
		"shiftDimension": shiftDim,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if c.Safety.Pretty {
		if hash, err := RequestHash(req); err == nil {
			payload["requestHash"] = hash
		}
		if norm, err := NormalizedRequestString(req); err == nil {
			payload["normalizedRequest"] = norm
		}
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("Deleted %s (shift %s)", rangeSpec, shiftDim)
	return nil
}

type SheetsEditFormatCmd struct {
	SpreadsheetID string                `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range         string                `arg:"" name:"range" help:"Range (eg. Sheet1!A1:B2)"`
	FormatJSON    string                `name:"format-json" help:"Cell format as JSON (Sheets API CellFormat)"`
	FormatFields  string                `name:"format-fields" help:"Format field mask (eg. userEnteredFormat.textFormat.bold)"`
	Safety        SheetsEditSafetyFlags `embed:""`
}

func (c *SheetsEditFormatCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "sheets")
	spreadsheetID := normalizeGoogleID(strings.TrimSpace(c.SpreadsheetID))
	rangeSpec := cleanRange(c.Range)
	if spreadsheetID == "" {
		return newSheetsEditError("format", spreadsheetID, "invalid_argument", "empty spreadsheetId", usage("empty spreadsheetId"))
	}
	if strings.TrimSpace(rangeSpec) == "" {
		return newSheetsEditError("format", spreadsheetID, "invalid_argument", "empty range", usage("empty range"))
	}
	if strings.TrimSpace(c.FormatJSON) == "" {
		return newSheetsEditError("format", spreadsheetID, "invalid_argument", "empty format-json", usage("empty format-json"))
	}
	if strings.TrimSpace(c.FormatFields) == "" {
		return newSheetsEditError("format", spreadsheetID, "invalid_argument", "empty format-fields", usage("empty format-fields"))
	}

	b, err := resolveInlineOrFileBytes(c.FormatJSON)
	if err != nil {
		return newSheetsEditError("format", spreadsheetID, "invalid_json", "read --format-json failed", err)
	}
	var format sheets.CellFormat
	if unmarshalErr := json.Unmarshal(b, &format); unmarshalErr != nil {
		return newSheetsEditError("format", spreadsheetID, "invalid_json", "invalid format JSON", unmarshalErr)
	}
	formatFields := strings.TrimSpace(c.FormatFields)
	normalizedFields, formatJSONPaths := normalizeFormatMask(formatFields)
	if normalizedFields != "" {
		formatFields = normalizedFields
	}
	if forceFieldsErr := applyForceSendFields(&format, formatJSONPaths); forceFieldsErr != nil {
		return newSheetsEditError("format", spreadsheetID, "invalid_argument", "invalid format-fields", forceFieldsErr)
	}
	rangeInfo, err := parseSheetRange(rangeSpec, "format")
	if err != nil {
		return newSheetsEditError("format", spreadsheetID, "invalid_argument", "invalid range", err)
	}
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				RepeatCell: &sheets.RepeatCellRequest{
					Cell: &sheets.CellData{
						UserEnteredFormat: &format,
					},
					Fields: formatFields,
				},
			},
		},
	}
	if _, decodeErr := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); decodeErr != nil {
		return newSheetsEditError("format", spreadsheetID, "invalid_json", "decode execute-from-file failed", decodeErr)
	}
	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newSheetsEditError("format", spreadsheetID, "output_write_failed", "write normalized request failed", normErr)
	}
	if c.Safety.ValidateOnly {
		hash, _ := RequestHash(req)
		payload := map[string]any{
			"validateOnly":  true,
			"valid":         true,
			"spreadsheetId": spreadsheetID,
			"range":         rangeSpec,
			"fields":        formatFields,
			"requestHash":   hash,
		}
		if normalizedForJSON != "" || c.Safety.Pretty {
			if norm, normErr := NormalizedRequestString(req); normErr == nil {
				payload["normalizedRequest"] = norm
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		u.Out().Printf("range\t%s", rangeSpec)
		return nil
	}
	if isEditDryRun(flags, c.Safety) {
		return SheetsDryRunOutput(ctx, u, spreadsheetID, req, map[string]any{
			"range":             rangeSpec,
			"fields":            formatFields,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return newSheetsEditError("format", spreadsheetID, "service_init_failed", "create sheets service failed", err)
	}
	sheetIDs, err := fetchSheetIDMap(ctx, svc, spreadsheetID)
	if err != nil {
		return newSheetsEditError("format", spreadsheetID, "api_error", "load sheet metadata failed", err)
	}
	gridRange, err := gridRangeFromMap(rangeInfo, sheetIDs, "format")
	if err != nil {
		return newSheetsEditError("format", spreadsheetID, "invalid_argument", "invalid grid range", err)
	}
	req.Requests[0].RepeatCell.Range = gridRange
	if _, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, req).Do(); err != nil {
		return newSheetsEditError("format", spreadsheetID, "api_error", "format failed", err)
	}
	payload := map[string]any{
		"spreadsheetId": spreadsheetID,
		"range":         rangeSpec,
		"fields":        formatFields,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("Formatted %s", rangeSpec)
	return nil
}

type SheetsEditInsertCmd struct {
	SpreadsheetID string                `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Sheet         string                `arg:"" name:"sheet" help:"Sheet name (eg. Sheet1)"`
	Dimension     string                `arg:"" name:"dimension" help:"Dimension to insert: rows or cols"`
	Start         int64                 `arg:"" name:"start" help:"Position before which to insert (1-based)"`
	Count         int64                 `name:"count" help:"Number of rows/columns to insert" default:"1"`
	After         bool                  `name:"after" help:"Insert after the position instead of before"`
	Safety        SheetsEditSafetyFlags `embed:""`
}

func (c *SheetsEditInsertCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "sheets")
	spreadsheetID := normalizeGoogleID(strings.TrimSpace(c.SpreadsheetID))
	sheetName := strings.TrimSpace(c.Sheet)
	if spreadsheetID == "" {
		return newSheetsEditError("insert", spreadsheetID, "invalid_argument", "empty spreadsheetId", usage("empty spreadsheetId"))
	}
	if sheetName == "" {
		return newSheetsEditError("insert", spreadsheetID, "invalid_argument", "empty sheet", usage("empty sheet"))
	}
	dim := strings.ToLower(strings.TrimSpace(c.Dimension))
	var apiDimension string
	switch dim {
	case "rows", "row":
		apiDimension = sheetsShiftRows
	case "cols", "col", "columns", "column":
		apiDimension = sheetsShiftColumns
	default:
		return newSheetsEditError("insert", spreadsheetID, "invalid_argument", "dimension must be rows or cols", usage("dimension must be rows or cols"))
	}
	if c.Start < 1 {
		return newSheetsEditError("insert", spreadsheetID, "invalid_argument", "start must be >= 1", usage("start must be >= 1"))
	}
	if c.Count < 1 {
		return newSheetsEditError("insert", spreadsheetID, "invalid_argument", "count must be >= 1", usage("count must be >= 1"))
	}
	startIndex := c.Start - 1
	if c.After {
		startIndex = c.Start
	}
	endIndex := startIndex + c.Count
	inheritFromBefore := c.After
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				InsertDimension: &sheets.InsertDimensionRequest{
					Range: &sheets.DimensionRange{
						Dimension:  apiDimension,
						StartIndex: startIndex,
						EndIndex:   endIndex,
					},
					InheritFromBefore: inheritFromBefore,
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return newSheetsEditError("insert", spreadsheetID, "invalid_json", "decode execute-from-file failed", err)
	}
	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newSheetsEditError("insert", spreadsheetID, "output_write_failed", "write normalized request failed", normErr)
	}
	if c.Safety.ValidateOnly {
		hash, _ := RequestHash(req)
		payload := map[string]any{
			"validateOnly":      true,
			"valid":             true,
			"spreadsheetId":     spreadsheetID,
			"sheet":             sheetName,
			"dimension":         apiDimension,
			"start":             c.Start,
			"count":             c.Count,
			"after":             c.After,
			"inheritFromBefore": inheritFromBefore,
			"requestHash":       hash,
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
		u.Out().Printf("id\t%s", spreadsheetID)
		return nil
	}
	if isEditDryRun(flags, c.Safety) {
		return SheetsDryRunOutput(ctx, u, spreadsheetID, req, map[string]any{
			"sheet":             sheetName,
			"dimension":         apiDimension,
			"start":             c.Start,
			"count":             c.Count,
			"after":             c.After,
			"inheritFromBefore": inheritFromBefore,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return newSheetsEditError("insert", spreadsheetID, "service_init_failed", "create sheets service failed", err)
	}
	sheetIDs, err := fetchSheetIDMap(ctx, svc, spreadsheetID)
	if err != nil {
		return newSheetsEditError("insert", spreadsheetID, "api_error", "load sheet metadata failed", err)
	}
	sheetID, ok := sheetIDs[sheetName]
	if !ok {
		return newSheetsEditError("insert", spreadsheetID, "invalid_argument", "unknown sheet", usagef("unknown sheet %q", sheetName))
	}
	req.Requests[0].InsertDimension.Range.SheetId = sheetID
	if _, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, req).Do(); err != nil {
		return newSheetsEditError("insert", spreadsheetID, "api_error", "insert dimension failed", err)
	}
	payload := map[string]any{
		"spreadsheetId":     spreadsheetID,
		"sheet":             sheetName,
		"sheetId":           sheetID,
		"dimension":         apiDimension,
		"start":             c.Start,
		"count":             c.Count,
		"after":             c.After,
		"inheritFromBefore": inheritFromBefore,
		"startIndex":        startIndex,
		"endIndex":          endIndex,
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("Inserted %d into %s", c.Count, sheetName)
	return nil
}

type SheetsEditBatchCmd struct {
	SpreadsheetID string                `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	RequestsFile  string                `name:"requests-file" help:"Path to JSON request body, or '-' for stdin" default:"-"`
	Safety        SheetsEditSafetyFlags `embed:""`
}

func (c *SheetsEditBatchCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "sheets")
	spreadsheetID := strings.TrimSpace(c.SpreadsheetID)
	if spreadsheetID == "" {
		return newSheetsEditError("batch", spreadsheetID, "invalid_argument", "empty spreadsheetId", usage("empty spreadsheetId"))
	}
	requestsFile := strings.TrimSpace(c.RequestsFile)
	executeFromFile := strings.TrimSpace(c.Safety.ExecuteFromFile)
	if executeFromFile != "" && strings.TrimSpace(c.RequestsFile) != "-" && strings.TrimSpace(c.RequestsFile) != "" {
		return newSheetsEditError("batch", spreadsheetID, "invalid_argument", "cannot combine --execute-from-file with --requests-file", usage("cannot combine --execute-from-file with --requests-file"))
	}
	if executeFromFile != "" {
		requestsFile = executeFromFile
	}
	if requestsFile == "" {
		return newSheetsEditError("batch", spreadsheetID, "invalid_argument", "empty requests-file", usage("empty requests-file"))
	}

	var reader io.Reader = os.Stdin
	if requestsFile != "-" {
		f, openErr := os.Open(requestsFile) //nolint:gosec // user-provided path
		if openErr != nil {
			return newSheetsEditError("batch", spreadsheetID, "input_open_failed", "open requests-file failed", openErr)
		}
		defer f.Close()
		reader = f
	}

	var req sheets.BatchUpdateSpreadsheetRequest
	if err := json.NewDecoder(reader).Decode(&req); err != nil {
		return newSheetsEditError("batch", spreadsheetID, "invalid_json", "decode requests JSON failed", err)
	}
	if len(req.Requests) == 0 {
		return newSheetsEditError("batch", spreadsheetID, "invalid_argument", "batch request has no operations", usage("batch request has no operations"))
	}
	for i, r := range req.Requests {
		if RequestOperationCount(r) != 1 {
			idx := i
			err := newSheetsEditError("batch", spreadsheetID, "invalid_request", fmt.Sprintf("request[%d] must set exactly one operation field", i), usage(fmt.Sprintf("request[%d] must set exactly one operation field", i)))
			var ee *EditError
			if errors.As(err, &ee) {
				ee.RequestIndex = &idx
			}
			return err
		}
	}

	requestHash, hashErr := RequestHash(&req)
	if hashErr != nil {
		return newSheetsEditError("batch", spreadsheetID, "invalid_request", "failed to hash normalized request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, &req)
	if normErr != nil {
		return newSheetsEditError("batch", spreadsheetID, "output_write_failed", "write normalized request failed", normErr)
	}

	requestKinds := make([]string, 0, len(req.Requests))
	for _, r := range req.Requests {
		requestKinds = append(requestKinds, RequestOperationName(r))
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":  true,
			"valid":         true,
			"spreadsheetId": spreadsheetID,
			"operations":    len(req.Requests),
			"requestKinds":  requestKinds,
			"requestHash":   requestHash,
		}
		if normalizedForJSON != "" {
			payload["normalizedRequest"] = normalizedForJSON
		}
		if c.Safety.Pretty {
			if pretty, err := NormalizedRequestString(&req); err == nil {
				payload["normalizedRequest"] = pretty
			}
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		u.Out().Printf("operations\t%d", len(req.Requests))
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		return SheetsDryRunOutput(ctx, u, spreadsheetID, &req, map[string]any{
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
	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return newSheetsEditError("batch", spreadsheetID, "service_init_failed", "create sheets service failed", err)
	}
	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, &req).Do()
	if err != nil {
		if isSheetsNotFound(err) {
			return newSheetsEditError("batch", spreadsheetID, "sheet_not_found", fmt.Sprintf("spreadsheet not found (id=%s)", spreadsheetID), err)
		}
		return newSheetsEditError("batch", spreadsheetID, "api_error", "batch update failed", err)
	}

	payload := map[string]any{
		"spreadsheetId": spreadsheetID,
		"operations":    len(req.Requests),
		"replies":       len(resp.Replies),
	}
	if normalizedForJSON != "" {
		payload["normalizedRequest"] = normalizedForJSON
	}
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, payload)
	}
	u.Out().Printf("id\t%s", spreadsheetID)
	u.Out().Printf("operations\t%d", len(req.Requests))
	u.Out().Printf("replies\t%d", len(resp.Replies))
	return nil
}

type sheetsEditValuesRequest struct {
	SpreadsheetID      string          `json:"spreadsheetId"`
	Range              string          `json:"range"`
	Values             [][]interface{} `json:"values"`
	ValueInputOption   string          `json:"valueInputOption"`
	CopyValidationFrom string          `json:"copyValidationFrom,omitempty"`
}

type sheetsEditAppendRequest struct {
	SpreadsheetID      string          `json:"spreadsheetId"`
	Range              string          `json:"range"`
	Values             [][]interface{} `json:"values"`
	ValueInputOption   string          `json:"valueInputOption"`
	InsertDataOption   string          `json:"insertDataOption,omitempty"`
	CopyValidationFrom string          `json:"copyValidationFrom,omitempty"`
}

type sheetsEditClearRequest struct {
	SpreadsheetID string `json:"spreadsheetId"`
	Range         string `json:"range"`
}

func sheetsParseValues(valuesJSON string, valuesArgs []string) ([][]interface{}, error) {
	switch {
	case strings.TrimSpace(valuesJSON) != "":
		var values [][]interface{}
		if err := json.Unmarshal([]byte(valuesJSON), &values); err != nil {
			return nil, fmt.Errorf("invalid JSON values: %w", err)
		}
		return values, nil
	case len(valuesArgs) > 0:
		rawValues := strings.Join(valuesArgs, " ")
		rows := strings.Split(rawValues, ",")
		values := make([][]interface{}, 0, len(rows))
		for _, row := range rows {
			cells := strings.Split(strings.TrimSpace(row), "|")
			rowData := make([]interface{}, len(cells))
			for i, cell := range cells {
				rowData[i] = strings.TrimSpace(cell)
			}
			values = append(values, rowData)
		}
		return values, nil
	default:
		return nil, errors.New("provide values as args or via --values-json")
	}
}

// SheetsEditReplaceTextCmd finds and replaces text across sheet cells.
type SheetsEditReplaceTextCmd struct {
	SpreadsheetID   string                 `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Find            string                 `name:"find" help:"Text to find"`
	Replace         string                 `name:"replace" help:"Replacement text"`
	SheetID         int64                  `name:"sheet-id" help:"Sheet ID to search in; omit to search all sheets"`
	AllSheets       bool                   `name:"all-sheets" help:"Search all sheets in the spreadsheet"`
	MatchCase       bool                   `name:"match-case" help:"Case-sensitive matching"`
	MatchEntireCell bool                   `name:"match-entire-cell" help:"Match only if find value is entire cell content"`
	UseRegex        bool                   `name:"regex" help:"Find value is a regular expression (Java pattern syntax)"`
	IncludeFormulas bool                   `name:"formulas" help:"Search formula cells in addition to values"`
	Safety          AgenticEditSafetyFlags `embed:""`
}

func (c *SheetsEditReplaceTextCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "sheets")
	spreadsheetID := strings.TrimSpace(c.SpreadsheetID)
	find := strings.TrimSpace(c.Find)
	replace := c.Replace

	if spreadsheetID == "" {
		return newSheetsEditError("replace-text", spreadsheetID, "invalid_argument", "empty spreadsheetId", nil)
	}
	if find == "" {
		return newSheetsEditError("replace-text", spreadsheetID, "invalid_argument", "empty find", nil)
	}

	// Build the batch request with FindReplace operation
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				FindReplace: &sheets.FindReplaceRequest{
					Find:            find,
					Replacement:     replace,
					MatchCase:       c.MatchCase,
					MatchEntireCell: c.MatchEntireCell,
					SearchByRegex:   c.UseRegex,
					IncludeFormulas: c.IncludeFormulas,
					AllSheets:       c.AllSheets,
					SheetId:         c.SheetID, // 0 if not specified, which includes all sheets when AllSheets is true
				},
			},
		},
	}
	if _, err := DecodeExecuteRequestIfProvided(c.Safety.ExecuteFromFile, req); err != nil {
		return newSheetsEditError("replace-text", spreadsheetID, "invalid_json", "decode execute-from-file failed", err)
	}

	requestHash, hashErr := RequestHash(req)
	if hashErr != nil {
		return newSheetsEditError("replace-text", spreadsheetID, "invalid_request", "failed to hash request", hashErr)
	}

	normalizedForJSON, normErr := NormalizedRequestForOutput(ctx, c.Safety.OutputRequestFile, req)
	if normErr != nil {
		return newSheetsEditError("replace-text", spreadsheetID, "output_write_failed", "write normalized request failed", normErr)
	}

	if c.Safety.ValidateOnly {
		payload := map[string]any{
			"validateOnly":    true,
			"valid":           true,
			"spreadsheetId":   spreadsheetID,
			"find":            find,
			"replace":         replace,
			"matchCase":       c.MatchCase,
			"matchEntireCell": c.MatchEntireCell,
			"useRegex":        c.UseRegex,
			"includeFormulas": c.IncludeFormulas,
			"allSheets":       c.AllSheets,
			"requestHash":     requestHash,
		}
		if c.SheetID != 0 {
			payload["sheetId"] = c.SheetID
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
		u.Out().Printf("id\t%s", spreadsheetID)
		u.Out().Printf("find\t%s", find)
		u.Out().Printf("replace\t%s", replace)
		return nil
	}

	if isEditDryRun(flags, c.Safety) {
		scope := "all sheets"
		if !c.AllSheets && c.SheetID != 0 {
			scope = fmt.Sprintf("sheet %d", c.SheetID)
		}
		return DryRunOutput(ctx, u, "sheets", spreadsheetID, req, map[string]any{
			"find":              find,
			"replace":           replace,
			"scope":             scope,
			"matchCase":         c.MatchCase,
			"matchEntireCell":   c.MatchEntireCell,
			"useRegex":          c.UseRegex,
			"includeFormulas":   c.IncludeFormulas,
			"requestHash":       requestHash,
			"normalizedRequest": normalizedForJSON,
		}, c.Safety.Pretty)
	}

	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return newSheetsEditError("replace-text", spreadsheetID, "service_init_failed", "create sheets service failed", err)
	}

	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return newSheetsEditError("replace-text", spreadsheetID, "api_error", "find/replace failed", err)
	}

	var replacements int64
	if len(resp.Replies) > 0 && resp.Replies[0] != nil && resp.Replies[0].FindReplace != nil {
		replacements = resp.Replies[0].FindReplace.OccurrencesChanged
	}

	payload := map[string]any{
		"spreadsheetId":      spreadsheetID,
		"occurrencesChanged": replacements,
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
	u.Out().Printf("id\t%s", spreadsheetID)
	u.Out().Printf("replaced\t%d", replacements)
	return nil
}

// SheetsEditMergeDataCmd generates Google Sheets from a template using JSON data (mail-merge).
type SheetsEditMergeDataCmd struct {
	TemplateID       string                `arg:"" name:"templateId" help:"Template spreadsheet ID"`
	DataFile         string                `name:"data-file" help:"Path to JSON array of data objects"`
	OutputFolderID   string                `name:"output-folder-id" help:"Drive folder ID for output (default: same as template)"`
	FilenameFormat   string                `name:"filename-format" help:"Format for output filenames using {{placeholder}} syntax (default: 'Generated - {{name}}')"`
	IncludeTimestamp bool                  `name:"include-timestamp" help:"Append timestamp to filename for uniqueness"`
	Safety           SheetsEditSafetyFlags `embed:""`
}

func (c *SheetsEditMergeDataCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	warnRequireRevisionUnsupported(ctx, u, c.Safety, "sheets")
	templateID := strings.TrimSpace(normalizeGoogleID(c.TemplateID))
	dataFile := strings.TrimSpace(c.DataFile)

	if templateID == "" {
		return newSheetsEditError("merge-data", templateID, "invalid_argument", "empty templateId", nil)
	}
	if dataFile == "" {
		return newSheetsEditError("merge-data", templateID, "invalid_argument", "empty data-file", nil)
	}

	dataRecords, sampleRecord, err := loadMergeDataRecords(dataFile, func(code, msg string, cause error) error {
		return newSheetsEditError("merge-data", templateID, code, msg, cause)
	})
	if err != nil {
		return err
	}
	operations := buildMergeDataPreview(dataRecords, c.FilenameFormat, c.IncludeTimestamp, "FindReplace")

	requestHash, hashErr := RequestHash(dataRecords)
	if hashErr != nil {
		return newSheetsEditError("merge-data", templateID, "invalid_request", "failed to hash data", hashErr)
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
			"service":     "sheets",
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
		u.Out().Printf("service\tsheets")
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
		return newSheetsEditError("merge-data", templateID, "service_init_failed", "create drive service failed", err)
	}
	sheetsSvc, err := newSheetsService(ctx, account)
	if err != nil {
		return newSheetsEditError("merge-data", templateID, "service_init_failed", "create sheets service failed", err)
	}

	outputFolderID := resolveMergeDataOutputFolder(ctx, driveSvc, templateID, c.OutputFolderID)

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
		newSheetID := copied.Id

		// 2. FindReplace for each placeholder (all sheets, value cells only)
		req := &sheets.BatchUpdateSpreadsheetRequest{
			Requests: make([]*sheets.Request, 0, len(record)),
		}
		for key, value := range record {
			textValue := fmt.Sprintf("%v", value)
			req.Requests = append(req.Requests, &sheets.Request{
				FindReplace: &sheets.FindReplaceRequest{
					Find:            fmt.Sprintf("{{%s}}", key),
					Replacement:     textValue,
					AllSheets:       true,
					MatchCase:       false,
					SearchByRegex:   false,
					IncludeFormulas: false,
				},
			})
		}

		_, batchErr := sheetsSvc.Spreadsheets.BatchUpdate(newSheetID, req).Context(ctx).Do()
		if batchErr != nil {
			results = append(results, map[string]any{
				"index": i, "status": "failed", "error": batchErr.Error(),
				"stage": "batch-update", "spreadsheetId": newSheetID,
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
				fileMeta, getErr := driveSvc.Files.Get(newSheetID).Fields("parents").Context(ctx).Do()
				if getErr != nil {
					results = append(results, map[string]any{
						"index": i, "status": "failed", "error": getErr.Error(),
						"stage": "get-parents", "spreadsheetId": newSheetID,
					})
					failedCount++
					continue
				}
				removeParents := strings.Join(fileMeta.Parents, ",")
				moveCall := driveSvc.Files.Update(newSheetID, &drive.File{}).AddParents(outputFolderID)
				if strings.TrimSpace(removeParents) != "" {
					moveCall = moveCall.RemoveParents(removeParents)
				}
				if _, moveErr := moveCall.Context(ctx).Do(); moveErr != nil {
					results = append(results, map[string]any{
						"index": i, "status": "failed", "error": moveErr.Error(),
						"stage": "move-output", "spreadsheetId": newSheetID,
					})
					failedCount++
					continue
				}
			}
		}

		results = append(results, map[string]any{
			"index":         i,
			"status":        "success",
			"spreadsheetId": newSheetID,
			"title":         filename,
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
