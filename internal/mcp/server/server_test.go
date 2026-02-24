//nolint:wsl_v5 // concise assertions in small tests
package server

import (
	"context"
	"errors"
	"testing"
)

var errBad = errors.New("bad")

func TestExecuteTool_NotFound(t *testing.T) {
	s := New()
	got := s.ExecuteTool(context.Background(), "nope", map[string]any{})
	if got.OK {
		t.Fatal("expected not found error")
	}
	if got.Error == nil || got.Error.Code != "not_found" {
		t.Fatalf("unexpected error: %#v", got.Error)
	}
}

func TestExecuteTool_ErrorNormalization(t *testing.T) {
	s := New()
	s.RegisterTool("x", func(context.Context, map[string]any) (map[string]any, error) {
		return map[string]any{"service": "docs", "operation": "batch", "error_code": "invalid_argument", "message": "bad"}, errBad
	})
	got := s.ExecuteTool(context.Background(), "x", map[string]any{})
	if got.OK {
		t.Fatal("expected error envelope")
	}
	if got.Error == nil || got.Error.Code != "invalid_argument" {
		t.Fatalf("unexpected error: %#v", got.Error)
	}
}
