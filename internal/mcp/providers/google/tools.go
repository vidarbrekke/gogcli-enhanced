//nolint:wsl_v5 // command argument composition is intentionally linear
package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/steipete/gogcli/internal/cmd"
	"github.com/steipete/gogcli/internal/mcp/server"
)

var (
	errMissingDocID              = errors.New("missing docId")
	errMissingRequest            = errors.New("missing request")
	errMissingPath               = errors.New("missing path")
	errMissingFileID             = errors.New("missing fileId")
	errMissingFileOrPermissionID = errors.New("missing fileId or permissionId")
	errToolCommandFailed         = errors.New("tool command failed")
	errToolStderr                = errors.New("tool stderr")
)

func Register(s *server.Server) {
	s.RegisterTool("docs.planBatch", docsPlanBatch)
	s.RegisterTool("docs.executeBatch", docsExecuteBatch)
	s.RegisterTool("drive.ensureFolder", driveEnsureFolder)
	s.RegisterTool("drive.untrash", driveUntrash)
	s.RegisterTool("drive.getPermission", driveGetPermission)
}

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
	args := []string{"--json", maybeOpID(input), "docs", "edit", "batch", docID, "--requests-file", path, "--validate-only"}
	return runCLI(cleanArgs(args), "docs", "planBatch") //nolint:contextcheck // cmd.Execute has no context-aware variant
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
	args := []string{"--json", maybeOpID(input), "docs", "edit", "batch", docID, "--requests-file", path}
	return runCLI(cleanArgs(args), "docs", "executeBatch") //nolint:contextcheck // cmd.Execute has no context-aware variant
}

func driveEnsureFolder(_ context.Context, input map[string]any) (map[string]any, error) {
	path := strings.TrimSpace(asString(input["path"]))
	if path == "" {
		return map[string]any{"service": "drive", "operation": "ensureFolder", "error_code": "invalid_argument", "message": "missing path"}, errMissingPath
	}
	args := []string{"--json", maybeAccount(input), maybeOpID(input), "drive", "ensure-folder", path}
	if parent := strings.TrimSpace(asString(input["parentId"])); parent != "" {
		args = append(args, "--parent", parent)
	}
	return runCLI(cleanArgs(args), "drive", "ensureFolder") //nolint:contextcheck // cmd.Execute has no context-aware variant
}

func driveUntrash(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	if fileID == "" {
		return map[string]any{"service": "drive", "operation": "untrash", "error_code": "invalid_argument", "message": "missing fileId"}, errMissingFileID
	}
	args := []string{"--json", maybeAccount(input), maybeOpID(input), "drive", "untrash", fileID}
	return runCLI(cleanArgs(args), "drive", "untrash") //nolint:contextcheck // cmd.Execute has no context-aware variant
}

func driveGetPermission(_ context.Context, input map[string]any) (map[string]any, error) {
	fileID := strings.TrimSpace(asString(input["fileId"]))
	permissionID := strings.TrimSpace(asString(input["permissionId"]))
	if fileID == "" || permissionID == "" {
		return map[string]any{"service": "drive", "operation": "getPermission", "error_code": "invalid_argument", "message": "missing fileId or permissionId"}, errMissingFileOrPermissionID
	}
	args := []string{"--json", maybeAccount(input), maybeOpID(input), "drive", "permission", fileID, permissionID}
	return runCLI(cleanArgs(args), "drive", "getPermission") //nolint:contextcheck // cmd.Execute has no context-aware variant
}

func runCLI(args []string, service, operation string) (map[string]any, error) {
	stdout, stderr := captureOutput(func() {
		_ = cmd.Execute(args)
	})
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

func captureOutput(fn func()) (string, string) {
	oldOut := os.Stdout
	oldErr := os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout = outW
	os.Stderr = errW
	fn()
	_ = outW.Close()
	_ = errW.Close()
	outBytes, _ := io.ReadAll(outR)
	errBytes, _ := io.ReadAll(errR)
	os.Stdout = oldOut
	os.Stderr = oldErr
	return string(outBytes), string(errBytes)
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

func maybeOpID(input map[string]any) string {
	if opID := strings.TrimSpace(asString(input["opId"])); opID != "" {
		return "--op-id " + opID
	}
	return ""
}

func maybeAccount(input map[string]any) string {
	if acct := strings.TrimSpace(asString(input["account"])); acct != "" {
		return "--account " + acct
	}
	return ""
}

func cleanArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if s := strings.TrimSpace(a); s != "" {
			for _, p := range strings.Split(s, " ") {
				if strings.TrimSpace(p) != "" {
					out = append(out, p)
				}
			}
		}
	}
	return out
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}
