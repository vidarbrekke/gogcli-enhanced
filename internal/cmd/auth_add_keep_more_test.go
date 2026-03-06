package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/gogcli/internal/config"
	"github.com/steipete/gogcli/internal/googleauth"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/secrets"
)

func TestAuthAddCmd_JSON_More(t *testing.T) {
	origOpen := openSecretsStore
	origAuth := authorizeGoogle
	origKeychain := ensureKeychainAccess
	origFetch := fetchAuthorizedEmail
	t.Cleanup(func() {
		openSecretsStore = origOpen
		authorizeGoogle = origAuth
		ensureKeychainAccess = origKeychain
		fetchAuthorizedEmail = origFetch
	})

	store := newMemSecretsStore()
	openSecretsStore = func() (secrets.Store, error) { return store, nil }
	authorizeGoogle = func(ctx context.Context, opts googleauth.AuthorizeOptions) (string, error) {
		if len(opts.Services) == 0 {
			t.Fatalf("expected services")
		}
		return "rt", nil
	}
	fetchAuthorizedEmail = func(context.Context, string, string, []string, time.Duration) (string, error) {
		return "a@b.com", nil
	}
	ensureKeychainAccess = func(bool) error { return nil }

	cmd := &AuthAddCmd{Email: "a@b.com", ServicesCSV: "gmail,drive"}
	out := captureStdout(t, func() {
		ctx := newTestUIContext(t, outfmt.Mode{JSON: true})
		if err := cmd.Run(ctx, &RootFlags{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
	})
	if !strings.Contains(out, "\"stored\"") {
		t.Fatalf("unexpected output: %q", out)
	}
	if _, err := store.GetToken(config.DefaultClientName, "a@b.com"); err != nil {
		t.Fatalf("expected token stored: %v", err)
	}
}

func TestAuthKeepCmd_JSON_More(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))

	payload := map[string]any{
		"type":         "service_account",
		"client_email": "svc@example.com",
		"private_key":  "key",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	keyPath := filepath.Join(t.TempDir(), "sa.json")
	if writeErr := os.WriteFile(keyPath, data, 0o600); writeErr != nil {
		t.Fatalf("write key: %v", writeErr)
	}

	cmd := &AuthKeepCmd{Email: "user@example.com", Key: keyPath}
	out := captureStdout(t, func() {
		ctx := newTestUIContext(t, outfmt.Mode{JSON: true})
		if runErr := cmd.Run(ctx, &RootFlags{}); runErr != nil {
			t.Fatalf("Run: %v", runErr)
		}
	})
	if !strings.Contains(out, "\"stored\"") || !strings.Contains(out, "\"path\"") {
		t.Fatalf("unexpected output: %q", out)
	}
}
