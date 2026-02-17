package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type SheetsEditCmd struct {
	Values SheetsEditValuesCmd `cmd:"" name:"values" help:"Update cell values in a range"`
	Append SheetsEditAppendCmd `cmd:"" name:"append" help:"Append rows to a sheet"`
	Clear  SheetsEditClearCmd  `cmd:"" name:"clear" help:"Clear values in a range"`
	Batch  SheetsEditBatchCmd  `cmd:"" name:"batch" help:"Apply multiple Sheets API batch operations from JSON"`
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
		valueInputOption = "USER_ENTERED"
	}

	req := &sheetsEditValuesRequest{
		SpreadsheetID:      spreadsheetID,
		Range:              rangeSpec,
		Values:             values,
		ValueInputOption:   valueInputOption,
		CopyValidationFrom: strings.TrimSpace(c.CopyValidationFrom),
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
			return outfmt.WriteJSON(os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		return nil
	}

	if c.Safety.DryRun {
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
		return outfmt.WriteJSON(os.Stdout, payload)
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
			return outfmt.WriteJSON(os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		return nil
	}

	if c.Safety.DryRun {
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
		return outfmt.WriteJSON(os.Stdout, payload)
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
	spreadsheetID := strings.TrimSpace(c.SpreadsheetID)
	rangeSpec := cleanRange(c.Range)
	if spreadsheetID == "" {
		return newSheetsEditError("clear", spreadsheetID, "invalid_argument", "empty spreadsheetId", usage("empty spreadsheetId"))
	}
	if strings.TrimSpace(rangeSpec) == "" {
		return newSheetsEditError("clear", spreadsheetID, "invalid_argument", "empty range", usage("empty range"))
	}
	if !c.Safety.DryRun && !outfmt.IsJSON(ctx) && (flags == nil || !flags.Force) {
		return newSheetsEditError("clear", spreadsheetID, "confirmation_required", "clear is destructive; rerun with --force or use --dry-run", usage("clear is destructive; rerun with --force or use --dry-run"))
	}

	req := &sheetsEditClearRequest{
		SpreadsheetID: spreadsheetID,
		Range:         rangeSpec,
	}
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
			return outfmt.WriteJSON(os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		return nil
	}

	if c.Safety.DryRun {
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
		return outfmt.WriteJSON(os.Stdout, payload)
	}
	u.Out().Printf("Cleared %s", resp.ClearedRange)
	return nil
}

type SheetsEditBatchCmd struct {
	SpreadsheetID string                `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	RequestsFile  string                `name:"requests-file" help:"Path to JSON request body, or '-' for stdin" default:"-"`
	Safety        SheetsEditSafetyFlags `embed:""`
}

func (c *SheetsEditBatchCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
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
		if sheetsRequestOperationCount(r) != 1 {
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
		requestKinds = append(requestKinds, sheetsRequestOperationName(r))
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
			return outfmt.WriteJSON(os.Stdout, payload)
		}
		u.Out().Printf("validate-only\ttrue")
		u.Out().Printf("valid\ttrue")
		u.Out().Printf("id\t%s", spreadsheetID)
		u.Out().Printf("operations\t%d", len(req.Requests))
		return nil
	}

	if c.Safety.DryRun {
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
		return outfmt.WriteJSON(os.Stdout, payload)
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
