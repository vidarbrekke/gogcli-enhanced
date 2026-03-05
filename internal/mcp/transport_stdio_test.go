//nolint:wsl_v5 // concise protocol assertions
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/steipete/gogcli/internal/mcp/server"
)

func TestServeStdio_InitializeAndToolsList(t *testing.T) {
	s := NewGoogleServer(func(args []string) (string, string, error) {
		return "{}", "", nil
	})
	in := bytes.NewBufferString("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n")
	var out bytes.Buffer
	if err := ServeStdio(context.Background(), in, &out, s); err != nil {
		t.Fatalf("ServeStdio: %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(out.Bytes()), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(lines))
	}
	var initResp map[string]any
	if err := json.Unmarshal(lines[0], &initResp); err != nil {
		t.Fatalf("parse init response: %v", err)
	}
	if initResp["error"] != nil {
		t.Fatalf("unexpected init error: %#v", initResp["error"])
	}
	var listResp map[string]any
	if err := json.Unmarshal(lines[1], &listResp); err != nil {
		t.Fatalf("parse list response: %v", err)
	}
	result, ok := listResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing list result: %#v", listResp)
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("expected tools list, got %#v", result["tools"])
	}
	first, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected tool shape: %#v", tools[0])
	}
	if _, ok := first["version"]; !ok {
		t.Fatalf("expected version metadata in tool listing: %#v", first)
	}
	if _, ok := first["policyClass"]; !ok {
		t.Fatalf("expected policyClass metadata in tool listing: %#v", first)
	}
}

func TestServeStdio_ToolsCall_ConcurrentResponses(t *testing.T) {
	s := server.New()
	s.RegisterTool("echo", func(_ context.Context, input map[string]any) (map[string]any, error) {
		return map[string]any{
			"service":   "test",
			"operation": "echo",
			"value":     input["value"],
		}, nil
	})
	var inLines []string
	for i := 1; i <= 20; i++ {
		inLines = append(inLines, fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"id\":%d,\"method\":\"tools/call\",\"params\":{\"name\":\"echo\",\"arguments\":{\"value\":%d}}}", i, i))
	}
	in := bytes.NewBufferString(strings.Join(inLines, "\n") + "\n")
	var out bytes.Buffer
	if err := ServeStdio(context.Background(), in, &out, s); err != nil {
		t.Fatalf("ServeStdio: %v", err)
	}
	lines := bytes.Split(bytes.TrimSpace(out.Bytes()), []byte("\n"))
	if len(lines) != 20 {
		t.Fatalf("expected 20 responses, got %d", len(lines))
	}
}

// TestServeStdio_ToolsCall_ResultShape asserts tools/call response has content only (no structuredContent).
// Downstream (e.g. OpenClaw) uses content for agent reasoning; we omit structuredContent to avoid sending the envelope twice.
func TestServeStdio_ToolsCall_ResultShape(t *testing.T) {
	s := server.New()
	s.RegisterTool("add", func(_ context.Context, input map[string]any) (map[string]any, error) {
		a, _ := input["a"].(float64)
		b, _ := input["b"].(float64)
		return map[string]any{"service": "test", "operation": "add", "sum": a + b}, nil
	})
	in := bytes.NewBufferString("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"add\",\"arguments\":{\"a\":2,\"b\":3}}}\n")
	var out bytes.Buffer
	if err := ServeStdio(context.Background(), in, &out, s); err != nil {
		t.Fatalf("ServeStdio: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %#v", resp["result"])
	}
	// Must have content (agent consumes this).
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("result must have non-empty content array, got %#v", result["content"])
	}
	part, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] must be object: %#v", content[0])
	}
	if part["type"] != "text" {
		t.Fatalf("content[0].type must be text, got %q", part["type"])
	}
	text, _ := part["text"].(string)
	if text == "" {
		t.Fatalf("content[0].text must be non-empty")
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("content text must be valid JSON: %v", err)
	}
	if parsed["ok"] != true {
		t.Fatalf("expected ok true, got %v", parsed["ok"])
	}
	if parsed["result"] == nil {
		t.Fatalf("expected result in envelope")
	}
	// Must NOT have structuredContent (we omit it for token efficiency).
	if _, has := result["structuredContent"]; has {
		t.Fatalf("result must not contain structuredContent (downstream uses content only); got %#v", result["structuredContent"])
	}
}
