package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/steipete/gogcli/internal/outfmt"
)

func TestTimeNowCmd_JSON(t *testing.T) {
	// No UI in context so stdoutWriter(ctx) falls back to os.Stdout (captureStdout's pipe)
	ctx := outfmt.WithMode(context.Background(), outfmt.Mode{JSON: true})

	out := captureStdout(t, func() {
		if err := runKong(t, &TimeNowCmd{}, []string{"--timezone", "UTC"}, ctx, &RootFlags{}); err != nil {
			t.Fatalf("runKong: %v", err)
		}
	})

	var parsed struct {
		Timezone    string `json:"timezone"`
		UTCOffset   string `json:"utc_offset"`
		CurrentTime string `json:"current_time"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json parse: %v", err)
	}
	if parsed.Timezone != "UTC" {
		t.Fatalf("unexpected timezone: %q", parsed.Timezone)
	}
	if parsed.UTCOffset != "+00:00" {
		t.Fatalf("unexpected offset: %q", parsed.UTCOffset)
	}
	if parsed.CurrentTime == "" {
		t.Fatalf("expected current_time")
	}
}

func TestTimeNowCmd_InvalidTimezone(t *testing.T) {
	if err := runKong(t, &TimeNowCmd{}, []string{"--timezone", "Nope/Zone"}, context.Background(), &RootFlags{}); err == nil {
		t.Fatalf("expected error")
	}
}
