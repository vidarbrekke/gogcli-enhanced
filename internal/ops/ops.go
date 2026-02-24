package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

type Envelope struct {
	OK          bool           `json:"ok"`
	OpID        string         `json:"op_id,omitempty"`
	Service     string         `json:"service,omitempty"`
	Operation   string         `json:"operation,omitempty"`
	RequestHash string         `json:"request_hash,omitempty"`
	DryRun      bool           `json:"dry_run,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
	Error       *ErrorEnvelope `json:"error,omitempty"`
}

type ErrorEnvelope struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type AgentOperation struct {
	OpID      string         `json:"op_id"`
	Service   string         `json:"service"`
	Operation string         `json:"operation"`
	Request   map[string]any `json:"request"`
	CreatedAt time.Time      `json:"created_at"`
}

func HashRequest(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	sum := sha256.Sum256(b)

	return hex.EncodeToString(sum[:]), nil
}
