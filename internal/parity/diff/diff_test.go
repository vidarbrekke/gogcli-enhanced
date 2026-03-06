package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaultRules = DiffRules{
	DriftPaths:    []string{"message", "google_reason"},
	LabelsSetByID: []string{"labels"},
}

func TestDiff_Identical(t *testing.T) {
	a := map[string]any{"labels": []any{}}
	breaking, drift := Diff(a, a, defaultRules)
	assert.Empty(t, breaking)
	assert.Empty(t, drift)
}

func TestDiff_Breaking_MissingKey(t *testing.T) {
	a := map[string]any{"labels": []any{}}
	b := map[string]any{}
	breaking, drift := Diff(a, b, defaultRules)
	require.Len(t, breaking, 1)
	assert.Contains(t, breaking[0].Summary, "only in")
	assert.Empty(t, drift)
}

func TestDiff_Drift_Message(t *testing.T) {
	a := map[string]any{"error": map[string]any{"code": float64(401), "message": "Access denied."}}
	b := map[string]any{"error": map[string]any{"code": float64(401), "message": "Unauthorized"}}
	breaking, drift := Diff(a, b, defaultRules)
	assert.Empty(t, breaking)
	require.Len(t, drift, 1)
	assert.Contains(t, drift[0].Path, "message")
}

func TestDiff_Drift_GoogleReason(t *testing.T) {
	a := map[string]any{"error": map[string]any{"code": float64(403), "google_reason": "forbidden"}}
	b := map[string]any{"error": map[string]any{"code": float64(403), "google_reason": "anotherReason"}}
	breaking, drift := Diff(a, b, defaultRules)
	assert.Empty(t, breaking)
	require.Len(t, drift, 1)
	assert.Contains(t, drift[0].Path, "google_reason")
}

func TestDiff_Labels_SetByID_SameSetDifferentOrder(t *testing.T) {
	a := map[string]any{
		"labels": []any{
			map[string]any{"id": "INBOX", "name": "INBOX", "type": "system"},
			map[string]any{"id": "Label_1", "name": "Custom", "type": "user"},
		},
	}
	b := map[string]any{
		"labels": []any{
			map[string]any{"id": "Label_1", "name": "Custom", "type": "user"},
			map[string]any{"id": "INBOX", "name": "INBOX", "type": "system"},
		},
	}
	breaking, drift := Diff(a, b, defaultRules)
	assert.Empty(t, breaking, "order difference must not be breaking")
	assert.Empty(t, drift)
}

func TestDiff_Labels_SetByID_MissingID(t *testing.T) {
	a := map[string]any{
		"labels": []any{
			map[string]any{"id": "INBOX", "name": "INBOX", "type": "system"},
		},
	}
	b := map[string]any{
		"labels": []any{},
	}
	breaking, drift := Diff(a, b, defaultRules)
	require.Len(t, breaking, 1)
	assert.Contains(t, breaking[0].Path, "INBOX")
	assert.Contains(t, breaking[0].Summary, "only in")
	assert.Empty(t, drift)
}

func TestDiff_Labels_SetByID_ContentDiff(t *testing.T) {
	a := map[string]any{
		"labels": []any{
			map[string]any{"id": "INBOX", "name": "Inbox", "type": "system"},
		},
	}
	b := map[string]any{
		"labels": []any{
			map[string]any{"id": "INBOX", "name": "INBOX", "type": "system"},
		},
	}
	breaking, drift := Diff(a, b, defaultRules)
	require.Len(t, breaking, 1)
	assert.Contains(t, breaking[0].Path, "INBOX")
	assert.Empty(t, drift)
}
