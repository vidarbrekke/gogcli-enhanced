package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/steipete/gogcli/internal/mcp/server"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func ServeStdio(ctx context.Context, r io.Reader, w io.Writer, s *server.Server) error {
	scanner := bufio.NewScanner(r)
	// Raise max token size above default 64KB so large tools/call payloads don't drop the connection.
	const maxScanTokenSize = 10 * 1024 * 1024
	const maxInFlight = 8
	const maxQueue = 256
	scanner.Buffer(make([]byte, 0, 64*1024), maxScanTokenSize)
	writeCh := make(chan rpcResponse, maxQueue)
	sem := make(chan struct{}, maxInFlight)
	var wg sync.WaitGroup
	done := make(chan struct{})
	var writeErr error
	var writeOnce sync.Once

	go func() {
		defer close(done)
		for resp := range writeCh {
			if err := writeRPC(w, resp); err != nil {
				writeOnce.Do(func() { writeErr = err })
				return
			}
		}
	}()

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if parseWriteErr := writeRPC(w, rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}}); parseWriteErr != nil {
				return parseWriteErr
			}
			continue
		}
		if req.ID == nil {
			continue
		}
		if req.Method == "tools/call" {
			select {
			case sem <- struct{}{}:
			default:
				writeCh <- rpcResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: map[string]any{
						"isError": true,
						"content": []map[string]any{
							{"type": "text", "text": "{\"ok\":false,\"error\":{\"code\":\"resource_exhausted\",\"message\":\"too many in-flight requests\"}}"},
						},
					},
				}
				continue
			}
			wg.Add(1)
			go func(reqCopy rpcRequest) {
				defer wg.Done()
				defer func() { <-sem }()
				writeCh <- handleRPC(ctx, s, reqCopy)
			}(req)
			continue
		}
		writeCh <- handleRPC(ctx, s, req)
	}
	wg.Wait()
	close(writeCh)
	<-done
	if writeErr != nil {
		return writeErr
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan stdio input: %w", err)
	}
	return nil
}

func handleRPC(ctx context.Context, s *server.Server, req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		mcpDebugLog(req.Method, nil)
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]any{
					"name":    "gog-agentic",
					"version": "v1",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
			},
		}
	case "tools/list":
		specs := s.ListToolSpecs()
		tools := make([]map[string]any, 0, len(specs))
		for _, spec := range specs {
			tool := map[string]any{
				"name":        spec.Name,
				"description": spec.Description,
				"inputSchema": spec.InputSchema,
			}
			if spec.Tier != "" {
				tool["tier"] = spec.Tier
			}
			if spec.Version != "" {
				tool["version"] = spec.Version
			}
			if spec.PolicyClass != "" {
				tool["policyClass"] = spec.PolicyClass
			}
			tools = append(tools, tool)
		}
		mcpDebugLog(req.Method, map[string]int{"tools": len(tools)})
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"tools": tools},
		}
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid params"}}
		}
		env := s.ExecuteTool(ctx, params.Name, params.Arguments)
		payload, _ := json.Marshal(env)
		// Send only content (agent loop consumes this). structuredContent is omitted to avoid
		// sending the same payload twice; OpenClaw uses content for reasoning.
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"isError": !env.OK,
				"content": []map[string]any{
					{"type": "text", "text": string(payload)},
				},
			},
		}
	default:
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "method not found"}}
	}
}

func writeRPC(w io.Writer, resp rpcResponse) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal rpc response: %w", err)
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("write rpc response: %w", err)
	}
	return nil
}

var mcpDebugLogMu sync.Mutex

// mcpDebugLog writes one line to the file in GOG_MCP_DEBUG_LOG when set (temporary debugging).
// Format: ISO8601 method [key=value ...]
func mcpDebugLog(method string, extra map[string]int) {
	path := os.Getenv("GOG_MCP_DEBUG_LOG")
	if path == "" {
		return
	}
	mcpDebugLogMu.Lock()
	defer mcpDebugLogMu.Unlock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	line := time.Now().UTC().Format(time.RFC3339) + " " + method
	for k, v := range extra {
		line += fmt.Sprintf(" %s=%d", k, v)
	}
	_, _ = f.WriteString(line + "\n")
	_ = f.Close()
}
