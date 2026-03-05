//nolint:wsl_v5 // command argument composition is intentionally linear
package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/steipete/gogcli/internal/mcp/server"
)

type Executor func(args []string) (stdout string, stderr string, err error)

var (
	errMissingDocID              = errors.New("missing docId")
	errMissingSpreadsheetID      = errors.New("missing spreadsheetId")
	errMissingPresentationID     = errors.New("missing presentationId")
	errMissingRequest            = errors.New("missing request")
	errMissingPath               = errors.New("missing path")
	errMissingFileID             = errors.New("missing fileId")
	errMissingText               = errors.New("missing text")
	errMissingFind               = errors.New("missing find text")
	errMissingRange              = errors.New("missing range")
	errMissingIndex              = errors.New("invalid index")
	errMissingLocalPath          = errors.New("missing localPath")
	errMissingQuery              = errors.New("missing query")
	errMissingFileOrPermissionID = errors.New("missing fileId or permissionId")
	errToolCommandFailed         = errors.New("tool command failed")
	errToolStderr                = errors.New("tool stderr")
	errExecutorNotConfigured     = errors.New("executor not configured")
)

// mcpDrivePageSize is the default page size for drive list/search when called from MCP.
// Small so each response fits within gateway/exec tool result length limits; agent uses nextPageToken to get more.
const mcpDrivePageSize = 4

// mcpDriveMaxCap caps page size when agent uses maxResults/pageSize so response still fits gateway.
const mcpDriveMaxCap = 25

// driveListSearchNormalizeInput copies Drive API-style args into our names so agents can use either.
// pageToken → page; maxResults or pageSize → max (capped at mcpDriveMaxCap).
func driveListSearchNormalizeInput(input map[string]any) {
	page := strings.TrimSpace(asString(input["page"]))
	if page == "" || strings.EqualFold(page, "null") {
		if pt := strings.TrimSpace(asString(input["pageToken"])); pt != "" && !strings.EqualFold(pt, "null") {
			input["page"] = pt
		}
	}
	if _, hasMax := input["max"]; hasMax {
		return
	}
	if n, ok := asInt(input["maxResults"]); ok && n > 0 {
		if n > mcpDriveMaxCap {
			n = mcpDriveMaxCap
		}
		input["max"] = n
		return
	}
	if n, ok := asInt(input["pageSize"]); ok && n > 0 {
		if n > mcpDriveMaxCap {
			n = mcpDriveMaxCap
		}
		input["max"] = n
	}
}

