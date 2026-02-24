package server

import "context"

type ToolHandler func(ctx context.Context, input map[string]any) (map[string]any, error)

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
