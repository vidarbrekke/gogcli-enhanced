package server

import "context"

type ToolHandler func(ctx context.Context, input map[string]any) (map[string]any, error)

type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
	Tier        string         `json:"tier,omitempty"`
	Version     string         `json:"version,omitempty"`
	PolicyClass string         `json:"policy_class,omitempty"`
	Handler     ToolHandler    `json:"-"`
}

type Envelope struct {
	OK          bool           `json:"ok"`
	Service     string         `json:"service,omitempty"`
	Operation   string         `json:"operation,omitempty"`
	OpID        string         `json:"op_id,omitempty"`
	RequestHash string         `json:"request_hash,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
	Error       *ErrorEnvelope `json:"error,omitempty"`
}

type ErrorEnvelope struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