func Register(s *server.Server, executor Executor) {
	p := &provider{exec: executor}
	toolSpecs := []server.ToolSpec{
		{
			Name:        "docs_planBatch",
			Description: "Validate and plan a Docs batch update request without applying changes.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "request"},
				"properties": map[string]any{
					"docId":             map[string]any{"type": "string"},
					"request":           map[string]any{"type": "object"},
					"opId":              map[string]any{"type": "string"},
					"requireRevisionId": map[string]any{"type": "string", "description": "Optional revision ID for optimistic concurrency"},
					"timeoutMs": map[string]any{
						"type": "integer",
					},
					"retries": map[string]any{
						"type": "integer",
					},
					"retryBackoffMs": map[string]any{
						"type": "integer",
					},
				},
			},
			Handler: p.docsPlanBatch,
		}, {
			Name:        "docs_executeBatch",
			Description: "Execute a Docs batch update request.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-heavy",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "request"},
				"properties": map[string]any{
					"docId":             map[string]any{"type": "string"},
					"request":           map[string]any{"type": "object"},
					"opId":              map[string]any{"type": "string"},
					"requireRevisionId": map[string]any{"type": "string", "description": "Optional revision ID for optimistic concurrency"},
					"timeoutMs": map[string]any{
						"type": "integer",
					},
					"retries": map[string]any{
						"type": "integer",
					},
					"retryBackoffMs": map[string]any{
						"type": "integer",
					},
				},
			},
			Handler: p.docsExecuteBatch,
		}, {
			Name:        "docs_sed",
			Description: "Run sed-like find-and-replace expressions on a Google Doc (sedmat).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId"},
				"properties": map[string]any{
					"docId":          map[string]any{"type": "string"},
					"expression":     map[string]any{"type": "string"},
					"expressions":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"fileContent":    map[string]any{"type": "string"},
					"dryRun":         map[string]any{"type": "boolean"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.docsSed,
		}, {
			Name:        "docs_smartEdit",
			Description: "Smart Docs edit: route to batch or sedmat by intent; returns engineSelected, riskLevel, decisionReason.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId"},
				"properties": map[string]any{
					"docId":          map[string]any{"type": "string"},
					"intentType":     map[string]any{"type": "string"},
					"request":        map[string]any{"type": "object"},
					"expressions":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"riskTolerance":  map[string]any{"type": "string"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.docsSmartEdit,
		}, {
			Name:        "docs_create",
			Description: "Create a new Google Doc. Optionally place it in a Drive folder by parentId (e.g. from drive.ensureFolder).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"title"},
				"properties": map[string]any{
					"title":          map[string]any{"type": "string"},
					"parentId":       map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.docsCreate,
		}, {
			Name:        "docs_createWithBody",
			Description: "Create a new Google Doc and optionally apply a batchUpdate in one call (faster than create + insertText). Use for creating a doc with initial content and/or formatting. parentId from drive.ensureFolder or drive.searchFiles.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"title"},
				"properties": map[string]any{
					"title":          map[string]any{"type": "string"},
					"parentId":       map[string]any{"type": "string"},
					"request":        map[string]any{"type": "object", "description": "Optional batchUpdate body with 'requests' array (e.g. insertText + updateParagraphStyle)."},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.docsCreateWithBody,
		}, {
			Name:        "docs_insertText",
			Description: "Insert text at a specific index in a Google Doc.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "text"},
				"properties": map[string]any{
					"docId":        map[string]any{"type": "string"},
					"text":         map[string]any{"type": "string"},
					"index":        map[string]any{"type": "integer"},
					"validateOnly": map[string]any{"type": "boolean"},
					"opId":         map[string]any{"type": "string"},
					"timeoutMs":    map[string]any{"type": "integer"},
					"retries":      map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{
						"type": "integer",
					},
				},
			},
			Handler: p.docsInsertText,
		}, {
			Name:        "docs_deleteRange",
			Description: "Delete a text range in a Google Doc.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "startIndex", "endIndex"},
				"properties": map[string]any{
					"docId":        map[string]any{"type": "string"},
					"startIndex":   map[string]any{"type": "integer"},
					"endIndex":     map[string]any{"type": "integer"},
					"validateOnly": map[string]any{"type": "boolean"},
					"force":        map[string]any{"type": "boolean"},
					"opId":         map[string]any{"type": "string"},
					"timeoutMs":    map[string]any{"type": "integer"},
					"retries":      map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{
						"type": "integer",
					},
				},
			},
			Handler: p.docsDeleteRange,
		}, {
			Name:        "docs_replaceAllText",
			Description: "Replace all text matches in a Google Doc.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "find"},
				"properties": map[string]any{
					"docId":          map[string]any{"type": "string"},
					"find":           map[string]any{"type": "string"},
					"replace":        map[string]any{"type": "string"},
					"matchCase":      map[string]any{"type": "boolean"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.docsReplaceAllText,
		}, {
			Name:        "docs_appendText",
			Description: "Append text to the end of a Google Doc.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "text"},
				"properties": map[string]any{
					"docId":          map[string]any{"type": "string"},
					"text":           map[string]any{"type": "string"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.docsAppendText,
		}, {
			Name:        "docs_insertTable",
			Description: "Insert a table in a Google Doc.",
			Tier:        "beta",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId"},
				"properties": map[string]any{
					"docId":          map[string]any{"type": "string"},
					"rows":           map[string]any{"type": "integer"},
					"cols":           map[string]any{"type": "integer"},
					"index":          map[string]any{"type": "integer"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.docsInsertTable,
		}, {
			Name:        "docs_insertImage",
			Description: "Insert an image into a Google Doc.",
			Tier:        "beta",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "uri"},
				"properties": map[string]any{
					"docId":          map[string]any{"type": "string"},
					"uri":            map[string]any{"type": "string"},
					"index":          map[string]any{"type": "integer"},
					"widthPt":        map[string]any{"type": "number"},
					"heightPt":       map[string]any{"type": "number"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.docsInsertImage,
		}, {
			Name:        "docs_locatorEdit",
			Description: "Apply anchor or marker-based edits in a Google Doc.",
			Tier:        "beta",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId"},
				"properties": map[string]any{
					"docId":          map[string]any{"type": "string"},
					"after":          map[string]any{"type": "string"},
					"insertText":     map[string]any{"type": "string"},
					"betweenStart":   map[string]any{"type": "string"},
					"betweenEnd":     map[string]any{"type": "string"},
					"replaceText":    map[string]any{"type": "string"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.docsLocatorEdit,
		}, {
			Name:        "docs_mergeData",
			Description: "Generate Google Docs from a template using JSON data (mail-merge). Use validateOnly to preview operations without creating files.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"templateId", "data"},
				"properties": map[string]any{
					"templateId":       map[string]any{"type": "string"},
					"data":             map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "Array of record objects for merge"},
					"filenameFormat":   map[string]any{"type": "string"},
					"outputFolderId":   map[string]any{"type": "string"},
					"includeTimestamp": map[string]any{"type": "boolean"},
					"validateOnly":     map[string]any{"type": "boolean"},
					"dryRun":           map[string]any{"type": "boolean"},
					"account":          map[string]any{"type": "string"},
				},
			},
			Handler: p.docsMergeData,
		}, {
			Name:        "docs_get",
			Description: "Get Google Doc metadata and revision (thin wrapper around gog docs info). Returns id, name, revisionId, webViewLink.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId"},
				"properties": map[string]any{
					"docId":   map[string]any{"type": "string"},
					"account": map[string]any{"type": "string"},
				},
			},
			Handler: p.docsGet,
		}, {
			Name:        "docs_cat",
			Description: "Extract plain text from a Google Doc (thin wrapper around gog docs cat). Use maxBytes to limit size; tab or allTabs for multi-tab docs.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId"},
				"properties": map[string]any{
					"docId":    map[string]any{"type": "string"},
					"maxBytes": map[string]any{"type": "integer", "description": "Max bytes to read (default 2000000)"},
					"tab":      map[string]any{"type": "string", "description": "Tab title or ID to read"},
					"allTabs":  map[string]any{"type": "boolean", "description": "Include all tabs with headers"},
					"account":  map[string]any{"type": "string"},
				},
			},
			Handler: p.docsCat,
		}, {
			Name:        "docs_listTabs",
			Description: "List all tabs in a Google Doc with id, title, index (thin wrapper around gog docs list-tabs).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId"},
				"properties": map[string]any{
					"docId":   map[string]any{"type": "string"},
					"account": map[string]any{"type": "string"},
				},
			},
			Handler: p.docsListTabs,
		}, {
			Name:        "docs_positionsEnd",
			Description: "Return the append index for a Google Doc (1-based position after last content). Use before docs.insertText or append to know where to insert.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId"},
				"properties": map[string]any{
					"docId":   map[string]any{"type": "string"},
					"account": map[string]any{"type": "string"},
				},
			},
			Handler: p.docsPositionsEnd,
		}, {
			Name:        "docs_positionsSearch",
			Description: "Search for text in a Google Doc and return startIndex/endIndex ranges (UTF-16). Use for precise insert/delete.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "text"},
				"properties": map[string]any{
					"docId":     map[string]any{"type": "string"},
					"text":      map[string]any{"type": "string"},
					"matchCase": map[string]any{"type": "boolean"},
					"account":   map[string]any{"type": "string"},
				},
			},
			Handler: p.docsPositionsSearch,
		}, {
			Name:        "docs_positionsHeadings",
			Description: "Return positions of HEADING_1..HEADING_6 paragraphs in a Google Doc (startIndex, endIndex, style, text).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId"},
				"properties": map[string]any{
					"docId":   map[string]any{"type": "string"},
					"account": map[string]any{"type": "string"},
				},
			},
			Handler: p.docsPositionsHeadings,
		}, {
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
					"opId":          map[string]any{"type": "string"},
					"timeoutMs":     map[string]any{"type": "integer"},
					"retries":       map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{
						"type": "integer",
					},
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
					"spreadsheetId":  map[string]any{"type": "string"},
					"request":        map[string]any{"type": "object"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
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
					"spreadsheetId":  map[string]any{"type": "string"},
					"range":          map[string]any{"type": "string"},
					"values":         map[string]any{"type": "array"},
					"valueInput":     map[string]any{"type": "string"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
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
					"spreadsheetId":  map[string]any{"type": "string"},
					"range":          map[string]any{"type": "string"},
					"values":         map[string]any{"type": "array"},
					"valueInput":     map[string]any{"type": "string"},
					"insert":         map[string]any{"type": "string"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
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
					"spreadsheetId":  map[string]any{"type": "string"},
					"range":          map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
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
					"account":           map[string]any{"type": "string"},
					"opId":              map[string]any{"type": "string"},
					"timeoutMs":         map[string]any{"type": "integer"},
					"retries":           map[string]any{"type": "integer"},
					"retryBackoffMs":    map[string]any{"type": "integer"},
				},
			},
			Handler: p.sheetsValuesGet,
		}, {
			Name:        "sheets_valuesRead",
			Description: "Alias for sheets_valuesGet. Get cell values from a Sheets range (full spreadsheet data). Returns range and values (2D array).",
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
					"account":           map[string]any{"type": "string"},
					"opId":              map[string]any{"type": "string"},
					"timeoutMs":         map[string]any{"type": "integer"},
					"retries":           map[string]any{"type": "integer"},
					"retryBackoffMs":    map[string]any{"type": "integer"},
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
					"spreadsheetId":  map[string]any{"type": "string"},
					"range":          map[string]any{"type": "string", "description": "A1 range including sheet name (e.g. Sheet1!A2:J200)"},
					"sortByColumn":   map[string]any{"type": "integer", "description": "Zero-based column index to sort by (0 = A)"},
					"desc":           map[string]any{"type": "boolean", "description": "Sort descending"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
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
					"spreadsheetId":  map[string]any{"type": "string"},
					"range":          map[string]any{"type": "string", "description": "A1 range including sheet name (e.g. Sheet1!A2:J200)"},
					"keyColumns":     map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Zero-based column indices for duplicate key; omit to use all columns"},
					"keep":           map[string]any{"type": "string", "description": "Which duplicate to keep: first (default)"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
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
					"account":         map[string]any{"type": "string"},
					"opId":            map[string]any{"type": "string"},
					"timeoutMs":       map[string]any{"type": "integer"},
					"retries":         map[string]any{"type": "integer"},
					"retryBackoffMs":  map[string]any{"type": "integer"},
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
					"spreadsheetId":  map[string]any{"type": "string"},
					"range":          map[string]any{"type": "string", "description": "A1 range containing existing rows (e.g. Sheet1!A2:J200)"},
					"keyColumns":     map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Zero-based column indices for row key"},
					"rows":           map[string]any{"type": "array", "description": "Rows to upsert (2D array)"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
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
					"account":         map[string]any{"type": "string"},
					"opId":            map[string]any{"type": "string"},
					"timeoutMs":       map[string]any{"type": "integer"},
					"retries":         map[string]any{"type": "integer"},
					"retryBackoffMs":  map[string]any{"type": "integer"},
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
					"spreadsheetId":  map[string]any{"type": "string"},
					"range":          map[string]any{"type": "string", "description": "Target column range (e.g. Sheet1!C2:C10)"},
					"formula":        map[string]any{"type": "string", "description": "Formula template with {row} placeholder (e.g. =A{row}+B{row})"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
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
					"spreadsheetId":  map[string]any{"type": "string"},
					"range":          map[string]any{"type": "string", "description": "Source A1 range (e.g. Sheet1!A2:D200)"},
					"targetSheet":    map[string]any{"type": "string", "description": "Summary sheet name (default Summary)"},
					"groupBy":        map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "description": "Zero-based column indices for grouping"},
					"metricColumn":   map[string]any{"type": "integer", "description": "Zero-based column index for sum/count"},
					"aggregate":      map[string]any{"type": "string", "description": "count or sum"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.sheetsSummarize,
		}, {
			Name:        "slides_planBatch",
			Description: "Validate and plan a Slides batch update request without applying changes.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"presentationId", "request"},
				"properties": map[string]any{
					"presentationId": map[string]any{"type": "string"},
					"request":        map[string]any{"type": "object"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.slidesPlanBatch,
		}, {
			Name:        "slides_executeBatch",
			Description: "Execute a Slides batch update request.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-heavy",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"presentationId", "request"},
				"properties": map[string]any{
					"presentationId": map[string]any{"type": "string"},
					"request":        map[string]any{"type": "object"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.slidesExecuteBatch,
		}, {
			Name:        "slides_replaceText",
			Description: "Find and replace text across slides.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"presentationId", "find"},
				"properties": map[string]any{
					"presentationId": map[string]any{"type": "string"},
					"find":           map[string]any{"type": "string"},
					"replace":        map[string]any{"type": "string"},
					"matchCase":      map[string]any{"type": "boolean"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.slidesReplaceText,
		}, {
			Name:        "slides_createSlide",
			Description: "Create a new slide in a presentation.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"presentationId"},
				"properties": map[string]any{
					"presentationId": map[string]any{"type": "string"},
					"layout":         map[string]any{"type": "string"},
					"index":          map[string]any{"type": "integer"},
					"validateOnly":   map[string]any{"type": "boolean"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.slidesCreateSlide,
		}, {
			Name:        "drive_ensureFolder",
			Description: "Ensure a folder path exists in Drive; create missing segments.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"path"},
				"properties": map[string]any{
					"path":           map[string]any{"type": "string"},
					"parentId":       map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveEnsureFolder,
		}, {
			Name:        "drive_untrash",
			Description: "Restore a trashed Drive file.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId"},
				"properties": map[string]any{
					"fileId":         map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveUntrash,
		}, {
			Name:        "drive_getPermission",
			Description: "Get one permission entry for a Drive file.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId", "permissionId"},
				"properties": map[string]any{
					"fileId":         map[string]any{"type": "string"},
					"permissionId":   map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveGetPermission,
		}, {
			Name:        "drive_listFiles",
			Description: "List files and folders in a Drive folder (default root). Returns one page (default 4 items) + nextPageToken. When the user asks for 'first N' items (e.g. first 15), always include \"max\" or \"maxResults\": N in the request. Use page or pageToken (from previous nextPageToken) for next page. Set global=true to list across all accessible files (cannot combine with parentId). For 'list all folders' use drive.searchFiles with query mimeType = 'application/vnd.google-apps.folder' and rawQuery true; then call again with page set to nextPageToken until nextPageToken is absent.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"parentId":   map[string]any{"type": "string"},
					"query":      map[string]any{"type": "string"},
					"max":        map[string]any{"type": "integer"},
					"page":       map[string]any{"type": "string"},
					"pageToken":  map[string]any{"type": "string"},
					"maxResults": map[string]any{"type": "integer"},
					"pageSize":   map[string]any{"type": "integer"},
					"global":     map[string]any{"type": "boolean"},
					"allDrives": map[string]any{
						"type": "boolean",
					},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveListFiles,
		}, {
			Name:        "drive_searchFiles",
			Description: "Search files and folders in Google Drive. Returns one page (default 4 items) + nextPageToken. When the user asks for 'first N' items (e.g. first 15), always include \"max\" or \"maxResults\": N in the request. Use page or pageToken (from previous nextPageToken) for next page. For 'list all folders' use query mimeType = 'application/vnd.google-apps.folder' with rawQuery true; then call again with page set to nextPageToken until nextPageToken is absent.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"query"},
				"properties": map[string]any{
					"query":          map[string]any{"type": "string"},
					"rawQuery":       map[string]any{"type": "boolean"},
					"max":            map[string]any{"type": "integer"},
					"page":           map[string]any{"type": "string"},
					"pageToken":      map[string]any{"type": "string"},
					"maxResults":     map[string]any{"type": "integer"},
					"pageSize":       map[string]any{"type": "integer"},
					"allDrives":      map[string]any{"type": "boolean"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveSearchFiles,
		}, {
			Name:        "drive_getFile",
			Description: "Get Drive file metadata.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId"},
				"properties": map[string]any{
					"fileId":         map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveGetFile,
		}, {
			Name:        "drive_uploadFile",
			Description: "Upload a local file to Google Drive. Use for backups: localPath is the path on the server where gog runs (e.g. on Linode use a path like /var/backups/file.tar.gz). Optionally set parentId (folder ID), name (Drive filename), or replaceFileId to overwrite an existing file.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-heavy",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"localPath"},
				"properties": map[string]any{
					"localPath": map[string]any{"type": "string", "description": "Path to the file on the server where gog/MCP runs (e.g. /var/backups/backup.tar.gz)"},
					"name":      map[string]any{"type": "string", "description": "Drive filename (default: basename of localPath)"},
					"parentId":  map[string]any{"type": "string", "description": "Drive folder ID to upload into"},
					"replaceFileId": map[string]any{
						"type":        "string",
						"description": "Overwrite this existing Drive file ID (keeps sharing); cannot combine with parentId",
					},
					"mimeType":            map[string]any{"type": "string"},
					"convert":             map[string]any{"type": "boolean"},
					"convertTo":           map[string]any{"type": "string"},
					"keepRevisionForever": map[string]any{"type": "boolean", "description": "Keep this revision forever (e.g. for backup retention)"},
					"account":             map[string]any{"type": "string"},
					"opId":                map[string]any{"type": "string"},
					"timeoutMs":           map[string]any{"type": "integer"},
					"retries":             map[string]any{"type": "integer"},
					"retryBackoffMs":      map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveUploadFile,
		}, {
			Name:        "drive_downloadFile",
			Description: "Download Drive file by ID.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId"},
				"properties": map[string]any{
					"fileId":         map[string]any{"type": "string"},
					"out":            map[string]any{"type": "string"},
					"format":         map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveDownloadFile,
		}, {
			Name:        "drive_listPermissions",
			Description: "List permissions on a Drive file.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId"},
				"properties": map[string]any{
					"fileId":         map[string]any{"type": "string"},
					"max":            map[string]any{"type": "integer"},
					"page":           map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveListPermissions,
		}, {
			Name:        "drive_listComments",
			Description: "List comments on a Drive file.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId"},
				"properties": map[string]any{
					"fileId":         map[string]any{"type": "string"},
					"max":            map[string]any{"type": "integer"},
					"page":           map[string]any{"type": "string"},
					"all":            map[string]any{"type": "boolean"},
					"includeQuoted":  map[string]any{"type": "boolean"},
					"failEmpty":      map[string]any{"type": "boolean"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveListComments,
		}, {
			Name:        "drive_deleteFile",
			Description: "Delete or trash a Drive file.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId"},
				"properties": map[string]any{
					"fileId":         map[string]any{"type": "string"},
					"permanent":      map[string]any{"type": "boolean"},
					"force":          map[string]any{"type": "boolean"},
					"validateOnly":   map[string]any{"type": "boolean", "description": "If true, return planned action without executing"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveDeleteFile,
		}, {
			Name:        "drive_moveFile",
			Description: "Move a Drive file to a different folder (thin wrapper around gog drive move).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId", "parentId"},
				"properties": map[string]any{
					"fileId":   map[string]any{"type": "string"},
					"parentId": map[string]any{"type": "string", "description": "New parent folder ID"},
					"account":  map[string]any{"type": "string"},
				},
			},
			Handler: p.driveMoveFile,
		}, {
			Name:        "drive_renameFile",
			Description: "Rename a Drive file or folder (thin wrapper around gog drive rename).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId", "name"},
				"properties": map[string]any{
					"fileId":  map[string]any{"type": "string"},
					"name":    map[string]any{"type": "string", "description": "New file or folder name"},
					"account": map[string]any{"type": "string"},
				},
			},
			Handler: p.driveRenameFile,
		}, {
			Name:        "drive_shareFile",
			Description: "Share a Drive file (add permission). Use to: anyone (link), user (email), or domain. Role: reader or writer.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId", "to"},
				"properties": map[string]any{
					"fileId":       map[string]any{"type": "string"},
					"to":           map[string]any{"type": "string", "description": "anyone|user|domain"},
					"email":        map[string]any{"type": "string", "description": "Required when to=user"},
					"domain":       map[string]any{"type": "string", "description": "Required when to=domain"},
					"role":         map[string]any{"type": "string", "description": "reader|writer (default reader)"},
					"discoverable": map[string]any{"type": "boolean"},
					"account":      map[string]any{"type": "string"},
				},
			},
			Handler: p.driveShareFile,
		}, {
			Name:        "drive_unshare",
			Description: "Remove a permission from a Drive file (thin wrapper around gog drive unshare). Requires force=true to skip confirmation.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId", "permissionId"},
				"properties": map[string]any{
					"fileId":       map[string]any{"type": "string"},
					"permissionId": map[string]any{"type": "string"},
					"force":        map[string]any{"type": "boolean"},
					"validateOnly": map[string]any{"type": "boolean", "description": "If true, return planned action without executing"},
					"account":      map[string]any{"type": "string"},
				},
			},
			Handler: p.driveUnshare,
		}, {
			Name:        "drive_createComment",
			Description: "Create a comment on a Drive file (thin wrapper around gog drive comments create). Optional quoted text for Docs.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId", "content"},
				"properties": map[string]any{
					"fileId":  map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
					"quoted":  map[string]any{"type": "string", "description": "Anchor comment to this text (e.g. for Google Docs)"},
					"account": map[string]any{"type": "string"},
				},
			},
			Handler: p.driveCreateComment,
		}, {
			Name:        "drive_deleteComment",
			Description: "Delete a comment on a Drive file (thin wrapper around gog drive comments delete). Requires force=true to skip confirmation.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId", "commentId"},
				"properties": map[string]any{
					"fileId":       map[string]any{"type": "string"},
					"commentId":    map[string]any{"type": "string"},
					"force":        map[string]any{"type": "boolean"},
					"validateOnly": map[string]any{"type": "boolean", "description": "If true, return planned action without executing"},
					"account":      map[string]any{"type": "string"},
				},
			},
			Handler: p.driveDeleteComment,
		}, {
			Name:        "drive_copyFile",
			Description: "Copy a Drive file to a new name and optionally to a folder (thin wrapper around gog drive copy).",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"fileId", "name"},
				"properties": map[string]any{
					"fileId":   map[string]any{"type": "string"},
					"name":     map[string]any{"type": "string", "description": "New file name"},
					"parentId": map[string]any{"type": "string"},
					"account":  map[string]any{"type": "string"},
				},
			},
			Handler: p.driveCopyFile,
		}, {
			Name:        "drive_bulkExecute",
			Description: "Execute a bounded list of Drive operations (move, rename, share, delete). Use validateOnly to preview without executing. Max 50 operations per call.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"operations"},
				"properties": map[string]any{
					"operations":   map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "List of { op: move|rename|share|delete, fileId, ... }"},
					"validateOnly": map[string]any{"type": "boolean"},
					"account":      map[string]any{"type": "string"},
				},
			},
			Handler: p.driveBulkExecute,
		},
	}
	for _, spec := range toolSpecs {
		s.RegisterToolSpec(spec)
	}
}

