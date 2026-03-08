package google

import (
	"github.com/steipete/gogcli/internal/mcp/server"
)

func sheetsSpecs(p *provider) []server.ToolSpec {
	return []server.ToolSpec{
		{
			Name:        "sheets_planBatch",
			Description: "Validate and plan a Sheets batch update request without applying changes.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "request"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"request":       map[string]any{"type": "object"},
				},
			},
			Handler: p.sheetsPlanBatch,
		}, {
			Name:        "sheets_executeBatch",
			Description: "Execute a Sheets batch update request.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-heavy",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "request"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"request":       map[string]any{"type": "object"},
				},
			},
			Handler: p.sheetsExecuteBatch,
		}, {
			Name:        "sheets_valuesUpdate",
			Description: "Update values in a Sheets range.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range", "values"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"range":         map[string]any{"type": "string"},
					"values":        map[string]any{"type": "array"},
					"valueInput":    map[string]any{"type": "string"},
					"validateOnly":  map[string]any{"type": "boolean"},
				},
			},
			Handler: p.sheetsValuesUpdate,
		}, {
			Name:        "sheets_valuesAppend",
			Description: "Append values in a Sheets range.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range", "values"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"range":         map[string]any{"type": "string"},
					"values":        map[string]any{"type": "array"},
					"valueInput":    map[string]any{"type": "string"},
					"insert":        map[string]any{"type": "string"},
					"validateOnly":  map[string]any{"type": "boolean"},
				},
			},
			Handler: p.sheetsValuesAppend,
		}, {
			Name:        "sheets_links",
			Description: "Get hyperlinks from a Sheets range.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"range":         map[string]any{"type": "string"},
				},
			},
			Handler: p.sheetsLinks,
		}, {
			Name:        "sheets_valuesGet",
			Description: "Get cell values from a Sheets range (full spreadsheet data). Returns range and values (2D array).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range"},
				"properties": map[string]any{
					"spreadsheetId":     map[string]any{"type": "string"},
					"range":             map[string]any{"type": "string"},
					"majorDimension":    map[string]any{"type": "string", "description": "ROWS or COLUMNS"},
					"valueRenderOption": map[string]any{"type": "string", "description": "FORMATTED_VALUE, UNFORMATTED_VALUE, or FORMULA"},
				},
			},
			Handler: p.sheetsValuesGet,
		}, {
			Name:        "sheets_valuesRead",
			Description: "Alias of sheets_valuesGet for spreadsheet values.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range"},
				"properties": map[string]any{
					"spreadsheetId":     map[string]any{"type": "string"},
					"range":             map[string]any{"type": "string"},
					"majorDimension":    map[string]any{"type": "string", "description": "ROWS or COLUMNS"},
					"valueRenderOption": map[string]any{"type": "string", "description": "FORMATTED_VALUE, UNFORMATTED_VALUE, or FORMULA"},
				},
			},
			Handler: p.sheetsValuesGet,
		}, {
			Name:        "sheets_sortRange",
			Description: "Sort a Sheets range by column (e.g. sort by Due_Date). Uses zero-based column index (0 = column A).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"range":         map[string]any{"type": "string", "description": "A1 range including sheet name (e.g. Sheet1!A2:J200)"},
					"sortByColumn":  map[string]any{"type": "integer", "description": "Zero-based column index to sort by (0 = A)"},
					"desc":          map[string]any{"type": "boolean", "description": "Sort descending"},
				},
			},
			Handler: p.sheetsSortRange,
		}, {
			Name:        "sheets_dedupeRows",
			Description: "Remove duplicate rows in a Sheets range by key columns; keeps first occurrence.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"range":         map[string]any{"type": "string", "description": "A1 range including sheet name (e.g. Sheet1!A2:J200)"},
					"keyColumns":    map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Zero-based column indices for duplicate key; omit to use all columns"},
					"keep":          map[string]any{"type": "string", "description": "Which duplicate to keep: first (default)"},
				},
			},
			Handler: p.sheetsDedupeRows,
		}, {
			Name:        "sheets_filterCopyRows",
			Description: "Filter rows in a Sheets range by condition (column op value) and copy matching rows to another sheet.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range", "targetSheet", "column", "op", "value"},
				"properties": map[string]any{
					"spreadsheetId":   map[string]any{"type": "string"},
					"range":           map[string]any{"type": "string", "description": "Source A1 range including sheet name (e.g. Sheet1!A2:J200)"},
					"targetSheet":     map[string]any{"type": "string", "description": "Destination sheet name"},
					"column":          map[string]any{"type": "integer", "description": "Zero-based column index to filter on"},
					"op":              map[string]any{"type": "string", "description": "Operator: eq, contains, gt, lt"},
					"value":           map[string]any{"type": "string", "description": "Value to compare against"},
					"destinationCell": map[string]any{"type": "string", "description": "Start cell on target sheet (default A1)"},
				},
			},
			Handler: p.sheetsFilterCopyRows,
		}, {
			Name:        "sheets_upsertRows",
			Description: "Upsert rows by key columns: update matching rows in range, append rows with new keys.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range", "keyColumns", "rows"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"range":         map[string]any{"type": "string", "description": "A1 range containing existing rows (e.g. Sheet1!A2:J200)"},
					"keyColumns":    map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Zero-based column indices for row key"},
					"rows":          map[string]any{"type": "array", "description": "Rows to upsert (2D array)"},
				},
			},
			Handler: p.sheetsUpsertRows,
		}, {
			Name:        "sheets_moveRows",
			Description: "Filter rows by condition (column op value) and copy or move matching rows to another sheet.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range", "targetSheet", "column", "op", "value"},
				"properties": map[string]any{
					"spreadsheetId":   map[string]any{"type": "string"},
					"range":           map[string]any{"type": "string", "description": "Source A1 range including sheet name"},
					"targetSheet":     map[string]any{"type": "string", "description": "Destination sheet name"},
					"column":          map[string]any{"type": "integer", "description": "Zero-based column index to filter on"},
					"op":              map[string]any{"type": "string", "description": "Operator: eq, contains, gt, lt"},
					"value":           map[string]any{"type": "string", "description": "Value to compare against"},
					"mode":            map[string]any{"type": "string", "description": "copy (default) or move"},
					"destinationCell": map[string]any{"type": "string", "description": "Start cell on target sheet (default A1)"},
				},
			},
			Handler: p.sheetsMoveRows,
		}, {
			Name:        "sheets_applyFormula",
			Description: "Apply a formula to a column range; formula template may contain {row} for 1-based row number (fill down).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range", "formula"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"range":         map[string]any{"type": "string", "description": "Target column range (e.g. Sheet1!C2:C10)"},
					"formula":       map[string]any{"type": "string", "description": "Formula template with {row} placeholder (e.g. =A{row}+B{row})"},
				},
			},
			Handler: p.sheetsApplyFormula,
		}, {
			Name:        "sheets_summarize",
			Description: "Create a summary tab: group rows by columns and aggregate a metric column (count or sum).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range", "groupBy", "aggregate"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"range":         map[string]any{"type": "string", "description": "Source A1 range (e.g. Sheet1!A2:D200)"},
					"targetSheet":   map[string]any{"type": "string", "description": "Summary sheet name (default Summary)"},
					"groupBy":       map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Zero-based column indices for grouping"},
					"metricColumn":  map[string]any{"type": "integer", "description": "Zero-based column index for sum/count"},
					"aggregate":     map[string]any{"type": "string", "description": "count or sum"},
				},
			},
			Handler: p.sheetsSummarize,
		}, {
			Name:        "sheets_clear",
			Description: "Clear values in a Sheets range (no format/data validation removal).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId", "range"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
					"range":         map[string]any{"type": "string", "description": "A1 range (e.g. Sheet1!A1:B10)"},
					"dryRun":        map[string]any{"type": "boolean"},
				},
			},
			Handler: p.sheetsClear,
		}, {
			Name:        "sheets_metadata",
			Description: "Get spreadsheet metadata (title, locale, timeZone, sheet list with id/title/rowCount/columnCount).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"spreadsheetId"},
				"properties": map[string]any{
					"spreadsheetId": map[string]any{"type": "string"},
				},
			},
			Handler: p.sheetsMetadata,
		},
	}
}
