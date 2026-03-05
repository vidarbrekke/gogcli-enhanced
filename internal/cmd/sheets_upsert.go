package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"google.golang.org/api/sheets/v4"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

// SheetsUpsertCmd upserts rows by key: updates matching rows, appends the rest.
type SheetsUpsertCmd struct {
	SpreadsheetID string `arg:"" name:"spreadsheetId" help:"Spreadsheet ID"`
	Range         string `arg:"" name:"range" help:"Range containing existing rows (e.g. Sheet1!A2:J200); must include sheet name"`
	KeyColumns    string `name:"key-columns" help:"Comma-separated 0-based column indices for row key"`
	RowsJSON      string `name:"rows-json" help:"Rows to upsert as JSON 2D array (e.g. [[\"a\",1],[\"b\",2]])"`
}

// Run runs the sheets upsert command.
func (c *SheetsUpsertCmd) Run(ctx context.Context, flags *RootFlags) error {
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
	if rangeSpec == "" {
		return usage("empty range")
	}
	if strings.TrimSpace(c.RowsJSON) == "" {
		return usage("rows-json is required")
	}

	var inputRows [][]interface{}
	if unmarshalErr := json.Unmarshal([]byte(c.RowsJSON), &inputRows); unmarshalErr != nil {
		return usagef("invalid rows-json: %v", unmarshalErr)
	}
	if len(inputRows) == 0 {
		if outfmt.IsJSON(ctx) {
			return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
				"spreadsheetId": spreadsheetID,
				"range":         rangeSpec,
				"updated":       0,
				"appended":      0,
			})
		}
		u.Out().Printf("No rows to upsert")
		return nil
	}

	keyCols, err := parseKeyColumns(c.KeyColumns)
	if err != nil {
		return err
	}

	svc, err := newSheetsService(ctx, account)
	if err != nil {
		return err
	}

	parsed, err := parseSheetRange(rangeSpec, "upsert")
	if err != nil {
		return err
	}

	resp, err := svc.Spreadsheets.Values.Get(spreadsheetID, rangeSpec).Context(ctx).Do()
	if err != nil {
		return err
	}

	existing := resp.Values
	keyToIndex := make(map[string]int) // key -> 0-based data row index
	for i, row := range existing {
		k := rowKey(row, keyCols)
		if k != "" {
			keyToIndex[k] = i
		}
	}

	var updates []struct {
		index int
		row   []interface{}
	}
	var appends [][]interface{}
	for _, row := range inputRows {
		k := rowKey(row, keyCols)
		if k == "" {
			appends = append(appends, row)
			continue
		}
		if idx, ok := keyToIndex[k]; ok {
			updates = append(updates, struct {
				index int
				row   []interface{}
			}{idx, row})
		} else {
			appends = append(appends, row)
		}
	}

	updated := 0
	if len(updates) > 0 {
		// Build dense grid: copy existing, overwrite updated indices
		dense := make([][]interface{}, len(existing))
		copy(dense, existing)
		for _, u := range updates {
			if u.index < len(dense) {
				dense[u.index] = u.row
				updated++
			}
		}
		updateRange := a1RangeString(parsed.SheetName, parsed.StartRow, parsed.StartRow+len(dense)-1, parsed.StartCol, parsed.EndCol)
		vr := &sheets.ValueRange{Values: dense}
		if _, err := svc.Spreadsheets.Values.Update(spreadsheetID, updateRange, vr).
			ValueInputOption("USER_ENTERED").Context(ctx).Do(); err != nil {
			return err
		}
	}

	appended := 0
	if len(appends) > 0 {
		appendStartRow := parsed.StartRow + len(existing)
		appendRange := a1RangeString(parsed.SheetName, appendStartRow, appendStartRow, parsed.StartCol, parsed.EndCol)
		vr := &sheets.ValueRange{Values: appends}
		if _, err := svc.Spreadsheets.Values.Append(spreadsheetID, appendRange, vr).
			ValueInputOption("USER_ENTERED").Context(ctx).Do(); err != nil {
			return err
		}
		appended = len(appends)
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"spreadsheetId": spreadsheetID,
			"range":         rangeSpec,
			"updated":       updated,
			"appended":      appended,
		})
	}
	u.Out().Printf("Upserted: %d updated, %d appended", updated, appended)
	return nil
}

func parseKeyColumns(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, usage("key-columns is required")
	}
	var out []int
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil, usagef("invalid key-column index %q", p)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, usage("key-columns must contain at least one 0-based index")
	}
	return out, nil
}

func rowKey(row []interface{}, keyCols []int) string {
	var parts []string
	for _, col := range keyCols {
		if col >= len(row) {
			return ""
		}
		parts = append(parts, fmt.Sprintf("%v", row[col]))
	}
	return strings.Join(parts, "\x00")
}