type provider struct {
	exec Executor
}

func (p *provider) docsPlanBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "docs", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	reqForFile := injectRequireRevision(requests, asString(input["requireRevisionId"]))
	path, err := writeTempJSON(reqForFile)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "batch", docID, "--requests-file", path, "--validate-only")
	return p.runCLI(cleanArgs(args), "docs", "planBatch")
}

func (p *provider) docsExecuteBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "docs", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	reqForFile := injectRequireRevision(requests, asString(input["requireRevisionId"]))
	path, err := writeTempJSON(reqForFile)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "batch", docID, "--requests-file", path)
	return p.runCLI(cleanArgs(args), "docs", "executeBatch")
}

func (p *provider) docsSed(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "sed", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	var exprs []string
	if e := strings.TrimSpace(asString(input["expression"])); e != "" {
		exprs = append(exprs, e)
	}
	if arr, ok := input["expressions"].([]any); ok {
		for _, v := range arr {
			if s := strings.TrimSpace(asString(v)); s != "" {
				exprs = append(exprs, s)
			}
		}
	}
	if len(exprs) == 0 {
		return map[string]any{"service": "docs", "operation": "sed", "error_code": server.ErrorCodeInvalidArgument, "message": "missing expression or expressions"}, errors.New("missing expression or expressions")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "sed", docID)
	if asBool(input["dryRun"]) {
		args = append(args, "-n")
	}
	if len(exprs) == 1 {
		args = append(args, exprs[0])
	} else {
		args = append(args, exprs[0])
		for _, e := range exprs[1:] {
			args = append(args, "-e", e)
		}
	}
	result, err := p.runCLI(cleanArgs(args), "docs", "sed")
	if err != nil {
		return result, err
	}
	if result != nil {
		result["engine"] = "sedmat"
	}
	return result, nil
}

