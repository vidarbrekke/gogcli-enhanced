package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SheetsSummarizeCmd creates a summary tab by grouping rows and aggregating a metric column.
type SheetsSummarizeCmd struct {
	SpreadsheetID string `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range         string `arg:"" name:"range" help:"Source range (e.g. Sheet1!A2:D200); must include sheet name"`
	TargetSheet   string `name:"target-sheet" help:"Summary sheet name (default: Summary)" default:"Summary"`
	GroupBy       string `name:"group-by" help:"Comma-separated 0-based column indices for grouping"`
	MetricColumn  int    `name:"metric-column" help:"Zero-based column index for aggregate (sum/count)" default:"0"`
	Aggregate     string `name:"aggregate" help:"Aggregate: count or sum" enum:"count,sum" default:"count"`
}

// Run runs the sheets summarize command.
func (c *SheetsSummarizeCmd) Run(ctx context.Context, flags *RootFlags) error {
	u := ui.FromContext(ctx)
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}

	spreadsheetID := normalizeGoogleID(strings.TrimSpace(c.SpreadsheetID))
	rangeSpec := cleanRange(c.Range)
	targetSheet := strings.TrimSpace(c.TargetSheet)
	if targetSheet == "" {
		targetSheet = "Summary"
	}
	if spreadsheetID == "" {
		return usage("empty spreadsheetId")
	}
	if rangeSpec == "" {
		return usage("empty range")
	}
	agg := strings.TrimSpace(strings.ToLower(c.Aggregate))
	if agg != "count" && agg != "sum" {
		return usage("aggregate must be count or sum")
	}

	groupCols, err := parseKeyColumns(c.GroupBy)
	if err != nil {
		return err
	}
	metricCol := c.MetricColumn
	if metricCol < 0 {
		return usagef("metric-column must be >= 0, got %d", metricCol)
	}

	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return err
	}

	sheetIDs, err := fetchSheetIDMap(ctx, svc, spreadsheetID)
	if err != nil {
		return err
	}
	if _, ok := sheetIDs[targetSheet]; !ok {
		req := &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{
				{
					AddSheet: &sheets.AddSheetRequest{
						Properties: &sheets.SheetProperties{Title: targetSheet},
					},
				},
			},
		}
		if _, updateErr := svc.Spreadsheets.BatchUpdate(spreadsheetID, req).Context(ctx).Do(); updateErr != nil {
			return updateErr
		}
		_, _ = fetchSheetIDMap(ctx, svc, spreadsheetID)
	}

	resp, err := svc.Spreadsheets.Values.Get(spreadsheetID, rangeSpec).Context(ctx).Do()
	if err != nil {
		return err
	}

	if len(resp.Values) == 0 {
		emptyRange := targetSheet + "!A1"
		vr := &sheets.ValueRange{Values: [][]interface{}{{"(no data)"}}}
		if _, err := svc.Spreadsheets.Values.Update(spreadsheetID, emptyRange, vr).ValueInputOption("USER_ENTERED").Context(ctx).Do(); err != nil {
			return err
		}
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
				"spreadsheetId": spreadsheetID,
				"range":         rangeSpec,
				"targetSheet":   targetSheet,
				"aggregate":     agg,
				"rowCount":      0,
			})
		}
		u.Out().Printf("No source data; wrote placeholder to %s", targetSheet)
		return nil
	}

	// group key -> aggregated value (count or sum)
	type groupVal struct {
		count int
		sum   float64
	}
	groups := make(map[string]*groupVal)
	for _, row := range resp.Values {
		key := rowKey(row, groupCols)
		if key == "" {
			continue
		}
		if groups[key] == nil {
			groups[key] = &groupVal{}
		}
		groups[key].count++
		if agg == "sum" && metricCol < len(row) {
			if n, err := toFloat(row[metricCol]); err == nil {
				groups[key].sum += n
			}
		}
	}

	numCols := len(groupCols) + 1
	var outRows [][]interface{}
	for key, gv := range groups {
		keyParts := strings.Split(key, "\x00")
		row := make([]interface{}, 0, numCols)
		for _, p := range keyParts {
			row = append(row, p)
		}
		if agg == "count" {
			row = append(row, gv.count)
		} else {
			row = append(row, gv.sum)
		}
		outRows = append(outRows, row)
	}
	if len(outRows) == 0 {
		outRows = [][]interface{}{{"(no groups)"}}
		numCols = 1
	}

	endCol := indexToColLetters(numCols)
	destRange := targetSheet + "!A1:" + endCol + strconv.Itoa(len(outRows))
	vr := &sheets.ValueRange{Values: outRows}
	if _, err := svc.Spreadsheets.Values.Update(spreadsheetID, destRange, vr).
		ValueInputOption("USER_ENTERED").Context(ctx).Do(); err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, stdoutWriter(ctx), map[string]any{
			"spreadsheetId": spreadsheetID,
			"range":         rangeSpec,
			"targetSheet":   targetSheet,
			"aggregate":     agg,
			"rowCount":      len(outRows),
		})
	}
	u.Out().Printf("Wrote %d summary rows to %s", len(outRows), targetSheet)
	return nil
}

func toFloat(v interface{}) (float64, error) {
	if v == nil {
		return 0, nil
	}
	switch x := v.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(x), 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float", v)
	}
}
