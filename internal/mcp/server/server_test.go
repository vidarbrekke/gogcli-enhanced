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

func TestExecuteTool_ErrorNormalization_PropagatesRequestHash(t *testing.T) {
	s := New()
	s.RegisterTool("x", func(context.Context, map[string]any) (map[string]any, error) {
		return map[string]any{
			"service":     "docs",
			"operation":   "batch",
			"requestHash": "abc",
			"error_code":  "invalid_argument",
			"message":     "bad",
		}, errBad
	})
	got := s.ExecuteTool(context.Background(), "x", map[string]any{})
	if got.OK {
		t.Fatal("expected error envelope")
	}
	if got.RequestHash != "abc" {
		t.Fatalf("expected requestHash to propagate, got %q", got.RequestHash)
	}
}

func TestListToolSpecs(t *testing.T) {
	s := New()
	s.RegisterToolSpec(ToolSpec{Name: "b.tool", Handler: func(context.Context, map[string]any) (map[string]any, error) { return map[string]any{}, nil }})
	s.RegisterToolSpec(ToolSpec{Name: "a.tool", Handler: func(context.Context, map[string]any) (map[string]any, error) { return map[string]any{}, nil }})
	specs := s.ListToolSpecs()
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}
	if specs[0].Name != "a.tool" || specs[1].Name != "b.tool" {
		t.Fatalf("unexpected ordering: %#v", specs)
	}
}