func (p *provider) docsSmartEdit(ctx context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "smartEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	intentType := strings.TrimSpace(strings.ToLower(asString(input["intentType"])))
	if intentType == "" {
		intentType = "sed"
	}
	validateOnly := asBool(input["validateOnly"])

	var expressions []string
	if arr, ok := input["expressions"].([]any); ok {
		for _, v := range arr {
			if s := strings.TrimSpace(asString(v)); s != "" {
				expressions = append(expressions, s)
			}
		}
	}

	// Route by intent
	switch intentType {
	case "batch":
		if req, ok := input["request"].(map[string]any); ok {
			if validateOnly {
				return p.docsPlanBatch(ctx, map[string]any{"docId": docID, "request": req, "opId": input["opId"], "timeoutMs": input["timeoutMs"], "retries": input["retries"], "retryBackoffMs": input["retryBackoffMs"]})
			}
			return p.docsExecuteBatch(ctx, map[string]any{"docId": docID, "request": req, "opId": input["opId"], "timeoutMs": input["timeoutMs"], "retries": input["retries"], "retryBackoffMs": input["retryBackoffMs"]})
		}
		return map[string]any{"service": "docs", "operation": "smartEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "intentType batch requires request"}, errMissingRequest
	case "sed":
		if len(expressions) == 0 {
			return map[string]any{"service": "docs", "operation": "smartEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "intentType sed requires expressions"}, errors.New("missing expressions")
		}
		riskLevel, decisionReason := ClassifySedRiskFromExpressions(expressions)
		requiresConfirmation := riskLevel == RiskHigh
		// If validateOnly or high risk, return assessment without executing write
		if validateOnly || (riskLevel == RiskHigh) {
			out := map[string]any{
				"service":              "docs",
				"operation":            "smartEdit",
				"engineSelected":       "sed",
				"decisionReason":       decisionReason,
				"riskLevel":            string(riskLevel),
				"requiresConfirmation": requiresConfirmation,
				"docId":                docID,
			}
			if input["opId"] != nil {
				out["opId"] = asString(input["opId"])
			}
			return out, nil
		}
		// Medium/low: execute sed and wrap result with routing envelope
		sedInput := map[string]any{
			"docId": docID, "opId": input["opId"],
			"timeoutMs": input["timeoutMs"], "retries": input["retries"], "retryBackoffMs": input["retryBackoffMs"],
			"account": input["account"], "dryRun": false,
		}
		if len(expressions) == 1 {
			sedInput["expression"] = expressions[0]
		} else {
			sedInput["expressions"] = toAnySlice(expressions)
		}
		result, err := p.docsSed(ctx, sedInput)
		if err != nil {
			return result, err
		}
		if result != nil {
			result["engineSelected"] = "sed"
			result["decisionReason"] = decisionReason
			result["riskLevel"] = string(riskLevel)
			result["requiresConfirmation"] = false
		}
		return result, nil
	default:
		return map[string]any{"service": "docs", "operation": "smartEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "intentType must be batch or sed"}, errors.New("invalid intentType")
	}
}

func toAnySlice(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

func (p *provider) docsCreate(_ context.Context, input map[string]any) (map[string]any, error) {
	title := strings.TrimSpace(asString(input["title"]))
	if title == "" {
		return map[string]any{"service": "docs", "operation": "create", "error_code": server.ErrorCodeInvalidArgument, "message": "missing title"}, errors.New("missing title")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "create", title)
	if parentID := strings.TrimSpace(asString(input["parentId"])); parentID != "" {
		args = append(args, "--parent", parentID)
	}
	return p.runCLI(cleanArgs(args), "docs", "create")
}

func (p *provider) docsCreateWithBody(ctx context.Context, input map[string]any) (map[string]any, error) {
	createResult, err := p.docsCreate(ctx, input)
	if err != nil {
		return createResult, err
	}
	// Parse docId from create output: gog docs create --json returns {"file": {"id": "...", ...}}
	var docID string
	if fileObj, ok := createResult["file"].(map[string]any); ok {
		docID = asString(fileObj["id"])
	}
	if docID != "" {
		createResult["documentId"] = docID
	}
	if docID == "" {
		return createResult, nil
	}
	reqObj, ok := input["request"].(map[string]any)
	if !ok || reqObj == nil {
		return createResult, nil
	}
	// Apply batchUpdate in same tool call to save a round-trip
	batchInput := map[string]any{"docId": docID, "request": reqObj}
	for _, k := range []string{"account", "opId", "timeoutMs", "retries", "retryBackoffMs"} {
		if v, ok := input[k]; ok {
			batchInput[k] = v
		}
	}
	batchResult, batchErr := p.docsExecuteBatch(ctx, batchInput)
	if batchErr != nil {
		createResult["service"] = "docs"
		createResult["operation"] = "createWithBody"
		createResult["documentId"] = docID
		createResult["batchError"] = batchResult
		return createResult, batchErr
	}
	createResult["documentId"] = docID
	createResult["batch"] = batchResult
	return createResult, nil
}

func (p *provider) docsInsertText(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	text := strings.TrimSpace(asString(input["text"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "insertText", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if text == "" {
		return map[string]any{"service": "docs", "operation": "insertText", "error_code": "invalid_argument", "message": "missing text"}, errMissingText
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "insert", docID, text)
	if idx, ok := asInt(input["index"]); ok {
		if idx < 1 {
			return map[string]any{"service": "docs", "operation": "insertText", "error_code": "invalid_argument", "message": "index must be >= 1"}, errMissingIndex
		}
		args = append(args, "--index", strconv.FormatInt(idx, 10))
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "insertText")
}

func (p *provider) docsDeleteRange(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	start, okStart := asInt(input["startIndex"])
	end, okEnd := asInt(input["endIndex"])
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "deleteRange", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if !okStart || !okEnd {
		return map[string]any{"service": "docs", "operation": "deleteRange", "error_code": "invalid_argument", "message": "missing startIndex/endIndex"}, errMissingRange
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "delete", docID, strconv.FormatInt(start, 10), strconv.FormatInt(end, 10))
	if asBool(input["force"]) {
		args = append(args, "--force")
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "deleteRange")
}

func (p *provider) docsReplaceAllText(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	find := strings.TrimSpace(asString(input["find"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "replaceAllText", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if find == "" {
		return map[string]any{"service": "docs", "operation": "replaceAllText", "error_code": "invalid_argument", "message": "missing find"}, errMissingFind
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "replace", docID, find, asString(input["replace"]))
	if asBool(input["matchCase"]) {
		args = append(args, "--match-case")
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "replaceAllText")
}

func (p *provider) docsAppendText(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	text := strings.TrimSpace(asString(input["text"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "appendText", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if text == "" {
		return map[string]any{"service": "docs", "operation": "appendText", "error_code": "invalid_argument", "message": "missing text"}, errMissingText
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "append", docID, text)
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "appendText")
}

func (p *provider) docsInsertTable(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "insertTable", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "insert-table", docID)
	if rows, ok := asInt(input["rows"]); ok {
		args = append(args, "--rows", strconv.FormatInt(rows, 10))
	}
	if cols, ok := asInt(input["cols"]); ok {
		args = append(args, "--cols", strconv.FormatInt(cols, 10))
	}
	if idx, ok := asInt(input["index"]); ok {
		args = append(args, "--index", strconv.FormatInt(idx, 10))
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "insertTable")
}

func (p *provider) docsInsertImage(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	uri := strings.TrimSpace(asString(input["uri"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "insertImage", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	if uri == "" {
		return map[string]any{"service": "docs", "operation": "insertImage", "error_code": "invalid_argument", "message": "missing uri"}, errMissingPath
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "insert-image", docID, uri)
	if idx, ok := asInt(input["index"]); ok {
		args = append(args, "--index", strconv.FormatInt(idx, 10))
	}
	if width, ok := asFloat(input["widthPt"]); ok {
		args = append(args, "--width-pt", trimFloat(width))
	}
	if height, ok := asFloat(input["heightPt"]); ok {
		args = append(args, "--height-pt", trimFloat(height))
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "insertImage")
}

func (p *provider) docsLocatorEdit(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "locatorEdit", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "locator", docID)
	if v := strings.TrimSpace(asString(input["after"])); v != "" {
		args = append(args, "--after", v)
	}
	if v := asString(input["insertText"]); v != "" {
		args = append(args, "--insert", v)
	}
	if v := strings.TrimSpace(asString(input["betweenStart"])); v != "" {
		args = append(args, "--between-start", v)
	}
	if v := strings.TrimSpace(asString(input["betweenEnd"])); v != "" {
		args = append(args, "--between-end", v)
	}
	if v := asString(input["replaceText"]); v != "" {
		args = append(args, "--replace", v)
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "docs", "locatorEdit")
}

func (p *provider) docsMergeData(_ context.Context, input map[string]any) (map[string]any, error) {
	templateID := strings.TrimSpace(asString(input["templateId"]))
	if templateID == "" {
		return map[string]any{"service": "docs", "operation": "mergeData", "error_code": server.ErrorCodeInvalidArgument, "message": "missing templateId"}, errMissingDocID
	}
	data, ok := input["data"].([]any)
	if !ok || len(data) == 0 {
		return map[string]any{"service": "docs", "operation": "mergeData", "error_code": server.ErrorCodeInvalidArgument, "message": "missing or empty data array"}, errMissingRequest
	}
	path, err := writeTempJSON(data)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "edit", "merge-data", templateID, "--data-file", path)
	if v := strings.TrimSpace(asString(input["filenameFormat"])); v != "" {
		args = append(args, "--filename-format", v)
	}
	if v := strings.TrimSpace(asString(input["outputFolderId"])); v != "" {
		args = append(args, "--output-folder-id", v)
	}
	if asBool(input["includeTimestamp"]) {
		args = append(args, "--include-timestamp")
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	if asBool(input["dryRun"]) {
		args = append(args, "--dry-run")
	}
	return p.runCLI(cleanArgs(args), "docs", "mergeData")
}

func (p *provider) docsGet(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "get", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "info", docID)
	return p.runCLI(cleanArgs(args), "docs", "get")
}

func (p *provider) docsCat(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "cat", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "cat", docID)
	if maxBytes, ok := asInt(input["maxBytes"]); ok && maxBytes > 0 {
		args = append(args, "--max-bytes", strconv.FormatInt(maxBytes, 10))
	}
	if tab := strings.TrimSpace(asString(input["tab"])); tab != "" {
		args = append(args, "--tab", tab)
	}
	if asBool(input["allTabs"]) {
		args = append(args, "--all-tabs")
	}
	return p.runCLI(cleanArgs(args), "docs", "cat")
}

func (p *provider) docsListTabs(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "listTabs", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "list-tabs", docID)
	return p.runCLI(cleanArgs(args), "docs", "listTabs")
}

func (p *provider) docsPositionsEnd(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "positionsEnd", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "positions", "end", docID)
	return p.runCLI(cleanArgs(args), "docs", "positionsEnd")
}

func (p *provider) docsPositionsSearch(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	text := strings.TrimSpace(asString(input["text"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "positionsSearch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	if text == "" {
		return map[string]any{"service": "docs", "operation": "positionsSearch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing text"}, errMissingText
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "positions", "search", docID, "--text", text)
	if asBool(input["matchCase"]) {
		args = append(args, "--match-case")
	}
	return p.runCLI(cleanArgs(args), "docs", "positionsSearch")
}

func (p *provider) docsPositionsHeadings(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "positionsHeadings", "error_code": server.ErrorCodeInvalidArgument, "message": "missing docId"}, errMissingDocID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "docs", "positions", "headings", docID)
	return p.runCLI(cleanArgs(args), "docs", "positionsHeadings")
}

func (p *provider) sheetsPlanBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "sheets", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "edit", "batch", spreadsheetID, "--requests-file", path, "--validate-only")
	return p.runCLI(cleanArgs(args), "sheets", "planBatch")
}

func (p *provider) sheetsExecuteBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "sheets", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "edit", "batch", spreadsheetID, "--requests-file", path)
	return p.runCLI(cleanArgs(args), "sheets", "executeBatch")
}

func (p *provider) sheetsValuesUpdate(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "valuesUpdate", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "valuesUpdate", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	values, ok := input["values"]
	if !ok {
		return map[string]any{"service": "sheets", "operation": "valuesUpdate", "error_code": server.ErrorCodeInvalidArgument, "message": "missing values"}, errMissingRequest
	}
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "edit", "values", spreadsheetID, rangeSpec, "--values-json", string(valuesJSON))
	if v := strings.TrimSpace(asString(input["valueInput"])); v != "" {
		args = append(args, "--input", v)
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "sheets", "valuesUpdate")
}

func (p *provider) sheetsValuesAppend(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "valuesAppend", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "valuesAppend", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	values, ok := input["values"]
	if !ok {
		return map[string]any{"service": "sheets", "operation": "valuesAppend", "error_code": server.ErrorCodeInvalidArgument, "message": "missing values"}, errMissingRequest
	}
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "sheets", "edit", "append", spreadsheetID, rangeSpec, "--values-json", string(valuesJSON))
	if v := strings.TrimSpace(asString(input["valueInput"])); v != "" {
		args = append(args, "--input", v)
	}
	if v := strings.TrimSpace(asString(input["insert"])); v != "" {
		args = append(args, "--insert", v)
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "sheets", "valuesAppend")
}

func (p *provider) sheetsLinks(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "links", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "links", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "links", spreadsheetID, rangeSpec)
	return p.runCLI(cleanArgs(args), "sheets", "links")
}

func (p *provider) sheetsValuesGet(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "valuesGet", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "valuesGet", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "get", spreadsheetID, rangeSpec)
	if v := strings.TrimSpace(asString(input["majorDimension"])); v != "" {
		args = append(args, "--dimension", v)
	}
	if v := strings.TrimSpace(asString(input["valueRenderOption"])); v != "" {
		args = append(args, "--render", v)
	}
	return p.runCLI(cleanArgs(args), "sheets", "valuesGet")
}

func (p *provider) sheetsSortRange(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "sortRange", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "sortRange", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	byColumn := int64(0)
	if n, ok := asInt(input["sortByColumn"]); ok {
		byColumn = n
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "sort", spreadsheetID, rangeSpec)
	if byColumn != 0 {
		args = append(args, "--by-column", strconv.FormatInt(byColumn, 10))
	}
	if asBool(input["desc"]) {
		args = append(args, "--desc")
	}
	return p.runCLI(cleanArgs(args), "sheets", "sortRange")
}

func (p *provider) sheetsDedupeRows(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "dedupeRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "dedupeRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "dedupe", spreadsheetID, rangeSpec)
	if keep := strings.TrimSpace(asString(input["keep"])); keep != "" {
		args = append(args, "--keep", keep)
	}
	if keyCols, ok := input["keyColumns"]; ok {
		switch v := keyCols.(type) {
		case []any:
			var parts []string
			for _, x := range v {
				if n, ok := asInt(x); ok {
					parts = append(parts, strconv.FormatInt(n, 10))
				}
			}
			if len(parts) > 0 {
				args = append(args, "--key-columns", strings.Join(parts, ","))
			}
		case []int:
			parts := make([]string, 0, len(v))
			for _, n := range v {
				parts = append(parts, strconv.Itoa(n))
			}
			if len(parts) > 0 {
				args = append(args, "--key-columns", strings.Join(parts, ","))
			}
		}
	}
	return p.runCLI(cleanArgs(args), "sheets", "dedupeRows")
}

