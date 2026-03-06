package normalize

import (
	"encoding/json"
)

// InvocationCtx carries case-derived context (service, operation, resource id) for canonical envelope.
type InvocationCtx struct {
	Service    string
	Operation  string
	ResourceID string
}

// CanonicalEnvelope is the normalized error representation.
// Contractual (gate CI): ErrorCode, HTTPStatus.
// Drift-only (never gate): GoogleReason, message text; Service, Operation, ResourceID are context from invocation.
// Defined here in code; no separate envelope schema file unless a consumer needs schema-based validation.
type CanonicalEnvelope struct {
	ErrorCode    string `json:"error_code"`
	HTTPStatus   int    `json:"http_status"`
	GoogleReason string `json:"google_reason,omitempty"`
	Service      string `json:"service,omitempty"`
	Operation    string `json:"operation,omitempty"`
	ResourceID   string `json:"resource_id,omitempty"`
}

// GWS error shape: {"error": {"code": 404, "message": "...", "reason": "notFound"}}
type gwsError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Reason  string `json:"reason"`
	} `json:"error"`
}

var statusToErrorCode = map[int]string{
	400: "invalid_argument",
	401: "unauthenticated",
	403: "permission_denied",
	404: "not_found",
	429: "resource_exhausted",
}

// ErrorCodeFromStatus maps HTTP status to contractual error_code. Returns "unknown" for unmapped status.
func ErrorCodeFromStatus(status int) string {
	if s, ok := statusToErrorCode[status]; ok {
		return s
	}
	return "unknown"
}

// NormalizeError parses stdout or stderr for gws error shape and returns a canonical envelope.
// Prefer stderr if both contain an error; otherwise use whichever has the error.
// If no error object is found, returns nil, false.
func NormalizeError(stdout, stderr []byte, ctx InvocationCtx) (*CanonicalEnvelope, bool) {
	var errOut *CanonicalEnvelope
	// Prefer stderr (native convention), then stdout (gws often puts error on stdout).
	for _, raw := range [][]byte{stderr, stdout} {
		var g gwsError
		if json.Unmarshal(raw, &g) != nil {
			continue
		}
		if g.Error.Code == 0 {
			continue
		}
		code := g.Error.Code
		errOut = &CanonicalEnvelope{
			ErrorCode:    ErrorCodeFromStatus(code),
			HTTPStatus:   code,
			GoogleReason: g.Error.Reason,
			Service:      ctx.Service,
			Operation:    ctx.Operation,
			ResourceID:   ctx.ResourceID,
		}
		return errOut, true
	}
	return nil, false
}
