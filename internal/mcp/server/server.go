//nolint:wsl_v5 // compact envelope normalization flow
package server

import (
	"context"
	"fmt"
	"strings"
)

type Server struct {
	tools map[string]ToolHandler
}

func New() *Server {
	return &Server{tools: map[string]ToolHandler{}}
}

func (s *Server) RegisterTool(name string, handler ToolHandler) {
	if s == nil || strings.TrimSpace(name) == "" || handler == nil {
		return
	}
	s.tools[strings.TrimSpace(name)] = handler
}

func (s *Server) ExecuteTool(ctx context.Context, name string, input map[string]any) Envelope {
	handler, ok := s.tools[strings.TrimSpace(name)]
	if !ok {
		return Envelope{
			OK: false,
			Error: &ErrorEnvelope{
				Code:    "not_found",
				Message: fmt.Sprintf("tool %q not found", name),
			},
		}
	}
	result, err := handler(ctx, input)
	if err != nil {
		return normalizeError(result, err)
	}
	return Envelope{
		OK:          true,
		Service:     str(result["service"]),
		Operation:   str(result["operation"]),
		OpID:        str(result["opId"]),
		RequestHash: str(result["requestHash"]),
		Result:      result,
	}
}

func str(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func normalizeError(result map[string]any, err error) Envelope {
	code := "api_error"
	message := err.Error()
	details := map[string]any{}
	if result != nil {
		if v := str(result["error_code"]); v != "" {
			code = v
		}
		if v := str(result["message"]); v != "" {
			message = v
		}
		for k, v := range result {
			details[k] = v
		}
	}
	return Envelope{
		OK:        false,
		Service:   str(result["service"]),
		Operation: str(result["operation"]),
		OpID:      str(result["opId"]),
		Error: &ErrorEnvelope{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}
