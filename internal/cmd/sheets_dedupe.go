package cmd

import (
	"context"
	"os"
	"strconv"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SheetsDedupeCmd removes duplicate rows by key columns, keeping the first occurrence.
type SheetsDedupeCmd struct {
	SpreadsheetID string `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range         string `arg:"" name:"range" help:"Range to dedupe (e.g. Sheet1!A2:J200); must include sheet name"`
	KeyColumns    string `name:"key-columns" help:"Comma-separated 0-based column indices for duplicate key (default: all columns)" default:""`
	Keep          string `name:"keep" help:"Which duplicate to keep: first (default)" enum:"first" default:"first"`
}

// Run runs the sheets dedupe command using the API DeleteDuplicates request.
func (c *SheetsDedupeCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	spreadsheetID := normalizeGoogleID(strings.TrimSpace(c.SpreadsheetID))
	rangeSpec := cleanRange(c.Range)
	if spreadsheetID == "" {
		return usage("empty spreadsheetId")
	}
	if strings.TrimSpace(rangeSpec) == "" {
		return usage("empty range")
	}
	if c.Keep != "" && c.Keep != "first" {
		return usage("keep must be first (only first is supported)")
	}

	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return err
	}

	parsed, err := parseSheetRange(rangeSpec, "dedupe")
	if err != nil {
		return err
	}

	sheetIDs, err := fetchSheetIDMap(ctx, svc, spreadsheetID)
	if err != nil {
		return err
	}

	grid, err := gridRangeFromMap(parsed, sheetIDs, "dedupe")
	if err != nil {
		return err
	}

	var comparisonColumns []*sheets.DimensionRange
	if strings.TrimSpace(c.KeyColumns) != "" {
		parts := strings.Split(c.KeyColumns, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			col, parseErr := strconv.ParseInt(p, 10, 64)
			if parseErr != nil || col < 0 {
				return usagef("invalid key-column index %q", p)
			}
			comparisonColumns = append(comparisonColumns, &sheets.DimensionRange{
				SheetId:    grid.SheetId,
				Dimension:  "COLUMNS",
				StartIndex: col,
				EndIndex:   col + 1,
			})
		}
		if len(comparisonColumns) == 0 {
			return usage("key-columns must contain at least one valid 0-based index")
		}
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteDuplicates: &sheets.DeleteDuplicatesRequest{
					Range:             grid,
					ComparisonColumns: comparisonColumns,
				},
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		out := map[string]any{
			"spreadsheetId": spreadsheetID,
			"range":         rangeSpec,
			"keep":          c.Keep,
		}
		if len(comparisonColumns) > 0 {
			keys := make([]int, 0, len(comparisonColumns))
			for _, dr := range comparisonColumns {
				keys = append(keys, int(dr.StartIndex))
			}
			out["keyColumns"] = keys
		}
		return outfmt.WriteJSON(ctx, os.Stdout, out)
	}

	u.Out().Printf("Deduplicated %s (keep %s)", rangeSpec, c.Keep)
	return nil
}