func (p *provider) sheetsFilterCopyRows(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	targetSheet := strings.TrimSpace(asString(input["targetSheet"]))
	op := strings.TrimSpace(asString(input["op"]))
	value := asString(input["value"])
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "filterCopyRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "filterCopyRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	if targetSheet == "" {
		return map[string]any{"service": "sheets", "operation": "filterCopyRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing targetSheet"}, errors.New("missing targetSheet")
	}
	if op == "" {
		return map[string]any{"service": "sheets", "operation": "filterCopyRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing op"}, errors.New("missing op")
	}
	col, _ := asInt(input["column"])
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "filter-copy", spreadsheetID, rangeSpec, targetSheet, "--column", strconv.FormatInt(col, 10), "--op", op, "--value", value)
	if dest := strings.TrimSpace(asString(input["destinationCell"])); dest != "" {
		args = append(args, "--destination-cell", dest)
	}
	return p.runCLI(cleanArgs(args), "sheets", "filterCopyRows")
}

func (p *provider) sheetsUpsertRows(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	rows, ok := input["rows"].([]any)
	if !ok || len(rows) == 0 {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing or empty rows"}, errors.New("missing or empty rows")
	}
	keyCols, ok := input["keyColumns"].([]any)
	if !ok || len(keyCols) == 0 {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing or empty keyColumns"}, errors.New("missing or empty keyColumns")
	}
	var keyParts []string
	for _, x := range keyCols {
		n, _ := asInt(x)
		keyParts = append(keyParts, strconv.FormatInt(n, 10))
	}
	rowsJSON, err := json.Marshal(input["rows"])
	if err != nil {
		return map[string]any{"service": "sheets", "operation": "upsertRows", "error_code": server.ErrorCodeInvalidArgument, "message": "invalid rows"}, err
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "upsert", spreadsheetID, rangeSpec, "--key-columns", strings.Join(keyParts, ","), "--rows-json", string(rowsJSON))
	return p.runCLI(cleanArgs(args), "sheets", "upsertRows")
}

