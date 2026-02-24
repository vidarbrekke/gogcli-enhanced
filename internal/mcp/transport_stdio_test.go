//nolint:wsl_v5 // concise protocol assertions
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
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
}
