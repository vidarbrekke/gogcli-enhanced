package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/steipete/gogcli/internal/outfmt"
)

func TestVersionStringVariants(t *testing.T) {
	origVersion, origCommit, origDate := version, commit, date
	t.Cleanup(func() { version, commit, date = origVersion, origCommit, origDate })

	version, commit, date = "v1", "", ""
	if got := VersionString(); got != "v1" {
		t.Fatalf("unexpected: %q", got)
	}
	version, commit, date = "v1", "abc", ""
	if got := VersionString(); got != "v1 (abc)" {
		t.Fatalf("unexpected: %q", got)
	}
	version, commit, date = "v1", "", "2025-01-01"
	if got := VersionString(); got != "v1 (2025-01-01)" {
		t.Fatalf("unexpected: %q", got)
	}
	version, commit, date = "v1", "abc", "2025-01-01"
	if got := VersionString(); got != "v1 (abc 2025-01-01)" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestVersionCmd_JSON(t *testing.T) {
	origVersion, origCommit, origDate := version, commit, date
	t.Cleanup(func() { version, commit, date = origVersion, origCommit, origDate })
	version, commit, date = "v2", "c1", "d1"

	// No UI in context so stdoutWriter(ctx) falls back to os.Stdout (captureStdout's pipe)
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})

	jsonOut := captureStdout(t, func() {
		if err := runKong(t, &VersionCmd{}, []string{}, ctx, nil); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	var parsed struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Date    string `json:"date"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &parsed); err != nil {
		t.Fatalf("json parse: %v", err)
	}
	if parsed.Version != "v2" || parsed.Commit != "c1" || parsed.Date != "d1" {
		t.Fatalf("unexpected json: %#v", parsed)
	}
}