func (p *provider) sheetsMoveRows(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	targetSheet := strings.TrimSpace(asString(input["targetSheet"]))
	op := strings.TrimSpace(asString(input["op"]))
	value := asString(input["value"])
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "moveRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "moveRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	if targetSheet == "" {
		return map[string]any{"service": "sheets", "operation": "moveRows", "error_code": server.ErrorCodeInvalidArgument, "message": "missing targetSheet"}, errors.New("missing targetSheet")
	}
	if op == "" {
		op = "eq"
	}
	col, _ := asInt(input["column"])
	mode := strings.TrimSpace(asString(input["mode"]))
	if mode == "" {
		mode = "copy"
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "move-rows", spreadsheetID, rangeSpec, targetSheet, "--column", strconv.FormatInt(col, 10), "--op", op, "--value", value, "--mode", mode)
	if dest := strings.TrimSpace(asString(input["destinationCell"])); dest != "" {
		args = append(args, "--destination-cell", dest)
	}
	return p.runCLI(cleanArgs(args), "sheets", "moveRows")
}

func (p *provider) sheetsApplyFormula(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	formula := strings.TrimSpace(asString(input["formula"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "applyFormula", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "applyFormula", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	if formula == "" {
		return map[string]any{"service": "sheets", "operation": "applyFormula", "error_code": server.ErrorCodeInvalidArgument, "message": "missing formula"}, errors.New("missing formula")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "apply-formula", spreadsheetID, rangeSpec, "--formula", formula)
	return p.runCLI(cleanArgs(args), "sheets", "applyFormula")
}

func (p *provider) sheetsSummarize(_ context.Context, input map[string]any) (map[string]any, error) {
	spreadsheetID := strings.TrimSpace(asString(input["spreadsheetId"]))
	rangeSpec := strings.TrimSpace(asString(input["range"]))
	if spreadsheetID == "" {
		return map[string]any{"service": "sheets", "operation": "summarize", "error_code": server.ErrorCodeInvalidArgument, "message": "missing spreadsheetId"}, errMissingSpreadsheetID
	}
	if rangeSpec == "" {
		return map[string]any{"service": "sheets", "operation": "summarize", "error_code": server.ErrorCodeInvalidArgument, "message": "missing range"}, errMissingRange
	}
	groupBy, ok := input["groupBy"].([]any)
	if !ok || len(groupBy) == 0 {
		return map[string]any{"service": "sheets", "operation": "summarize", "error_code": server.ErrorCodeInvalidArgument, "message": "missing or empty groupBy"}, errors.New("missing or empty groupBy")
	}
	agg := strings.TrimSpace(asString(input["aggregate"]))
	if agg == "" {
		agg = "count"
	}
	var groupParts []string
	for _, x := range groupBy {
		n, _ := asInt(x)
		groupParts = append(groupParts, strconv.FormatInt(n, 10))
	}
	metricCol, _ := asInt(input["metricColumn"])
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "sheets", "summarize", spreadsheetID, rangeSpec, "--group-by", strings.Join(groupParts, ","), "--metric-column", strconv.FormatInt(metricCol, 10), "--aggregate", agg)
	if target := strings.TrimSpace(asString(input["targetSheet"])); target != "" {
		args = append(args, "--target-sheet", target)
	}
	return p.runCLI(cleanArgs(args), "sheets", "summarize")
}

func (p *provider) slidesPlanBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	presentationID := strings.TrimSpace(asString(input["presentationId"]))
	if presentationID == "" {
		return map[string]any{"service": "slides", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing presentationId"}, errMissingPresentationID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "slides", "operation": "planBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "slides", "edit", "batch", presentationID, "--requests-file", path, "--validate-only")
	return p.runCLI(cleanArgs(args), "slides", "planBatch")
}

func (p *provider) slidesExecuteBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	presentationID := strings.TrimSpace(asString(input["presentationId"]))
	if presentationID == "" {
		return map[string]any{"service": "slides", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing presentationId"}, errMissingPresentationID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "slides", "operation": "executeBatch", "error_code": server.ErrorCodeInvalidArgument, "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "slides", "edit", "batch", presentationID, "--requests-file", path)
	return p.runCLI(cleanArgs(args), "slides", "executeBatch")
}

func (p *provider) slidesReplaceText(_ context.Context, input map[string]any) (map[string]any, error) {
	presentationID := strings.TrimSpace(asString(input["presentationId"]))
	find := strings.TrimSpace(asString(input["find"]))
	if presentationID == "" {
		return map[string]any{"service": "slides", "operation": "replaceText", "error_code": server.ErrorCodeInvalidArgument, "message": "missing presentationId"}, errMissingPresentationID
	}
	if find == "" {
		return map[string]any{"service": "slides", "operation": "replaceText", "error_code": server.ErrorCodeInvalidArgument, "message": "missing find"}, errMissingFind
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "slides", "edit", "replace-text", presentationID, "--find", find)
	if v := asString(input["replace"]); v != "" {
		args = append(args, "--replace", v)
	}
	if asBool(input["matchCase"]) {
		args = append(args, "--match-case")
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "slides", "replaceText")
}

func (p *provider) slidesCreateSlide(_ context.Context, input map[string]any) (map[string]any, error) {
	presentationID := strings.TrimSpace(asString(input["presentationId"]))
	if presentationID == "" {
		return map[string]any{"service": "slides", "operation": "createSlide", "error_code": server.ErrorCodeInvalidArgument, "message": "missing presentationId"}, errMissingPresentationID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "slides", "edit", "create-slide", presentationID)
	if v := strings.TrimSpace(asString(input["layout"])); v != "" {
		args = append(args, "--layout", v)
	}
	if idx, ok := asInt(input["index"]); ok {
		args = append(args, "--index", strconv.FormatInt(idx, 10))
	}
	if asBool(input["validateOnly"]) {
		args = append(args, "--validate-only")
	}
	return p.runCLI(cleanArgs(args), "slides", "createSlide")
}

func (p *provider) driveEnsureFolder(_ context.Context, input map[string]any) (map[string]any, error) {
	path := strings.TrimSpace(asString(input["path"]))
	if path == "" {
		return map[string]any{"service": "drive", "operation": "ensureFolder", "error_code": "invalid_argument", "message": "missing path"}, errMissingPath
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "ensure-folder", path)
	if parent := strings.TrimSpace(asString(input["parentId"])); parent != "" {
		args = append(args, "--parent", parent)
	}
	return p.runCLI(cleanArgs(args), "drive", "ensureFolder")
}

func (p *provider) driveUntrash(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "untrash", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "untrash", fileID)
	return p.runCLI(cleanArgs(args), "drive", "untrash")
}

func (p *provider) driveGetPermission(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	permissionID := strings.TrimSpace(asString(input["permissionId"]))
	if fileID == "" || permissionID == "" {
		return map[string]any{"service": "drive", "operation": "getPermission", "error_code": "invalid_argument", "message": "missing fileId or permissionId"}, errMissingFileOrPermissionID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "permission", fileID, permissionID)
	return p.runCLI(cleanArgs(args), "drive", "getPermission")
}

func (p *provider) driveListFiles(ctx context.Context, input map[string]any) (map[string]any, error) {
	driveListSearchNormalizeInput(input)
	query := strings.TrimSpace(asString(input["query"]))
	global := asBool(input["global"])
	parentID := strings.TrimSpace(asString(input["parentId"]))
	if global && parentID != "" {
		return map[string]any{"service": "drive", "operation": "listFiles", "error_code": "invalid_argument", "message": "global cannot be combined with parentId"}, errors.New("global cannot be combined with parentId")
	}
	if parentID == "" {
		parentID = "root"
	}
	// Redirect to folders-only when: no query (agent often means "list folders" when using {}),
	// or query is trashed=false / folder mimeType — so response has only folders and fits gateway.
	redirectToFoldersOnly := false
	if !global && query == "" {
		redirectToFoldersOnly = true
	} else {
		qLower := strings.ToLower(query)
		redirectToFoldersOnly = strings.Contains(qLower, "application/vnd.google-apps.folder") ||
			(strings.Contains(qLower, "mimetype") && strings.Contains(qLower, "folder")) ||
			qLower == "trashed=false"
	}
	if redirectToFoldersOnly {
		searchInput := make(map[string]any)
		for k, v := range input {
			searchInput[k] = v
		}
		searchInput["query"] = "mimeType = 'application/vnd.google-apps.folder' and '" + parentID + "' in parents"
		searchInput["rawQuery"] = true
		return p.driveSearchFiles(ctx, searchInput)
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "ls")
	if global {
		args = append(args, "--global")
	}
	if parent := strings.TrimSpace(asString(input["parentId"])); parent != "" {
		args = append(args, "--parent", parent)
	}
	if query := strings.TrimSpace(asString(input["query"])); query != "" {
		args = append(args, "--query", query)
	}
	if max, ok := asInt(input["max"]); ok && max > 0 {
		args = append(args, "--max", strconv.FormatInt(max, 10))
	}
	pageVal := strings.TrimSpace(asString(input["page"]))
	if pageVal != "" && !strings.EqualFold(pageVal, "null") {
		args = append(args, "--page", pageVal)
	} else {
		// Paginated mode (no --all): one page of mcpDrivePageSize so response fits gateway/exec limit.
		// Agent gets nextPageToken and can call again with page=<token> to get more.
		if _, hasMax := input["max"]; !hasMax {
			args = append(args, "--max", strconv.Itoa(mcpDrivePageSize))
		}
		args = append(args, "--compact")
	}
	if _, ok := input["allDrives"]; ok {
		if asBool(input["allDrives"]) {
			args = append(args, "--all-drives")
		} else {
			args = append(args, "--no-all-drives")
		}
	}
	return p.runCLI(cleanArgs(args), "drive", "listFiles")
}

func (p *provider) driveSearchFiles(_ context.Context, input map[string]any) (map[string]any, error) {
	driveListSearchNormalizeInput(input)
	query := strings.TrimSpace(asString(input["query"]))
	if query == "" {
		return map[string]any{"service": "drive", "operation": "searchFiles", "error_code": "invalid_argument", "message": "missing query"}, errMissingQuery
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "search")
	if asBool(input["rawQuery"]) {
		args = append(args, "--raw-query")
	}
	if max, ok := asInt(input["max"]); ok && max > 0 {
		args = append(args, "--max", strconv.FormatInt(max, 10))
	}
	pageVal := strings.TrimSpace(asString(input["page"]))
	if pageVal != "" && !strings.EqualFold(pageVal, "null") {
		args = append(args, "--page", pageVal)
	} else {
		// Paginated mode (no --all): one page of mcpDrivePageSize so response fits gateway/exec limit.
		if _, hasMax := input["max"]; !hasMax {
			args = append(args, "--max", strconv.Itoa(mcpDrivePageSize))
		}
		args = append(args, "--compact")
	}
	// Default allDrives to true so search includes shared drives; only restrict when explicitly false
	if v, ok := input["allDrives"]; ok && !asBool(v) {
		args = append(args, "--no-all-drives")
	} else {
		args = append(args, "--all-drives")
	}
	args = append(args, query)
	return p.runCLI(cleanArgs(args), "drive", "searchFiles")
}

func (p *provider) driveGetFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "getFile", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "get", fileID)
	return p.runCLI(cleanArgs(args), "drive", "getFile")
}

