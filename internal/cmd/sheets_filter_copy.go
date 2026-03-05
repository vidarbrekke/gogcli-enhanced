package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SheetsFilterCopyCmd filters rows by a simple condition and writes matching rows to another sheet.
type SheetsFilterCopyCmd struct {
	SpreadsheetID   string `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range           string `arg:"" name:"range" help:"Source range (e.g. Sheet1!A2:J200); must include sheet name"`
	TargetSheet     string `arg:"" name:"targetSheet" help:"Destination sheet name (e.g. Filtered)"`
	Column          int    `name:"column" help:"Zero-based column index to filter on (0 = column A)" default:"0"`
	Op              string `name:"op" help:"Operator: eq, contains, gt, lt" enum:"eq,contains,gt,lt" default:"eq"`
	Value           string `name:"value" help:"Value to compare against"`
	DestinationCell string `name:"destination-cell" help:"Start cell on target sheet (default A1)" default:"A1"`
}

// Run runs the sheets filter-copy command.
func (c *SheetsFilterCopyCmd) Run(ctx context.Context, flags *RootFlags) error {
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
		return usage("op is required (eq, contains, gt, lt)")
	}
	switch op {
	case "eq", "contains", "gt", "lt":
	default:
		return usagef("invalid op %q; use eq, contains, gt, or lt", c.Op)
	}

	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return err
	}

	resp, err := svc.Spreadsheets.Values.Get(spreadsheetID, rangeSpec).Context(ctx).Do()
	if err != nil {
		return err
	}

	if len(resp.Values) == 0 {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
				"spreadsheetId": spreadsheetID,
				"range":         rangeSpec,
				"targetSheet":   targetSheet,
				"rowsCopied":    0,
			})
		}
		u.Out().Printf("No rows in source; nothing copied")
		return nil
	}

	colIdx := c.Column
	if colIdx < 0 {
		return usagef("column must be >= 0, got %d", colIdx)
	}

	var matched [][]interface{}
	for _, row := range resp.Values {
		if colIdx >= len(row) {
			continue
		}
		cellVal := row[colIdx]
		if filterMatch(cellVal, op, c.Value) {
			matched = append(matched, row)
		}
	}

	destCell := strings.TrimSpace(c.DestinationCell)
	if destCell == "" {
		destCell = "A1"
	}
	destRange := targetSheet + "!" + destCell

	if len(matched) == 0 {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
				"spreadsheetId": spreadsheetID,
				"range":         rangeSpec,
				"targetSheet":   targetSheet,
				"destination":   destRange,
				"rowsCopied":    0,
			})
		}
		u.Out().Printf("No rows matched; nothing copied")
		return nil
	}

	vr := &sheets.ValueRange{Values: matched}
	_, err = svc.Spreadsheets.Values.Update(spreadsheetID, destRange, vr).
		ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"spreadsheetId": spreadsheetID,
			"range":         rangeSpec,
			"targetSheet":   targetSheet,
			"destination":   destRange,
			"rowsCopied":    len(matched),
		})
	}
	u.Out().Printf("Copied %d rows to %s", len(matched), destRange)
	return nil
}

func filterMatch(cell interface{}, op, want string) bool {
	s := cellString(cell)
	switch op {
	case "eq":
		return s == want
	case "contains":
		return strings.Contains(s, want)
	case "gt":
		return compareCell(s, want) > 0
	case "lt":
		return compareCell(s, want) < 0
	default:
		return false
	}
}

func cellString(cell interface{}) string {
	if cell == nil {
		return ""
	}
	return fmt.Sprintf("%v", cell)
}

func compareCell(a, b string) int {
	fa, errA := strconv.ParseFloat(strings.TrimSpace(a), 64)
	fb, errB := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if errA == nil && errB == nil {
		if fa < fb {
			return -1
		}
		if fa > fb {
			return 1
		}
		return 0
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
