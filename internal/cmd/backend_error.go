package cmd

import (
	"fmt"
	"strings"

	"github.com/steipete/gogcli/internal/parity/normalize"
)

const errCodeInvalidArgument = "invalid_argument"

// BackendError is a normalized gws (or other backend) error for JSON envelopes.
// It implements error and jsonErrorFieldsProvider so formatJSONErrorEnvelope emits error_code and http_status.
type BackendError struct {
	Env *normalize.CanonicalEnvelope
}

func (e *BackendError) Error() string {
	if e == nil || e.Env == nil {
		return "backend error"
	}
	return fmt.Sprintf("%s (http %d)", e.Env.ErrorCode, e.Env.HTTPStatus)
}

// JSONErrorFields returns fields for the JSON error envelope. Do not parse message for semantics.
func (e *BackendError) JSONErrorFields() map[string]any {
	if e == nil || e.Env == nil {
		return map[string]any{"error_code": "unknown"}
	}
	m := map[string]any{
		"error_code":  e.Env.ErrorCode,
		"http_status": e.Env.HTTPStatus,
		"service":     e.Env.Service,
		"operation":   e.Env.Operation,
		"resource_id": e.Env.ResourceID,
	}
	if strings.TrimSpace(e.Env.GoogleReason) != "" {
		m["google_reason"] = e.Env.GoogleReason
	}
	return m
}

// backendErrorExitCode maps normalized error_code to exit code for BackendError.
func backendErrorExitCode(code string) int {
	switch code {
	case "unauthenticated":
		return exitCodeAuthRequired
	case "not_found":
		return exitCodeNotFound
	case "permission_denied":
		return exitCodePermissionDenied
	case "resource_exhausted":
		return exitCodeRateLimited
	case errCodeInvalidArgument:
		return 2
	default:
		return 1
	}
}
