package main

import "testing"

func TestReportPath(t *testing.T) {
	t.Run("preserves json pointer style suffix", func(t *testing.T) {
		got := reportPath("gmail-labels-list", "/labels/id:INBOX")
		want := "gmail-labels-list/labels/id:INBOX"
		if got != want {
			t.Fatalf("reportPath() = %q, want %q", got, want)
		}
	})

	t.Run("normalizes non-pointer suffix", func(t *testing.T) {
		got := reportPath("gmail-labels-list", "labels")
		want := "gmail-labels-list/labels"
		if got != want {
			t.Fatalf("reportPath() = %q, want %q", got, want)
		}
	})
}

func TestIsHardGatedBreakingPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "gmail-labels-401/anything", want: true},
		{path: "gmail-labels-get-not-found", want: true},
		{path: "gmail-labels-get-not-found/error_code", want: true},
		{path: "gmail-labels-403-forbidden/error_code", want: false},
		{path: "gmail-labels-list/labels/id:INBOX", want: false},
	}

	for _, tt := range tests {
		if got := isHardGatedBreakingPath(tt.path); got != tt.want {
			t.Fatalf("isHardGatedBreakingPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
