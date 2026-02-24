//nolint:wsl_v5 // protocol switch intentionally compact
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

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
	scanner.Buffer(make([]byte, 0, 64*1024), maxScanTokenSize)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if writeErr := writeRPC(w, rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}}); writeErr != nil {
				return writeErr
			}
			continue
		}
		resp := handleRPC(ctx, s, req)
		if req.ID == nil {
			continue
		}
		if err := writeRPC(w, resp); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan stdio input: %w", err)
	}
	return nil
}

func handleRPC(ctx context.Context, s *server.Server, req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]any{
					"name":    "gogcli-mcp",
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
			tools = append(tools, map[string]any{
				"name":        spec.Name,
				"description": spec.Description,
				"inputSchema": spec.InputSchema,
			})
		}
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
		return rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"isError":           !env.OK,
				"structuredContent": env,
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
