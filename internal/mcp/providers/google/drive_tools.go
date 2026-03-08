package google

import (
	"github.com/steipete/gogcli/internal/mcp/server"
)

func driveSpecs(p *provider) []server.ToolSpec {
	return []server.ToolSpec{
		{
			Name:        "drive_ensureFolder",
			Description: "Ensure a folder path exists in Drive; create missing segments.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"path"},
				"properties": map[string]any{
					"path":     map[string]any{"type": "string"},
					"parentId": map[string]any{"type": "string"},
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
					"fileId": map[string]any{"type": "string"},
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
					"fileId":       map[string]any{"type": "string"},
					"permissionId": map[string]any{"type": "string"},
				},
			},
			Handler: p.driveGetPermission,
		}, {
			Name:        "drive_listFiles",
			Description: "List files and folders in Drive (default root; set parentId for a specific folder). Use max/maxResults for page size and page/pageToken for next page. Set global=true for cross-drive listing.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"parentId":      map[string]any{"type": "string"},
					"query":         map[string]any{"type": "string"},
					"max":           map[string]any{"type": "integer"},
					"page":          map[string]any{"type": "string"},
					"pageToken":     map[string]any{"type": "string"},
					"maxResults":    map[string]any{"type": "integer"},
					"pageSize":      map[string]any{"type": "integer"},
					"fetchAllPages": map[string]any{"type": "boolean", "description": "If true, fetch all pages and return totalCount"},
					"global":        map[string]any{"type": "boolean"},
					"allDrives": map[string]any{
						"type": "boolean",
					},
				},
			},
			Handler: p.driveListFiles,
		}, {
			Name:        "drive_searchFiles",
			Description: "Search files/folders in Drive. Use query with optional rawQuery, max/maxResults for N items, and page/pageToken for next page. Use fetchAllPages for count-all workflows.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "read-fast",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"query"},
				"properties": map[string]any{
					"query":         map[string]any{"type": "string"},
					"rawQuery":      map[string]any{"type": "boolean"},
					"max":           map[string]any{"type": "integer"},
					"page":          map[string]any{"type": "string"},
					"pageToken":     map[string]any{"type": "string"},
					"maxResults":    map[string]any{"type": "integer"},
					"pageSize":      map[string]any{"type": "integer"},
					"fetchAllPages": map[string]any{"type": "boolean", "description": "If true, fetch all pages and return totalCount"},
					"allDrives":     map[string]any{"type": "boolean"},
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
					"fileId":    map[string]any{"type": "string"},
					"pageCount": map[string]any{"type": "boolean", "description": "Include PDF page count if available"},
				},
			},
			Handler: p.driveGetFile,
		}, {
			Name:        "drive_uploadFile",
			Description: "Upload a local file to Drive. Optionally set parentId, name, or replaceFileId.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-heavy",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"localPath"},
				"properties": map[string]any{
					"localPath": map[string]any{"type": "string", "description": "Path on the server where gog runs"},
					"name":      map[string]any{"type": "string", "description": "Drive filename (defaults to local file name)"},
					"parentId":  map[string]any{"type": "string", "description": "Target Drive folder ID"},
					"replaceFileId": map[string]any{
						"type":        "string",
						"description": "Overwrite an existing Drive file ID (keeps sharing); cannot combine with parentId",
					},
					"mimeType":            map[string]any{"type": "string"},
					"convert":             map[string]any{"type": "boolean"},
					"convertTo":           map[string]any{"type": "string"},
					"keepRevisionForever": map[string]any{"type": "boolean", "description": "Keep this revision forever (e.g. for backup retention)"},
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
					"fileId": map[string]any{"type": "string"},
					"out":    map[string]any{"type": "string"},
					"format": map[string]any{"type": "string"},
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
					"fileId": map[string]any{"type": "string"},
					"max":    map[string]any{"type": "integer"},
					"page":   map[string]any{"type": "string"},
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
					"fileId":        map[string]any{"type": "string"},
					"max":           map[string]any{"type": "integer"},
					"page":          map[string]any{"type": "string"},
					"all":           map[string]any{"type": "boolean"},
					"includeQuoted": map[string]any{"type": "boolean"},
					"failEmpty":     map[string]any{"type": "boolean"},
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
					"fileId":       map[string]any{"type": "string"},
					"permanent":    map[string]any{"type": "boolean"},
					"force":        map[string]any{"type": "boolean"},
					"validateOnly": map[string]any{"type": "boolean", "description": "If true, return planned action without executing"},
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
					"fileId": map[string]any{"type": "string"},
					"name":   map[string]any{"type": "string", "description": "New file or folder name"},
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
				},
			},
			Handler: p.driveShareFile,
		}, {
			Name:        "drive_unshare",
			Description: "Remove a permission from a Drive file.",
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
				},
			},
			Handler: p.driveUnshare,
		}, {
			Name:        "drive_createComment",
			Description: "Create a comment on a Drive file.",
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
				},
			},
			Handler: p.driveCreateComment,
		}, {
			Name:        "drive_deleteComment",
			Description: "Delete a comment on a Drive file.",
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
				},
			},
			Handler: p.driveDeleteComment,
		}, {
			Name:        "drive_copyFile",
			Description: "Copy a Drive file to a new name and optionally to a folder.",
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
				},
			},
			Handler: p.driveCopyFile,
		}, {
			Name:        "drive_bulkExecute",
			Description: "Execute up to 50 Drive operations (move/rename/share/delete); validateOnly previews first.",
			Tier:        "ga",
			Version:     "v1",
			PolicyClass: "write-safe",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"operations"},
				"properties": map[string]any{
					"operations":   map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "List of { op: move|rename|share|delete, fileId, ... }"},
					"validateOnly": map[string]any{"type": "boolean"},
				},
			},
			Handler: p.driveBulkExecute,
		},
	}
}
