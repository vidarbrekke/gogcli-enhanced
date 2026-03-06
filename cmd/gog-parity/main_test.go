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
