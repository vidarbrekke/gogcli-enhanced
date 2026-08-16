package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/secrets"
	"github.com/steipete/gogcli/internal/ui"
)

func writeFakeGWSBinary(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "gws")
	script := `#!/usr/bin/env bash
set -euo pipefail

if [[ -n "${GOG_GWS_ARGS_FILE:-}" ]]; then
  printf '%s\n' "$*" >> "$GOG_GWS_ARGS_FILE"
fi

if [[ "$1" == "gmail" && "$2" == "users" && "$3" == "labels" && "$4" == "list" ]]; then
  cat <<'EOF'
{"labels":[{"id":"INBOX","name":"INBOX","type":"system"}]}
EOF
  exit 0
fi

if [[ "$1" == "gmail" && "$2" == "users" && "$3" == "labels" && "$4" == "get" ]]; then
  cat <<'EOF'
{"id":"INBOX","name":"INBOX","type":"system","messagesTotal":1,"messagesUnread":2,"threadsTotal":3,"threadsUnread":4}
EOF
  exit 0
fi

if [[ "$1" == "drive" && "$2" == "files" && "$3" == "get" ]]; then
  cat <<'EOF'
{"id":"fileId123","name":"Doc","mimeType":"application/pdf","size":"1024","createdTime":"2025-01-01T00:00:00Z","modifiedTime":"2025-12-12T14:37:47Z","webViewLink":"https://example.test/doc"}
EOF
  exit 0
fi

if [[ "$1" == "drive" && "$2" == "files" && "$3" == "list" ]]; then
  cat <<'EOF'
{"files":[{"id":"f1","name":"Doc","mimeType":"application/pdf","size":"1024","modifiedTime":"2025-12-12T14:37:47Z"}],"nextPageToken":"npt"}
EOF
  exit 0
fi

echo "unexpected args: $*" >&2
exit 1
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake gws: %v", err)
	}
	return path
}

func trackNativeDriveService(t *testing.T) *bool {
	t.Helper()
	nativeCalled := false
	origNew := newDriveService
	newDriveService = func(ctx context.Context, account string) (*drive.Service, error) {
		nativeCalled = true
		return origNew(ctx, account)
	}
	t.Cleanup(func() { newDriveService = origNew })
	return &nativeCalled
}

func withDefaultAccount(t *testing.T, email string) {
	t.Helper()

	prev := openSecretsStoreForAccount
	t.Cleanup(func() { openSecretsStoreForAccount = prev })
	openSecretsStoreForAccount = func() (secrets.Store, error) {
		return &fakeSecretsStore{defaultAccount: email}, nil
	}
}

func TestGmailLabelsGetCmd_GWS_TextParity(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")
	t.Setenv("GOG_GWS_PATH", writeFakeGWSBinary(t))
	withDefaultAccount(t, "default@example.com")

	out := captureStdout(t, func() {
		ctx := newTestUIContext(t, outfmt.Mode{})
		if err := runKong(t, &GmailLabelsGetCmd{}, []string{"INBOX"}, ctx, &RootFlags{}); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if !strings.Contains(out, "id\tINBOX") || !strings.Contains(out, "messages_total\t1") || !strings.Contains(out, "threads_unread\t4") {
		t.Fatalf("unexpected GWS text output: %q", out)
	}
}

func TestDriveLsCmd_GWS_TextParity(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")
	t.Setenv("GOG_GWS_PATH", writeFakeGWSBinary(t))
	withDefaultAccount(t, "default@example.com")

	var errBuf strings.Builder
	out := captureStdout(t, func() {
		u, err := ui.New(ui.Options{Stdout: os.Stdout, Stderr: &errBuf, Color: "never"})
		if err != nil {
			t.Fatalf("ui.New: %v", err)
		}
		ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{})
		if err := runKong(t, &DriveLsCmd{}, []string{}, ctx, &RootFlags{}); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if !strings.Contains(out, "1.0 KB") || !strings.Contains(out, "2025-12-12 14:37") {
		t.Fatalf("unexpected GWS drive ls text output: %q", out)
	}
	if strings.Contains(errBuf.String(), "--page") {
		t.Fatalf("unexpected next-page hint on GWS path: %q", errBuf.String())
	}
}

func TestDriveLsCmd_GWS_RejectsExplicitAccount(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")

	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	err := runKong(t, &DriveLsCmd{}, []string{}, ctx, &RootFlags{Account: "other@example.com"})
	if err == nil || !strings.Contains(err.Error(), "explicit --account is not supported with GOG_BACKEND=gws") {
		t.Fatalf("expected explicit-account rejection, got: %v", err)
	}
}

func TestDriveLsCmd_GWS_AllowsAutoAccount(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")
	t.Setenv("GOG_GWS_PATH", writeFakeGWSBinary(t))
	withDefaultAccount(t, "default@example.com")

	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	err := runKong(t, &DriveLsCmd{}, []string{}, ctx, &RootFlags{Account: "default"})
	if err != nil {
		t.Fatalf("expected auto/default account to be accepted, got: %v", err)
	}
}

func TestGmailLabelsGetCmd_GWS_RejectsExplicitAccountFromEnv(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")
	t.Setenv("GOG_ACCOUNT", "other@example.com")

	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	err := runKong(t, &GmailLabelsGetCmd{}, []string{"INBOX"}, ctx, &RootFlags{})
	if err == nil || !strings.Contains(err.Error(), "explicit GOG_ACCOUNT is not supported with GOG_BACKEND=gws") {
		t.Fatalf("expected explicit env-account rejection, got: %v", err)
	}
}

func TestDriveGetCmd_GOG_BACKEND_gws_uses_gws_path(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")
	argsFile := filepath.Join(t.TempDir(), "gws-args.txt")
	t.Setenv("GOG_GWS_ARGS_FILE", argsFile)
	t.Setenv("GOG_GWS_PATH", writeFakeGWSBinary(t))
	withDefaultAccount(t, "default@example.com")
	nativeCalled := trackNativeDriveService(t)

	out := captureStdout(t, func() {
		ctx := newTestUIContext(t, outfmt.Mode{JSON: true})
		if err := runKong(t, &DriveGetCmd{}, []string{"fileId123"}, ctx, &RootFlags{}); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if *nativeCalled {
		t.Fatal("GOG_BACKEND=gws must not call native newDriveService")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json: %v (out=%q)", err, out)
	}
	file, _ := payload["file"].(map[string]any)
	if file == nil || file["id"] != "fileId123" {
		t.Fatalf("expected wrapped file.id=fileId123, got %#v", payload)
	}
	argsRaw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args log: %v", err)
	}
	argsLine := strings.TrimSpace(string(argsRaw))
	if !strings.Contains(argsLine, "drive files get") || !strings.Contains(argsLine, "fileId123") {
		t.Fatalf("expected gws drive files get with fileId123, got %q", argsLine)
	}
}

func TestDriveSearchCmd_GOG_BACKEND_gws_uses_gws_path(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")
	argsFile := filepath.Join(t.TempDir(), "gws-args.txt")
	t.Setenv("GOG_GWS_ARGS_FILE", argsFile)
	t.Setenv("GOG_GWS_PATH", writeFakeGWSBinary(t))
	withDefaultAccount(t, "default@example.com")
	nativeCalled := trackNativeDriveService(t)

	out := captureStdout(t, func() {
		ctx := newTestUIContext(t, outfmt.Mode{JSON: true})
		if err := runKong(t, &DriveSearchCmd{}, []string{"foo"}, ctx, &RootFlags{}); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if *nativeCalled {
		t.Fatal("GOG_BACKEND=gws must not call native newDriveService")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json: %v (out=%q)", err, out)
	}
	files, _ := payload["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("expected files[1], got %#v", payload)
	}
	argsRaw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args log: %v", err)
	}
	argsLine := strings.TrimSpace(string(argsRaw))
	if !strings.Contains(argsLine, "drive files list") || !strings.Contains(argsLine, "fullText contains 'foo'") {
		t.Fatalf("expected gws drive files list with search q, got %q", argsLine)
	}
}

func TestDriveLsCmd_GOG_BACKEND_gws_uses_gws_path(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")
	argsFile := filepath.Join(t.TempDir(), "gws-args.txt")
	t.Setenv("GOG_GWS_ARGS_FILE", argsFile)
	t.Setenv("GOG_GWS_PATH", writeFakeGWSBinary(t))
	withDefaultAccount(t, "default@example.com")
	nativeCalled := trackNativeDriveService(t)

	out := captureStdout(t, func() {
		ctx := newTestUIContext(t, outfmt.Mode{JSON: true})
		if err := runKong(t, &DriveLsCmd{}, []string{}, ctx, &RootFlags{}); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if *nativeCalled {
		t.Fatal("GOG_BACKEND=gws must not call native newDriveService")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json: %v (out=%q)", err, out)
	}
	files, _ := payload["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("expected files[1], got %#v", payload)
	}
	argsRaw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args log: %v", err)
	}
	argsLine := strings.TrimSpace(string(argsRaw))
	if !strings.Contains(argsLine, "drive files list") || !strings.Contains(argsLine, "'root' in parents") {
		t.Fatalf("expected gws drive files list for root folder, got %q", argsLine)
	}
}
