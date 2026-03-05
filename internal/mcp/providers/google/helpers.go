package google

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/steipete/gogcli/internal/mcp/server"
)

func injectRequireRevision(requests map[string]any, revID string) map[string]any {
	revID = strings.TrimSpace(revID)
	if revID == "" {
		return requests
	}
	out := make(map[string]any, len(requests)+1)
	for k, v := range requests {
		out[k] = v
	}
	wc, _ := out["writeControl"].(map[string]any)
	if wc == nil {
		wc = make(map[string]any)
		out["writeControl"] = wc
	}
	wc["requiredRevisionId"] = revID
	return out
}

func writeTempJSON(v any) (string, error) {
	f, err := os.CreateTemp("", "gog-mcp-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	if _, err := f.Write(b); err != nil {
		return "", fmt.Errorf("write temp file: %w", err)
	}
	return f.Name(), nil
}

func maybeOpIDArgs(input map[string]any) []string {
	if opID := strings.TrimSpace(asString(input["opId"])); opID != "" {
		return []string{"--op-id", opID}
	}
	return nil
}

func maybeAccountArgs(input map[string]any) []string {
	acct := strings.TrimSpace(asString(input["account"]))
	if acct != "" && !strings.HasPrefix(acct, "-") {
		return []string{"--account", acct}
	}
	return nil
}

func cleanArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if s := strings.TrimSpace(a); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func asInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return i, true
		}
	}
	return 0, false
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

func trimFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func policyArgs(input map[string]any) []string {
	args := make([]string, 0, 6)
	if timeoutMs, ok := asInt(input["timeoutMs"]); ok && timeoutMs > 0 {
		if timeoutMs > 15*60*1000 {
			timeoutMs = 15 * 60 * 1000
		}
		args = append(args, "--request-timeout", fmt.Sprintf("%dms", timeoutMs))
	}
	if retries, ok := asInt(input["retries"]); ok {
		if retries < -1 {
			retries = -1
		}
		if retries > 10 {
			retries = 10
		}
		args = append(args, "--retries", strconv.FormatInt(retries, 10))
	}
	if retryBackoffMs, ok := asInt(input["retryBackoffMs"]); ok && retryBackoffMs > 0 {
		if retryBackoffMs > 30000 {
			retryBackoffMs = 30000
		}
		args = append(args, "--retry-backoff", fmt.Sprintf("%dms", retryBackoffMs))
	}
	return args
}

// validateSheetColumn returns the 0-based column index or an error envelope for invalid_argument.
func validateSheetColumn(input map[string]any, key, svc, op string) (int64, map[string]any, error) {
	n, ok := asInt(input[key])
	if !ok || n < 0 {
		return 0, map[string]any{
			"service": svc, "operation": op,
			"error_code": server.ErrorCodeInvalidArgument,
			"message":    fmt.Sprintf("missing or invalid %s (must be non-negative integer)", key),
		}, errMissingIndex
	}
	return n, nil, nil
}

// validateRequiredIntSlice returns the slice of 0-based indices or an error envelope.
func validateRequiredIntSlice(input map[string]any, key, svc, op string) ([]int64, map[string]any, error) {
	raw, ok := input[key].([]any)
	if !ok || len(raw) == 0 {
		return nil, map[string]any{
			"service": svc, "operation": op,
			"error_code": server.ErrorCodeInvalidArgument,
			"message":    fmt.Sprintf("missing or empty %s (must be non-empty array of integers)", key),
		}, errMissingIndex
	}
	out := make([]int64, 0, len(raw))
	for i, x := range raw {
		n, ok := asInt(x)
		if !ok {
			return nil, map[string]any{
				"service": svc, "operation": op,
				"error_code": server.ErrorCodeInvalidArgument,
				"message":    fmt.Sprintf("%s[%d] must be an integer", key, i),
			}, errMissingIndex
		}
		out = append(out, n)
	}
	return out, nil, nil
}

var allowedSheetOps = map[string]bool{"eq": true, "contains": true, "gt": true, "lt": true}

// validateSheetOp returns nil if input[key] is in allowed set, else an error envelope.
func validateSheetOp(input map[string]any, key, svc, op string) (map[string]any, error) {
	v := strings.TrimSpace(strings.ToLower(asString(input[key])))
	if v == "" {
		return map[string]any{
			"service": svc, "operation": op,
			"error_code": server.ErrorCodeInvalidArgument,
			"message":    fmt.Sprintf("missing %s (use eq, contains, gt, lt)", key),
		}, errMissingIndex
	}
	if !allowedSheetOps[v] {
		return map[string]any{
			"service": svc, "operation": op,
			"error_code": server.ErrorCodeInvalidArgument,
			"message":    fmt.Sprintf("invalid %s %q (use eq, contains, gt, lt)", key, v),
		}, errMissingIndex
	}
	return map[string]any{}, nil
}
