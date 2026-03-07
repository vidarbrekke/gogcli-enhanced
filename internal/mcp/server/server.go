package server

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

type Server struct {
	tools map[string]ToolSpec
}

func New() *Server {
	return &Server{tools: map[string]ToolSpec{}}
}

func (s *Server) RegisterTool(name string, handler ToolHandler) {
	if s == nil || strings.TrimSpace(name) == "" || handler == nil {
		return
	}
	s.tools[strings.TrimSpace(name)] = ToolSpec{
		Name:    strings.TrimSpace(name),
		Handler: handler,
	}
}

func (s *Server) RegisterToolSpec(spec ToolSpec) {
	if s == nil || strings.TrimSpace(spec.Name) == "" || spec.Handler == nil {
		return
	}
	spec.Name = strings.TrimSpace(spec.Name)
	s.tools[spec.Name] = spec
}

func (s *Server) ListToolSpecs() []ToolSpec {
	if s == nil {
		return nil
	}
	names := make([]string, 0, len(s.tools))
	for name := range s.tools {
		names = append(names, name)
	}
	slices.Sort(names)
	out := make([]ToolSpec, 0, len(names))
	for _, name := range names {
		out = append(out, s.tools[name])
	}
	return out
}

func (s *Server) ExecuteTool(ctx context.Context, name string, input map[string]any) Envelope {
	spec, ok := s.tools[strings.TrimSpace(name)]
	if !ok {
		return Envelope{
			OK: false,
			Error: &ErrorEnvelope{
				Code:    ErrorCodeNotFound,
				Message: fmt.Sprintf("tool %q not found", name),
			},
		}
	}
	result, err := spec.Handler(ctx, input)
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
	code := ErrorCodeAPI
	message := err.Error()
	details := map[string]any{}
	var service, operation, opID, requestHash string
	if result != nil {
		service = str(result["service"])
		operation = str(result["operation"])
		opID = str(result["opId"])
		requestHash = str(result["requestHash"])
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
		OK:          false,
		Service:     service,
		Operation:   operation,
		OpID:        opID,
		RequestHash: requestHash,
		Error: &ErrorEnvelope{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}
