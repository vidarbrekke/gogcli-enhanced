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

func Register(s *server.Server, executor Executor) {
	p := &provider{exec: executor}
	toolSpecs := []server.ToolSpec{
		{
			Name:        "docs.planBatch",
			Description: "Validate and plan a Docs batch update request without applying changes.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "request"},
				"properties": map[string]any{
					"docId":   map[string]any{"type": "string"},
					"request": map[string]any{"type": "object"},
					"opId":    map[string]any{"type": "string"},
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
			Name:        "docs.executeBatch",
			Description: "Execute a Docs batch update request.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-heavy",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"docId", "request"},
				"properties": map[string]any{
					"docId":   map[string]any{"type": "string"},
					"request": map[string]any{"type": "object"},
					"opId":    map[string]any{"type": "string"},
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
			Name:        "docs.sed",
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
			Name:        "docs.smartEdit",
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
			Name:        "docs.create",
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
			Name:        "docs.createWithBody",
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
			Name:        "docs.insertText",
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
			Name:        "docs.deleteRange",
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
			Name:        "docs.replaceAllText",
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
			Name:        "docs.appendText",
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
			Name:        "docs.insertTable",
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
			Name:        "docs.insertImage",
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
			Name:        "docs.locatorEdit",
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
			Name:        "sheets.planBatch",
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
			Name:        "sheets.executeBatch",
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
			Name:        "sheets.valuesUpdate",
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
			Name:        "sheets.valuesAppend",
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
			Name:        "slides.planBatch",
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
			Name:        "slides.executeBatch",
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
			Name:        "slides.replaceText",
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
			Name:        "slides.createSlide",
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
			Name:        "drive.ensureFolder",
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
			Name:        "drive.untrash",
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
			Name:        "drive.getPermission",
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
			Name:        "drive.listFiles",
			Description: "List files in Drive folder context.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"parentId": map[string]any{"type": "string"},
					"query":    map[string]any{"type": "string"},
					"max":      map[string]any{"type": "integer"},
					"page":     map[string]any{"type": "string"},
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
		}, 		{
			Name:        "drive.searchFiles",
			Description: "Search files and folders in Google Drive. To check if a folder exists by name: use query with the folder name (e.g. 'testing123'); results include My Drive and shared drives when allDrives is true (default). Use rawQuery: true with Drive query language (e.g. \"name contains 'testing123' and mimeType = 'application/vnd.google-apps.folder'\") for folder-only matches.",
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
			Name:        "drive.getFile",
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
			Name:        "drive.uploadFile",
			Description: "Upload local file to Drive.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-heavy",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"localPath"},
				"properties": map[string]any{
					"localPath": map[string]any{"type": "string"},
					"name":      map[string]any{"type": "string"},
					"parentId":  map[string]any{"type": "string"},
					"replaceFileId": map[string]any{
						"type": "string",
					},
					"mimeType":       map[string]any{"type": "string"},
					"convert":        map[string]any{"type": "boolean"},
					"convertTo":      map[string]any{"type": "string"},
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveUploadFile,
		}, {
			Name:        "drive.downloadFile",
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
			Name:        "drive.listPermissions",
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
			Name:        "drive.listComments",
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
			Name:        "drive.deleteFile",
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
					"account":        map[string]any{"type": "string"},
					"opId":           map[string]any{"type": "string"},
					"timeoutMs":      map[string]any{"type": "integer"},
					"retries":        map[string]any{"type": "integer"},
					"retryBackoffMs": map[string]any{"type": "integer"},
				},
			},
			Handler: p.driveDeleteFile,
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
	path, err := writeTempJSON(requests)
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
	path, err := writeTempJSON(requests)
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

func (p *provider) driveListFiles(_ context.Context, input map[string]any) (map[string]any, error) {
	args := []string{"--json"}
	args = append(args, policyArgs(input)...)
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "ls")
	if parent := strings.TrimSpace(asString(input["parentId"])); parent != "" {
		args = append(args, "--parent", parent)
	}
	if query := strings.TrimSpace(asString(input["query"])); query != "" {
		args = append(args, "--query", query)
	}
	if max, ok := asInt(input["max"]); ok && max > 0 {
		args = append(args, "--max", strconv.FormatInt(max, 10))
	}
	if page := strings.TrimSpace(asString(input["page"])); page != "" {
		args = append(args, "--page", page)
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
	if page := strings.TrimSpace(asString(input["page"])); page != "" {
		args = append(args, "--page", page)
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
