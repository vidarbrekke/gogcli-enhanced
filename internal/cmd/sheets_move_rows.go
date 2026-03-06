package cmd

import (
	"context"
	"sort"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SheetsMoveRowsCmd filters rows by condition and copies or moves them to another sheet.
type SheetsMoveRowsCmd struct {
	SpreadsheetID   string `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range           string `arg:"" name:"range" help:"Source range (e.g. Sheet1!A2:J200); must include sheet name"`
	TargetSheet     string `arg:"" name:"targetSheet" help:"Destination sheet name"`
	Column          int    `name:"column" help:"Zero-based column index to filter on (0 = column A)" default:"0"`
	Op              string `name:"op" help:"Operator: eq, contains, gt, lt" enum:"eq,contains,gt,lt" default:"eq"`
	Value           string `name:"value" help:"Value to compare against"`
	Mode            string `name:"mode" help:"copy (default) or move" enum:"copy,move" default:"copy"`
	DestinationCell string `name:"destination-cell" help:"Start cell on target sheet (default A1)" default:"A1"`
}

// Run runs the sheets move-rows command.
func (c *SheetsMoveRowsCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	spreadsheetID := normalizeGoogleID(strings.TrimSpace(c.SpreadsheetID))
	rangeSpec := cleanRange(c.Range)
	targetSheet := strings.TrimSpace(c.TargetSheet)
	if spreadsheetID == "" {
		return usage("empty spreadsheetId")
	}
	if rangeSpec == "" {
		return usage("empty range")
	}
	if targetSheet == "" {
		return usage("empty targetSheet")
	}
	op := strings.TrimSpace(strings.ToLower(c.Op))
	if op == "" {
		op = "eq"
	}

	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return err
	}

	parsed, err := parseSheetRange(rangeSpec, "move-rows")
	if err != nil {
		return err
	}

	sheetIDs, err := fetchSheetIDMap(ctx, svc, spreadsheetID)
	if err != nil {
		return err
	}
	sourceSheetID, ok := sheetIDs[parsed.SheetName]
	if !ok {
		return usagef("unknown source sheet %q", parsed.SheetName)
	}

	resp, err := svc.Spreadsheets.Values.Get(spreadsheetID, rangeSpec).Context(ctx).Do()
	if err != nil {
		return err
	}

	if len(resp.Values) == 0 {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
				"spreadsheetId": spreadsheetID,
				"range":         rangeSpec,
				"targetSheet":   targetSheet,
				"mode":          c.Mode,
				"rowsMoved":     0,
			})
		}
		u.Out().Printf("No rows in source; nothing to %s", c.Mode)
		return nil
	}

	colIdx := c.Column
	if colIdx < 0 {
		return usagef("column must be >= 0, got %d", colIdx)
	}

	var matched [][]interface{}
	var matchedIndices []int // 0-based data row indices
	for i, row := range resp.Values {
		if colIdx >= len(row) {
			continue
		}
		if filterMatch(row[colIdx], op, c.Value) {
			matched = append(matched, row)
			matchedIndices = append(matchedIndices, i)
		}
	}

	if len(matched) == 0 {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
				"spreadsheetId": spreadsheetID,
				"range":         rangeSpec,
				"targetSheet":   targetSheet,
				"mode":          c.Mode,
				"rowsMoved":     0,
			})
		}
		u.Out().Printf("No rows matched; nothing to %s", c.Mode)
		return nil
	}

	destCell := strings.TrimSpace(c.DestinationCell)
	if destCell == "" {
		destCell = "A1"
	}
	destRange := targetSheet + "!" + destCell

	vr := &sheets.ValueRange{Values: matched}
	_, err = svc.Spreadsheets.Values.Update(spreadsheetID, destRange, vr).
		ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		return err
	}

	if c.Mode == "move" && len(matchedIndices) > 0 {
		// Delete source rows bottom-up so indices stay valid (sheet row indices are 0-based in API)
		startRow0 := parsed.StartRow - 1
		sheetRowIndices := make([]int, len(matchedIndices))
		for i, di := range matchedIndices {
			sheetRowIndices[i] = startRow0 + di
		}
		sort.Sort(sort.Reverse(sort.IntSlice(sheetRowIndices)))

		var requests []*sheets.Request
		for _, rowIndex := range sheetRowIndices {
			requests = append(requests, &sheets.Request{
				DeleteDimension: &sheets.DeleteDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sourceSheetID,
						Dimension:  "ROWS",
						StartIndex: int64(rowIndex),
						EndIndex:   int64(rowIndex + 1),
					},
				},
			})
		}
		_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{Requests: requests}).Context(ctx).Do()
		if err != nil {
			return err
		}
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
			"spreadsheetId": spreadsheetID,
			"range":         rangeSpec,
			"targetSheet":   targetSheet,
			"destination":   destRange,
			"mode":          c.Mode,
			"rowsMoved":     len(matched),
		})
	}
	verb := "Copied"
	if c.Mode == "move" {
		verb = "Moved"
	}
	u.Out().Printf("%s %d rows to %s", verb, len(matched), destRange)
	return nil
}
