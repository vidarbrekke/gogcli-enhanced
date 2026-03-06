package cmd

import (
	"context"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SheetsSortCmd sorts a range by one or more columns via the Sheets API SortRangeRequest.
type SheetsSortCmd struct {
	SpreadsheetID string `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range         string `arg:"" name:"range" help:"Range to sort (e.g. Sheet1!A2:J200); must include sheet name"`
	ByColumn      int    `name:"by-column" help:"Zero-based column index to sort by (0 = column A)" default:"0"`
	Desc          bool   `name:"desc" help:"Sort descending"`
}

// Run runs the sheets sort command.
func (c *SheetsSortCmd) Run(ctx context.Context, flags *RootFlags) error {
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

	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return err
	}

	parsed, err := parseSheetRange(rangeSpec, "sort")
	if err != nil {
		return err
	}

	sheetIDs, err := fetchSheetIDMap(ctx, svc, spreadsheetID)
	if err != nil {
		return err
	}

	grid, err := gridRangeFromMap(parsed, sheetIDs, "sort")
	if err != nil {
		return err
	}

	sortOrder := "ASCENDING"
	if c.Desc {
		sortOrder = "DESCENDING"
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				SortRange: &sheets.SortRangeRequest{
					Range: grid,
					SortSpecs: []*sheets.SortSpec{
						{
							DimensionIndex: int64(c.ByColumn),
							SortOrder:      sortOrder,
						},
					},
				},
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
			"spreadsheetId": spreadsheetID,
			"range":         rangeSpec,
			"sortByColumn":  c.ByColumn,
			"sortOrder":     sortOrder,
		})
	}

	u.Out().Printf("Sorted %s by column %d (%s)", rangeSpec, c.ByColumn, strings.ToLower(sortOrder))
	return nil
}
