package normalize

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorCodeFromStatus(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{400, "invalid_argument"},
		{401, "unauthenticated"},
		{403, "permission_denied"},
		{404, "not_found"},
		{429, "resource_exhausted"},
		{500, "unknown"},
		{0, "unknown"},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := ErrorCodeFromStatus(tt.status)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeError_GWS_401(t *testing.T) {
	stdout := []byte(`{"error":{"code":401,"message":"Access denied.","reason":"authError"}}`)
	stderr := []byte("{}")
	ctx := InvocationCtx{Service: "gmail", Operation: "labels list"}

	env, ok := NormalizeError(stdout, stderr, ctx)
	require.True(t, ok)
	assert.Equal(t, "unauthenticated", env.ErrorCode)
	assert.Equal(t, 401, env.HTTPStatus)
	assert.Equal(t, "authError", env.GoogleReason)
	assert.Equal(t, "gmail", env.Service)
	assert.Equal(t, "labels list", env.Operation)
}

func TestNormalizeError_GWS_404(t *testing.T) {
	stdout := []byte(`{"error":{"code":404,"message":"Requested entity was not found.","reason":"notFound"}}`)
	stderr := []byte("{}")
	ctx := InvocationCtx{Service: "gmail", Operation: "labels get", ResourceID: "Label_DoesNotExist_123"}

	env, ok := NormalizeError(stdout, stderr, ctx)
	require.True(t, ok)
	assert.Equal(t, "not_found", env.ErrorCode)
	assert.Equal(t, 404, env.HTTPStatus)
	assert.Equal(t, "notFound", env.GoogleReason)
	assert.Equal(t, "Label_DoesNotExist_123", env.ResourceID)
}

func TestNormalizeError_PreferStderr(t *testing.T) {
	stdout := []byte(`{"error":{"code":401}}`)
	stderr := []byte(`{"error":{"code":403,"reason":"forbidden"}}`)
	ctx := InvocationCtx{}

	env, ok := NormalizeError(stdout, stderr, ctx)
	require.True(t, ok)
	assert.Equal(t, 403, env.HTTPStatus)
	assert.Equal(t, "permission_denied", env.ErrorCode)
	assert.Equal(t, "forbidden", env.GoogleReason)
}

func TestNormalizeError_NoError(t *testing.T) {
	stdout := []byte(`{"labels":[]}`)
	stderr := []byte("{}")

	env, ok := NormalizeError(stdout, stderr, InvocationCtx{})
	assert.False(t, ok)
	assert.Nil(t, env)
}

func TestNormalizeError_GoogleReason_DriftOnly(t *testing.T) {
	// google_reason is stored but never used for breaking; just ensure it's present
	stdout := []byte(`{"error":{"code":403,"message":"Insufficient Permission","reason":"forbidden"}}`)
	stderr := []byte("{}")
	env, ok := NormalizeError(stdout, stderr, InvocationCtx{})
	require.True(t, ok)
	assert.Equal(t, "forbidden", env.GoogleReason)
	// Serialize and ensure JSON has the field (drift-only, not used for gating)
	b, err := json.Marshal(env)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Contains(t, m, "google_reason")
}