func (p *provider) driveUploadFile(_ context.Context, input map[string]any) (map[string]any, error) {
	localPath := strings.TrimSpace(asString(input["localPath"]))
	if localPath == "" {
		return map[string]any{"service": "drive", "operation": "uploadFile", "error_code": "invalid_argument", "message": "missing localPath"}, errMissingLocalPath
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "upload", localPath)
	if v := strings.TrimSpace(asString(input["name"])); v != "" {
		args = append(args, "--name", v)
	}
	if v := strings.TrimSpace(asString(input["parentId"])); v != "" {
		args = append(args, "--parent", v)
	}
	if v := strings.TrimSpace(asString(input["replaceFileId"])); v != "" {
		args = append(args, "--replace", v)
	}
	if v := strings.TrimSpace(asString(input["mimeType"])); v != "" {
		args = append(args, "--mime-type", v)
	}
	if asBool(input["convert"]) {
		args = append(args, "--convert")
	}
	if v := strings.TrimSpace(asString(input["convertTo"])); v != "" {
		args = append(args, "--convert-to", v)
	}
	if asBool(input["keepRevisionForever"]) {
		args = append(args, "--keep-revision-forever")
	}
	return p.runCLI(cleanArgs(args), "drive", "uploadFile")
}

func (p *provider) driveDownloadFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "downloadFile", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "download", fileID)
	if out := strings.TrimSpace(asString(input["out"])); out != "" {
		args = append(args, "--out", out)
	}
	if format := strings.TrimSpace(asString(input["format"])); format != "" {
		args = append(args, "--format", format)
	}
	return p.runCLI(cleanArgs(args), "drive", "downloadFile")
}

func (p *provider) driveListPermissions(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "listPermissions", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "permissions", fileID)
	if max, ok := asInt(input["max"]); ok && max > 0 {
		args = append(args, "--max", strconv.FormatInt(max, 10))
	}
	if page := strings.TrimSpace(asString(input["page"])); page != "" {
		args = append(args, "--page", page)
	}
	return p.runCLI(cleanArgs(args), "drive", "listPermissions")
}

func (p *provider) driveListComments(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "listComments", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "comments", "list", fileID)
	if max, ok := asInt(input["max"]); ok && max > 0 {
		args = append(args, "--max", strconv.FormatInt(max, 10))
	}
	if page := strings.TrimSpace(asString(input["page"])); page != "" {
		args = append(args, "--page", page)
	}
	if asBool(input["all"]) {
		args = append(args, "--all")
	}
	if asBool(input["includeQuoted"]) {
		args = append(args, "--include-quoted")
	}
	if asBool(input["failEmpty"]) {
		args = append(args, "--fail-empty")
	}
	return p.runCLI(cleanArgs(args), "drive", "listComments")
}

func (p *provider) driveDeleteFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "deleteFile", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	if asBool(input["validateOnly"]) {
		planned := map[string]any{"fileId": fileID}
		if asBool(input["permanent"]) {
			planned["permanent"] = true
		}
		return map[string]any{
			"service":      "drive",
			"operation":    "deleteFile",
			"validateOnly": true,
			"planned":      planned,
		}, nil
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "delete", fileID)
	if asBool(input["permanent"]) {
		args = append(args, "--permanent")
	}
	if asBool(input["force"]) {
		args = append(args, "--force")
	}
	return p.runCLI(cleanArgs(args), "drive", "deleteFile")
}

func (p *provider) driveMoveFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	parentID := strings.TrimSpace(asString(input["parentId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "moveFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if parentID == "" {
		return map[string]any{"service": "drive", "operation": "moveFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing parentId"}, errMissingPath
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "move", fileID, "--parent", parentID)
	return p.runCLI(cleanArgs(args), "drive", "moveFile")
}

func (p *provider) driveRenameFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	name := strings.TrimSpace(asString(input["name"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "renameFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if name == "" {
		return map[string]any{"service": "drive", "operation": "renameFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing name"}, errors.New("missing name")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "rename", fileID, name)
	return p.runCLI(cleanArgs(args), "drive", "renameFile")
}

func (p *provider) driveShareFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	to := strings.TrimSpace(asString(input["to"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if to == "" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing to (anyone|user|domain)"}, errors.New("missing to")
	}
	to = strings.ToLower(to)
	if to != "anyone" && to != "user" && to != "domain" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "to must be anyone|user|domain"}, errors.New("invalid to")
	}
	if to == "user" && strings.TrimSpace(asString(input["email"])) == "" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "email required when to=user"}, errors.New("missing email")
	}
	if to == "domain" && strings.TrimSpace(asString(input["domain"])) == "" {
		return map[string]any{"service": "drive", "operation": "shareFile", "error_code": server.ErrorCodeInvalidArgument, "message": "domain required when to=domain"}, errors.New("missing domain")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "share", fileID, "--to", to)
	if email := strings.TrimSpace(asString(input["email"])); email != "" {
		args = append(args, "--email", email)
	}
	if domain := strings.TrimSpace(asString(input["domain"])); domain != "" {
		args = append(args, "--domain", domain)
	}
	if role := strings.TrimSpace(asString(input["role"])); role != "" {
		args = append(args, "--role", role)
	}
	if asBool(input["discoverable"]) {
		args = append(args, "--discoverable")
	}
	return p.runCLI(cleanArgs(args), "drive", "shareFile")
}

