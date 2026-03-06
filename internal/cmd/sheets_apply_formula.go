package cmd

import (
	"context"
	"strconv"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SheetsApplyFormulaCmd applies a formula to a column range (fill down).
// Formula template may contain {row} placeholder (1-based sheet row).
type SheetsApplyFormulaCmd struct {
	SpreadsheetID string `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range         string `arg:"" name:"range" help:"Target column range (e.g. Sheet1!C2:C10); formula is written to each row"`
	Formula       string `name:"formula" help:"Formula template; use {row} for 1-based row number (e.g. =A{row}+B{row})"`
}

// Run runs the sheets apply-formula command.
func (c *SheetsApplyFormulaCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	spreadsheetID := normalizeGoogleID(strings.TrimSpace(c.SpreadsheetID))
	rangeSpec := cleanRange(c.Range)
	formula := strings.TrimSpace(c.Formula)
	if spreadsheetID == "" {
		return usage("empty spreadsheetId")
	}
	if rangeSpec == "" {
		return usage("empty range")
	}
	if formula == "" {
		return usage("formula is required")
	}
	if !strings.Contains(formula, "{row}") {
		return usage("formula must contain {row} placeholder (e.g. =A{row}+B{row})")
	}

	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return err
	}

	parsed, err := parseSheetRange(rangeSpec, "apply-formula")
	if err != nil {
		return err
	}

	rowCount := parsed.EndRow - parsed.StartRow + 1
	if rowCount < 1 {
		return usage("range must span at least one row")
	}

	values := make([][]interface{}, rowCount)
	for i := 0; i < rowCount; i++ {
		rowNum := parsed.StartRow + i
		values[i] = []interface{}{strings.ReplaceAll(formula, "{row}", strconv.Itoa(rowNum))}
	}

	vr := &sheets.ValueRange{Values: values}
	_, err = svc.Spreadsheets.Values.Update(spreadsheetID, rangeSpec, vr).
		ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
			"spreadsheetId": spreadsheetID,
			"range":         rangeSpec,
			"rowsUpdated":   rowCount,
		})
	}
	u.Out().Printf("Applied formula to %d rows in %s", rowCount, rangeSpec)
	return nil
}
