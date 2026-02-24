//nolint:wsl_v5 // command argument composition is intentionally linear
package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/steipete/gogcli/internal/mcp/server"
)

type Executor func(args []string) (stdout string, stderr string, err error)

var (
	errMissingDocID              = errors.New("missing docId")
	errMissingRequest            = errors.New("missing request")
	errMissingPath               = errors.New("missing path")
	errMissingFileID             = errors.New("missing fileId")
	errMissingFileOrPermissionID = errors.New("missing fileId or permissionId")
	errToolCommandFailed         = errors.New("tool command failed")
	errToolStderr                = errors.New("tool stderr")
	errExecutorNotConfigured     = errors.New("executor not configured")
)

func Register(s *server.Server, executor Executor) {
	s.RegisterToolSpec(server.ToolSpec{
		Name:        "docs.planBatch",
		Description: "Validate and plan a Docs batch update request without applying changes.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"docId", "request"},
			"properties": map[string]any{
				"docId":   map[string]any{"type": "string"},
				"request": map[string]any{"type": "object"},
				"opId":    map[string]any{"type": "string"},
			},
		},
		Handler: docsPlanBatch,
	})
	s.RegisterToolSpec(server.ToolSpec{
		Name:        "docs.executeBatch",
		Description: "Execute a Docs batch update request.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"docId", "request"},
			"properties": map[string]any{
				"docId":   map[string]any{"type": "string"},
				"request": map[string]any{"type": "object"},
				"opId":    map[string]any{"type": "string"},
			},
		},
		Handler: docsExecuteBatch,
	})
	s.RegisterToolSpec(server.ToolSpec{
		Name:        "drive.ensureFolder",
		Description: "Ensure a folder path exists in Drive; create missing segments.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"path"},
			"properties": map[string]any{
				"path":     map[string]any{"type": "string"},
				"parentId": map[string]any{"type": "string"},
				"account":  map[string]any{"type": "string"},
				"opId":     map[string]any{"type": "string"},
			},
		},
		Handler: driveEnsureFolder,
	})
	s.RegisterToolSpec(server.ToolSpec{
		Name:        "drive.untrash",
		Description: "Restore a trashed Drive file.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"fileId"},
			"properties": map[string]any{
				"fileId":  map[string]any{"type": "string"},
				"account": map[string]any{"type": "string"},
				"opId":    map[string]any{"type": "string"},
			},
		},
		Handler: driveUntrash,
	})
	s.RegisterToolSpec(server.ToolSpec{
		Name:        "drive.getPermission",
		Description: "Get one permission entry for a Drive file.",
		InputSchema: map[string]any{
			"type":     "object",
			"required": []string{"fileId", "permissionId"},
			"properties": map[string]any{
				"fileId":       map[string]any{"type": "string"},
				"permissionId": map[string]any{"type": "string"},
				"account":      map[string]any{"type": "string"},
				"opId":         map[string]any{"type": "string"},
			},
		},
		Handler: driveGetPermission,
	})
	execCommand = executor
}

var execCommand Executor

func docsPlanBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "planBatch", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "docs", "operation": "planBatch", "error_code": "invalid_argument", "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "batch", docID, "--requests-file", path, "--validate-only")
	return runCLI(cleanArgs(args), "docs", "planBatch")
}

func docsExecuteBatch(_ context.Context, input map[string]any) (map[string]any, error) {
	docID := strings.TrimSpace(asString(input["docId"]))
	if docID == "" {
		return map[string]any{"service": "docs", "operation": "executeBatch", "error_code": "invalid_argument", "message": "missing docId"}, errMissingDocID
	}
	requests, ok := input["request"].(map[string]any)
	if !ok {
		return map[string]any{"service": "docs", "operation": "executeBatch", "error_code": "invalid_argument", "message": "missing request object"}, errMissingRequest
	}
	path, err := writeTempJSON(requests)
	if err != nil {
		return nil, err
	}
	defer os.Remove(path)
	args := []string{"--json"}
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "docs", "edit", "batch", docID, "--requests-file", path)
	return runCLI(cleanArgs(args), "docs", "executeBatch")
}

func driveEnsureFolder(_ context.Context, input map[string]any) (map[string]any, error) {
	path := strings.TrimSpace(asString(input["path"]))
	if path == "" {
		return map[string]any{"service": "drive", "operation": "ensureFolder", "error_code": "invalid_argument", "message": "missing path"}, errMissingPath
	}
	args := []string{"--json"}
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "ensure-folder", path)
	if parent := strings.TrimSpace(asString(input["parentId"])); parent != "" {
		args = append(args, "--parent", parent)
	}
	return runCLI(cleanArgs(args), "drive", "ensureFolder")
}

func driveUntrash(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "untrash", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json"}
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "untrash", fileID)
	return runCLI(cleanArgs(args), "drive", "untrash")
}

func driveGetPermission(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	permissionID := strings.TrimSpace(asString(input["permissionId"]))
	if fileID == "" || permissionID == "" {
		return map[string]any{"service": "drive", "operation": "getPermission", "error_code": "invalid_argument", "message": "missing fileId or permissionId"}, errMissingFileOrPermissionID
	}
	args := []string{"--json"}
	args = append(args, maybeAccountArgs(input)...)
	args = append(args, maybeOpIDArgs(input)...)
	args = append(args, "drive", "permission", fileID, permissionID)
	return runCLI(cleanArgs(args), "drive", "getPermission")
}

func runCLI(args []string, service, operation string) (map[string]any, error) {
	if execCommand == nil {
		return map[string]any{
			"service":    service,
			"operation":  operation,
			"error_code": "internal_error",
			"message":    "mcp executor is not configured",
		}, errExecutorNotConfigured
	}

	stdout, stderr, execErr := execCommand(args)
	if execErr != nil && strings.TrimSpace(stderr) == "" {
		return map[string]any{
			"service":    service,
			"operation":  operation,
			"error_code": "api_error",
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
		return map[string]any{"service": service, "operation": operation, "error_code": "api_error", "message": strings.TrimSpace(stderr)}, errToolStderr
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &parsed); err != nil {
		return map[string]any{"service": service, "operation": operation, "error_code": "invalid_json", "message": "failed to parse command output"}, fmt.Errorf("parse command output: %w", err)
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
	if acct := strings.TrimSpace(asString(input["account"])); acct != "" {
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
