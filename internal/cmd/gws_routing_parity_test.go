package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/gmail/v1"

	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/secrets"
	"github.com/steipete/gogcli/internal/ui"
)

// Fake gws harness for live GOG_BACKEND=gws routing.
// Env: GOG_GWS_PATH (binary), GOG_GWS_ARGS_FILE (argv log), GOG_GWS_FAKE_MODE=error (401 JSON).

func writeFakeGWSBinary(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "gws")
	script := `#!/usr/bin/env bash
set -euo pipefail

if [[ -n "${GOG_GWS_ARGS_FILE:-}" ]]; then
  printf '%s\n' "$*" >> "$GOG_GWS_ARGS_FILE"
fi

if [[ "${GOG_GWS_FAKE_MODE:-}" == "error" ]]; then
  cat <<'EOF'
{"error":{"code":401,"message":"Request had invalid authentication credentials.","reason":"authError"}}
EOF
  exit 1
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

func withDefaultAccount(t *testing.T, email string) {
	t.Helper()
	prev := openSecretsStoreForAccount
	t.Cleanup(func() { openSecretsStoreForAccount = prev })
	openSecretsStoreForAccount = func() (secrets.Store, error) {
		return &fakeSecretsStore{defaultAccount: email}, nil
	}
}

func setupGWS(t *testing.T) (argsFile string) {
	t.Helper()
	t.Setenv("GOG_BACKEND", "gws")
	t.Setenv("GOG_ACCOUNT", "")
	argsFile = filepath.Join(t.TempDir(), "gws-args.txt")
	t.Setenv("GOG_GWS_ARGS_FILE", argsFile)
	t.Setenv("GOG_GWS_PATH", writeFakeGWSBinary(t))
	withDefaultAccount(t, "default@example.com")
	return argsFile
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

func trackNativeGmailService(t *testing.T) *bool {
	t.Helper()
	nativeCalled := false
	origNew := newGmailService
	newGmailService = func(ctx context.Context, account string) (*gmail.Service, error) {
		nativeCalled = true
		return origNew(ctx, account)
	}
	t.Cleanup(func() { newGmailService = origNew })
	return &nativeCalled
}

func readGWSArgs(t *testing.T, argsFile string) string {
	t.Helper()
	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args log: %v", err)
	}
	return strings.TrimSpace(string(raw))
}

func assertArgvContains(t *testing.T, argsLine string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(argsLine, n) {
			t.Fatalf("argv missing %q; got %q", n, argsLine)
		}
	}
}

// --- Account policy (once; shared by all gws-routed commands) ---

func TestGWS_RejectsExplicitAccountFlag(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")
	t.Setenv("GOG_ACCOUNT", "")
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	err := runKong(t, &DriveLsCmd{}, []string{}, ctx, &RootFlags{Account: "other@example.com"})
	if err == nil || !strings.Contains(err.Error(), "explicit --account is not supported with GOG_BACKEND=gws") {
		t.Fatalf("expected explicit-account rejection, got: %v", err)
	}
}

func TestGWS_RejectsExplicitAccountEnv(t *testing.T) {
	t.Setenv("GOG_BACKEND", "gws")
	t.Setenv("GOG_ACCOUNT", "other@example.com")
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})
	err := runKong(t, &GmailLabelsGetCmd{}, []string{"INBOX"}, ctx, &RootFlags{})
	if err == nil || !strings.Contains(err.Error(), "explicit GOG_ACCOUNT is not supported with GOG_BACKEND=gws") {
		t.Fatalf("expected explicit env-account rejection, got: %v", err)
	}
}

func TestGWS_AllowsAutoAccount(t *testing.T) {
	argsFile := setupGWS(t)
	nativeCalled := trackNativeDriveService(t)
	_ = captureStdout(t, func() {
		ctx := newTestUIContext(t, outfmt.Mode{JSON: true})
		if err := runKong(t, &DriveLsCmd{}, []string{}, ctx, &RootFlags{Account: "default"}); err != nil {
			t.Fatalf("expected auto/default account to be accepted, got: %v", err)
		}
	})
	if *nativeCalled {
		t.Fatal("auto account must still use gws path")
	}
	assertArgvContains(t, readGWSArgs(t, argsFile), "drive files list")
}

// --- Routed command success (JSON + argv + native not called; text where format is unique) ---

func TestGWS_RoutedCommands(t *testing.T) {
	tests := []struct {
		name           string
		cmd            any
		args           []string
		track          string // "drive" | "gmail"
		wantArgv       []string
		checkJSON      func(t *testing.T, payload map[string]any)
		checkText      func(t *testing.T, out, errOut string)
		wantTextArgvOK bool // text mode still invokes gws (labels get does list+get)
	}{
		{
			name:     "gmail labels list",
			cmd:      &GmailLabelsListCmd{},
			args:     nil,
			track:    "gmail",
			wantArgv: []string{"gmail users labels list"},
			checkJSON: func(t *testing.T, payload map[string]any) {
				t.Helper()
				labels, _ := payload["labels"].([]any)
				if len(labels) != 1 {
					t.Fatalf("expected labels[1], got %#v", payload)
				}
			},
		},
		{
			name:     "gmail labels get",
			cmd:      &GmailLabelsGetCmd{},
			args:     []string{"INBOX"},
			track:    "gmail",
			wantArgv: []string{"gmail users labels get", "INBOX"},
			checkJSON: func(t *testing.T, payload map[string]any) {
				t.Helper()
				label, _ := payload["label"].(map[string]any)
				if label == nil || label["id"] != "INBOX" {
					t.Fatalf("expected wrapped label.id=INBOX, got %#v", payload)
				}
			},
			checkText: func(t *testing.T, out, _ string) {
				t.Helper()
				if !strings.Contains(out, "id\tINBOX") || !strings.Contains(out, "messages_total\t1") || !strings.Contains(out, "threads_unread\t4") {
					t.Fatalf("unexpected labels get text: %q", out)
				}
			},
			wantTextArgvOK: true,
		},
		{
			name:     "drive ls",
			cmd:      &DriveLsCmd{},
			args:     nil,
			track:    "drive",
			wantArgv: []string{"drive files list", "'root' in parents"},
			checkJSON: func(t *testing.T, payload map[string]any) {
				t.Helper()
				files, _ := payload["files"].([]any)
				if len(files) != 1 {
					t.Fatalf("expected files[1], got %#v", payload)
				}
			},
			checkText: func(t *testing.T, out, errOut string) {
				t.Helper()
				if !strings.Contains(out, "1.0 KB") || !strings.Contains(out, "2025-12-12 14:37") {
					t.Fatalf("unexpected drive ls text: %q", out)
				}
				if strings.Contains(errOut, "--page") {
					t.Fatalf("unexpected next-page hint on gws path: %q", errOut)
				}
			},
			wantTextArgvOK: true,
		},
		{
			name:     "drive get",
			cmd:      &DriveGetCmd{},
			args:     []string{"fileId123"},
			track:    "drive",
			wantArgv: []string{"drive files get", "fileId123"},
			checkJSON: func(t *testing.T, payload map[string]any) {
				t.Helper()
				file, _ := payload["file"].(map[string]any)
				if file == nil || file["id"] != "fileId123" {
					t.Fatalf("expected wrapped file.id=fileId123, got %#v", payload)
				}
			},
		},
		{
			name:     "drive search",
			cmd:      &DriveSearchCmd{},
			args:     []string{"foo"},
			track:    "drive",
			wantArgv: []string{"drive files list", "fullText contains 'foo'"},
			checkJSON: func(t *testing.T, payload map[string]any) {
				t.Helper()
				files, _ := payload["files"].([]any)
				if len(files) != 1 {
					t.Fatalf("expected files[1], got %#v", payload)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/json", func(t *testing.T) {
			argsFile := setupGWS(t)
			var native *bool
			switch tt.track {
			case "drive":
				native = trackNativeDriveService(t)
			case "gmail":
				native = trackNativeGmailService(t)
			default:
				t.Fatalf("unknown track %q", tt.track)
			}

			out := captureStdout(t, func() {
				ctx := newTestUIContext(t, outfmt.Mode{JSON: true})
				if err := runKong(t, tt.cmd, tt.args, ctx, &RootFlags{}); err != nil {
					t.Fatalf("execute: %v", err)
				}
			})
			if *native {
				t.Fatal("gws route must not call native service constructor")
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(out), &payload); err != nil {
				t.Fatalf("json: %v (out=%q)", err, out)
			}
			tt.checkJSON(t, payload)
			assertArgvContains(t, readGWSArgs(t, argsFile), tt.wantArgv...)
		})

		if tt.checkText == nil {
			continue
		}
		t.Run(tt.name+"/text", func(t *testing.T) {
			argsFile := setupGWS(t)
			var errBuf strings.Builder
			out := captureStdout(t, func() {
				u, err := ui.New(ui.Options{Stdout: os.Stdout, Stderr: &errBuf, Color: "never"})
				if err != nil {
					t.Fatalf("ui.New: %v", err)
				}
				ctx := outfmt.WithMode(ui.WithUI(context.Background(), u), outfmt.Mode{})
				if err := runKong(t, tt.cmd, tt.args, ctx, &RootFlags{}); err != nil {
					t.Fatalf("execute: %v", err)
				}
			})
			tt.checkText(t, out, errBuf.String())
			if tt.wantTextArgvOK {
				assertArgvContains(t, readGWSArgs(t, argsFile), tt.wantArgv...)
			}
		})
	}
}

// --- Flags that must stay native under GOG_BACKEND=gws ---

func TestGWS_KeepsNativeForBoundedFlags(t *testing.T) {
	tests := []struct {
		name string
		cmd  any
		args []string
	}{
		{name: "drive ls --global", cmd: &DriveLsCmd{}, args: []string{"--global"}},
		{name: "drive ls --all", cmd: &DriveLsCmd{}, args: []string{"--all"}},
		{name: "drive search --all", cmd: &DriveSearchCmd{}, args: []string{"foo", "--all"}},
		{name: "drive get --page-count", cmd: &DriveGetCmd{}, args: []string{"fileId123", "--page-count"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsFile := setupGWS(t)
			nativeCalled := trackNativeDriveService(t)
			ctx := newTestUIContext(t, outfmt.Mode{JSON: true})
			_ = runKong(t, tt.cmd, tt.args, ctx, &RootFlags{})
			if !*nativeCalled {
				t.Fatal("expected native drive service for bounded flag")
			}
			if _, err := os.Stat(argsFile); err == nil {
				if raw := readGWSArgs(t, argsFile); raw != "" {
					t.Fatalf("gws must not be invoked for native fallback; argv=%q", raw)
				}
			}
		})
	}
}

// --- Error normalization on gws path ---

func TestGWS_NormalizesProviderError(t *testing.T) {
	setupGWS(t)
	t.Setenv("GOG_GWS_FAKE_MODE", "error")
	trackNativeDriveService(t)

	ctx := newTestUIContext(t, outfmt.Mode{JSON: true})
	err := runKong(t, &DriveGetCmd{}, []string{"fileId123"}, ctx, &RootFlags{})
	var be *BackendError
	if !errors.As(err, &be) || be.Env == nil {
		t.Fatalf("expected BackendError, got: %v", err)
	}
	if be.Env.ErrorCode != "unauthenticated" {
		t.Fatalf("error_code=%q, want unauthenticated", be.Env.ErrorCode)
	}
	if be.Env.Service != "drive" || be.Env.Operation != "get" {
		t.Fatalf("service/operation = %q/%q", be.Env.Service, be.Env.Operation)
	}
}
