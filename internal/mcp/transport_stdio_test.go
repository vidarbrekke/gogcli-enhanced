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
