package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_GmailLabelsList_Pass(t *testing.T) {
	doc := []byte(`{"labels":[{"id":"INBOX","name":"INBOX","type":"system"}]}`)
	schema := []byte(`{
	  "type":"object",
	  "required":["labels"],
	  "properties":{"labels":{"type":"array","items":{"$ref":"#/$defs/label"}}},
	  "$defs":{"label":{"type":"object","required":["id","name","type"],"properties":{"id":{"type":"string"},"name":{"type":"string"},"type":{"type":"string"}}}}
	}`)
	violations, err := Validate(doc, schema)
	require.NoError(t, err)
	assert.Empty(t, violations)
}

func TestValidate_GmailLabelsList_MissingRequired(t *testing.T) {
	doc := []byte(`{}`)
	schema := []byte(`{"type":"object","required":["labels"],"properties":{"labels":{"type":"array"}}}`)
	violations, err := Validate(doc, schema)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Contains(t, violations[0].Msg, "missing required")
}

func TestValidate_GmailLabelsList_ItemMissingRequired(t *testing.T) {
	doc := []byte(`{"labels":[{"id":"X","name":"X"}]}`)
	schema := []byte(`{
	  "type":"object",
	  "required":["labels"],
	  "properties":{"labels":{"type":"array","items":{"$ref":"#/$defs/label"}}},
	  "$defs":{"label":{"type":"object","required":["id","name","type"],"properties":{}}}
	}`)
	violations, err := Validate(doc, schema)
	require.NoError(t, err)
	require.Len(t, violations, 1)
	assert.Contains(t, violations[0].Msg, "missing required")
	assert.Contains(t, violations[0].Path, "labels")
}

func TestValidate_InvalidJSON(t *testing.T) {
	_, err := Validate([]byte("not json"), []byte("{}"))
	assert.Error(t, err)
}

func TestLoadSchema(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "foo.json"), []byte(`{"type":"object"}`), 0o644))
	data, err := LoadSchema(root, "foo.json")
	require.NoError(t, err)
	assert.Equal(t, `{"type":"object"}`, string(data))
}

func TestLoadSchema_NotFound(t *testing.T) {
	_, err := LoadSchema(t.TempDir(), "missing.json")
	assert.Error(t, err)
}
