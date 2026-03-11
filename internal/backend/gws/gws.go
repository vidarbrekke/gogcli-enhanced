package gws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Result holds raw output from a gws invocation.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Path returns the gws binary path: GOG_GWS_PATH env if set, else "gws".
func Path() string {
	if p := strings.TrimSpace(os.Getenv("GOG_GWS_PATH")); p != "" {
		return p
	}

	return "gws"
}

// RunLabelsList runs: gws gmail users labels list --params '{"userId":"me"}'.
func RunLabelsList(ctx context.Context) (Result, error) {
	params := `{"userId":"me"}`
	return run(ctx, []string{"gmail", "users", "labels", "list", "--params", params})
}

// RunLabelsGet runs: gws gmail users labels get --params '{"userId":"me","id":"<id>"}'.
func RunLabelsGet(ctx context.Context, labelID string) (Result, error) {
	params := fmt.Sprintf(`{"userId":"me","id":%q}`, labelID)
	return run(ctx, []string{"gmail", "users", "labels", "get", "--params", params})
}

// RunDriveLs runs: gws drive files list (or equivalent) with params for folder list.
// parentID is the folder ID (e.g. "root"); pageToken for pagination; max is page size.
func RunDriveLs(ctx context.Context, parentID, pageToken string, pageSize int64) (Result, error) {
	if parentID == "" {
		parentID = "root"
	}

	if pageSize <= 0 {
		pageSize = 20
	}

	// Drive API v3 list: q = "'parentId' in parents and trashed = false", pageSize, pageToken.
	q := fmt.Sprintf("'%s' in parents and trashed = false", parentID)
	params := fmt.Sprintf(`{"pageSize":%d,"pageToken":%q,"q":%q}`, pageSize, pageToken, q)

	return run(ctx, []string{"drive", "files", "list", "--params", params})
}

// RunDriveGet runs: gws drive files get --params '{"fileId":"<id>"}'.
func RunDriveGet(ctx context.Context, fileID string) (Result, error) {
	if fileID == "" {
		return Result{ExitCode: 1}, nil
	}
	params := fmt.Sprintf(`{"fileId":%q}`, fileID)

	return run(ctx, []string{"drive", "files", "get", "--params", params})
}

// RunDriveSearch runs: gws drive files list --params '{"q":"<query>","pageSize":N,"pageToken":"..."}'.
func RunDriveSearch(ctx context.Context, query, pageToken string, pageSize int64) (Result, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	params := fmt.Sprintf(`{"q":%q,"pageSize":%d,"pageToken":%q}`, query, pageSize, pageToken)

	return run(ctx, []string{"drive", "files", "list", "--params", params})
}

func run(ctx context.Context, args []string) (Result, error) {
	bin := Path()
	// #nosec G204 -- bin is from GOG_GWS_PATH env or literal "gws", not user input
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return Result{
				Stdout:   bytes.TrimSpace(stdout.Bytes()),
				Stderr:   bytes.TrimSpace(stderr.Bytes()),
				ExitCode: exitErr.ExitCode(),
			}, nil
		}

		return Result{}, fmt.Errorf("run gws: %w", runErr)
	}

	return Result{
		Stdout:   bytes.TrimSpace(stdout.Bytes()),
		Stderr:   bytes.TrimSpace(stderr.Bytes()),
		ExitCode: 0,
	}, nil
}

// HasTopLevelError returns true if stdout or stderr contains JSON with a top-level "error" key.
func HasTopLevelError(stdout, stderr []byte) bool {
	for _, raw := range [][]byte{stdout, stderr} {
		if len(raw) == 0 {
			continue
		}

		var m map[string]json.RawMessage
		if json.Unmarshal(raw, &m) == nil {
			if _, ok := m["error"]; ok {
				return true
			}
		}
	}

	return false
}