func (p *provider) driveUnshare(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	permID := strings.TrimSpace(asString(input["permissionId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "unshare", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if permID == "" {
		return map[string]any{"service": "drive", "operation": "unshare", "error_code": server.ErrorCodeInvalidArgument, "message": "missing permissionId"}, errMissingFileOrPermissionID
	}
	if asBool(input["validateOnly"]) {
		return map[string]any{
			"service":      "drive",
			"operation":    "unshare",
			"validateOnly": true,
			"planned":      map[string]any{"fileId": fileID, "permissionId": permID},
		}, nil
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "unshare", fileID, permID)
	if asBool(input["force"]) {
		args = append(args, "--force")
	}
	return p.runCLI(cleanArgs(args), "drive", "unshare")
}

func (p *provider) driveCreateComment(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	content := strings.TrimSpace(asString(input["content"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "createComment", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if content == "" {
		return map[string]any{"service": "drive", "operation": "createComment", "error_code": server.ErrorCodeInvalidArgument, "message": "missing content"}, errMissingText
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "comments", "create", fileID, content)
	if quoted := strings.TrimSpace(asString(input["quoted"])); quoted != "" {
		args = append(args, "--quoted", quoted)
	}
	return p.runCLI(cleanArgs(args), "drive", "createComment")
}

func (p *provider) driveDeleteComment(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	commentID := strings.TrimSpace(asString(input["commentId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "deleteComment", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if commentID == "" {
		return map[string]any{"service": "drive", "operation": "deleteComment", "error_code": server.ErrorCodeInvalidArgument, "message": "missing commentId"}, errMissingFileOrPermissionID
	}
	if asBool(input["validateOnly"]) {
		return map[string]any{
			"service":      "drive",
			"operation":    "deleteComment",
			"validateOnly": true,
			"planned":      map[string]any{"fileId": fileID, "commentId": commentID},
		}, nil
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "comments", "delete", fileID, commentID)
	if asBool(input["force"]) {
		args = append(args, "--force")
	}
	return p.runCLI(cleanArgs(args), "drive", "deleteComment")
}

func (p *provider) driveCopyFile(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	name := strings.TrimSpace(asString(input["name"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "copyFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing fileId"}, errMissingFileID
	}
	if name == "" {
		return map[string]any{"service": "drive", "operation": "copyFile", "error_code": server.ErrorCodeInvalidArgument, "message": "missing name"}, errors.New("missing name")
	}
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, "drive", "copy", fileID, name)
	if parentID := strings.TrimSpace(asString(input["parentId"])); parentID != "" {
		args = append(args, "--parent", parentID)
	}
	return p.runCLI(cleanArgs(args), "drive", "copyFile")
}

const driveBulkMaxOps = 50

func (p *provider) driveBulkExecute(_ context.Context, input map[string]any) (map[string]any, error) {
	raw, ok := input["operations"].([]any)
	if !ok || len(raw) == 0 {
		return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInvalidArgument, "message": "missing or empty operations array"}, errMissingRequest
	}
	if len(raw) > driveBulkMaxOps {
		return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInvalidArgument, "message": "operations exceeds max " + strconv.Itoa(driveBulkMaxOps)}, errors.New("operations exceeds max")
	}
	operations := make([]map[string]any, 0, len(raw))
	for i, r := range raw {
		m, ok := r.(map[string]any)
		if !ok {
			return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInvalidArgument, "message": "operations[" + strconv.Itoa(i) + "] must be an object"}, errMissingRequest
		}
		if strings.TrimSpace(asString(m["fileId"])) == "" {
			return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInvalidArgument, "message": "operations[" + strconv.Itoa(i) + "] missing fileId"}, errMissingFileID
		}
		operations = append(operations, m)
	}
	if asBool(input["validateOnly"]) {
		planned := make([]map[string]any, 0, len(operations))
		for _, m := range operations {
			planned = append(planned, m)
		}
		return map[string]any{
			"service":      "drive",
			"operation":    "bulkExecute",
			"validateOnly": true,
			"planned":      planned,
			"count":        len(planned),
		}, nil
	}
	if p == nil || p.exec == nil {
		return map[string]any{"service": "drive", "operation": "bulkExecute", "error_code": server.ErrorCodeInternal, "message": "executor not configured"}, errExecutorNotConfigured
	}
	baseArgs := []string{"--json"}
	baseArgs = append(baseArgs, policyArgs(input)...)
	baseArgs = append(baseArgs, maybeAccountArgs(input)...)
	var results []map[string]any
	var succeeded, failed int
	for _, m := range operations {
		op := strings.ToLower(strings.TrimSpace(asString(m["op"])))
		fileID := strings.TrimSpace(asString(m["fileId"]))
		var args []string
		switch op {
		case "move":
			parentID := strings.TrimSpace(asString(m["parentId"]))
			if parentID == "" {
				results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": "missing parentId"})
				failed++
				continue
			}
			args = append(append(append([]string{}, baseArgs...), "drive", "move", fileID, "--parent", parentID))
		case "rename":
			name := strings.TrimSpace(asString(m["name"]))
			if name == "" {
				results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": "missing name"})
				failed++
				continue
			}
			args = append(append(append([]string{}, baseArgs...), "drive", "rename", fileID, name))
		case "share":
			to := strings.TrimSpace(asString(m["to"]))
			if to == "" {
				results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": "missing to"})
				failed++
				continue
			}
			args = append(append([]string{}, baseArgs...), "drive", "share", fileID, "--to", to)
			if email := asString(m["email"]); email != "" {
				args = append(args, "--email", email)
			}
			if domain := asString(m["domain"]); domain != "" {
				args = append(args, "--domain", domain)
			}
			if role := asString(m["role"]); role != "" {
				args = append(args, "--role", role)
			}
		case "delete":
			args = append(append(append([]string{}, baseArgs...), "drive", "delete", fileID))
			if asBool(m["permanent"]) {
				args = append(args, "--permanent")
			}
			args = append(args, "--force")
		default:
			results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": "unsupported op"})
			failed++
			continue
		}
		stdout, stderr, err := p.exec(cleanArgs(args))
		if err != nil || strings.TrimSpace(stderr) != "" {
			results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": false, "error": strings.TrimSpace(stderr), "execErr": err != nil})
			failed++
			continue
		}
		var parsed map[string]any
		if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); jsonErr == nil {
			results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": true, "result": parsed})
		} else {
			results = append(results, map[string]any{"op": op, "fileId": fileID, "ok": true})
		}
		succeeded++
	}
	return map[string]any{
		"service":   "drive",
		"operation": "bulkExecute",
		"results":   results,
		"succeeded": succeeded,
		"failed":    failed,
	}, nil
}

func (p *provider) runCLI(args []string, service, operation string) (map[string]any, error) {
	if p == nil || p.exec == nil {
		return map[string]any{
			"service":    service,
			"operation":  operation,
			"error_code": server.ErrorCodeInternal,
			"message":    "mcp executor is not configured",
		}, errExecutorNotConfigured
	}

	stdout, stderr, execErr := p.exec(args)
	if execErr != nil && strings.TrimSpace(stderr) == "" {
		return map[string]any{
			"service":    service,
			"operation":  operation,
			"error_code": server.ErrorCodeAPI,
			"message":    execErr.Error(),
		}, execErr
	}
	if strings.TrimSpace(stderr) != "" {
		var parsed map[string]any
		if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &parsed); jsonErr == nil {
			if errObj, ok := parsed["error"].(map[string]any); ok {
				errObj["service"] = service
				errObj["operation"] = operation
				return errObj, errToolCommandFailed
			}
		}
		return map[string]any{"service": service, "operation": operation, "error_code": server.ErrorCodeAPI, "message": strings.TrimSpace(stderr)}, errToolStderr
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
		return map[string]any{"service": service, "operation": operation, "error_code": server.ErrorCodeInvalidJSON, "message": "failed to parse command output"}, fmt.Errorf("parse command output: %w", err)
	}
	parsed["service"] = service
	parsed["operation"] = operation
	return parsed, nil
}

func injectRequireRevision(requests map[string]any, revID string) map[string]any {
	revID = strings.TrimSpace(revID)
	if revID == "" {
		return requests
	}
	out := make(map[string]any, len(requests)+1)
	for k, v := range requests {
		out[k] = v
	}
	wc, _ := out["writeControl"].(map[string]any)
	if wc == nil {
		wc = make(map[string]any)
		out["writeControl"] = wc
	}
	wc["requiredRevisionId"] = revID
	return out
}

func writeTempJSON(v any) (string, error) {
	f, err := os.CreateTemp("", "gog-mcp-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	if _, err := f.Write(b); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	return f.Name(), nil
}

func maybeOpIDArgs(input map[string]any) []string {
	if opID := strings.TrimSpace(asString(input["opId"])); opID != "" {
		return []string{"--op-id", opID}
	}
	return nil
}

func maybeAccountArgs(input map[string]any) []string {
	acct := strings.TrimSpace(asString(input["account"]))
	// Ignore values that look like flags (e.g. client passing --json as account)
	if acct != "" && !strings.HasPrefix(acct, "-") {
		return []string{"--account", acct}
	}
	return nil
}

// cleanArgs trims and drops empty strings. It does not split on spaces so paths and values with spaces stay intact.
func cleanArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if s := strings.TrimSpace(a); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func asInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return i, true
		}
	}
	return 0, false
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

func trimFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func policyArgs(input map[string]any) []string {
	args := make([]string, 0, 6)
	if timeoutMs, ok := asInt(input["timeoutMs"]); ok && timeoutMs > 0 {
		if timeoutMs > 15*60*1000 {
			timeoutMs = 15 * 60 * 1000
		}
		args = append(args, "--request-timeout", fmt.Sprintf("%dms", timeoutMs))
	}
	if retries, ok := asInt(input["retries"]); ok {
		if retries < -1 {
			retries = -1
		}
		if retries > 10 {
			retries = 10
		}
		args = append(args, "--retries", strconv.FormatInt(retries, 10))
	}
	if retryBackoffMs, ok := asInt(input["retryBackoffMs"]); ok && retryBackoffMs > 0 {
		if retryBackoffMs > 30000 {
			retryBackoffMs = 30000
		}
		args = append(args, "--retry-backoff", fmt.Sprintf("%dms", retryBackoffMs))
	}
	return args
}
