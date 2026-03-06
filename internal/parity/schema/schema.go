package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SchemaViolation describes a single validation failure.
type SchemaViolation struct {
	Path string
	Msg  string
}

// LoadSchema reads a JSON Schema file from the schemas root. name is the filename (e.g. "gmail-labels-list.json").
func LoadSchema(schemasRoot, name string) ([]byte, error) {
	path := filepath.Join(schemasRoot, name)
	data, err := os.ReadFile(path) // #nosec G304 -- path from filepath.Join(schemasRoot, name), not user input
	if err != nil {
		return nil, fmt.Errorf("read schema %s: %w", name, err)
	}
	return data, nil
}

// Validate checks the document against the schema. Returns violations (required fields, types).
// Supports minimal JSON Schema: type object/array, required, properties, items, $ref to #/$defs/XXX.
func Validate(docJSON, schemaJSON []byte) ([]SchemaViolation, error) {
	var doc, schema map[string]any
	if err := json.Unmarshal(docJSON, &doc); err != nil {
		return nil, fmt.Errorf("document is not valid JSON: %w", err)
	}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, fmt.Errorf("schema is not valid JSON: %w", err)
	}
	defs, _ := schema["$defs"].(map[string]any)
	var out []SchemaViolation
	validateNode("/", doc, schema, defs, &out)
	return out, nil
}

func validateNode(path string, doc any, nodeSchema, defs map[string]any, out *[]SchemaViolation) {
	if nodeSchema == nil {
		return
	}
	switch doc := doc.(type) {
	case map[string]any:
		if typeStr, _ := nodeSchema["type"].(string); typeStr != "" && typeStr != "object" {
			*out = append(*out, SchemaViolation{Path: path, Msg: "expected type " + typeStr + ", got object"})
			return
		}
		if req, ok := nodeSchema["required"].([]any); ok {
			for _, r := range req {
				key, _ := r.(string)
				if _, has := doc[key]; !has {
					*out = append(*out, SchemaViolation{Path: path, Msg: "missing required property: " + key})
				}
			}
		}
		props, _ := nodeSchema["properties"].(map[string]any)
		for key, propSchema := range props {
			propSchemaM, _ := propSchema.(map[string]any)
			childPath := path + "/" + key
			if val, has := doc[key]; has {
				resolved := resolveRef(propSchemaM, defs)
				if resolved != nil {
					validateNode(childPath, val, resolved, defs, out)
				}
			}
		}
	case []any:
		if typeStr, _ := nodeSchema["type"].(string); typeStr != "" && typeStr != "array" {
			*out = append(*out, SchemaViolation{Path: path, Msg: "expected type " + typeStr + ", got array"})
			return
		}
		items, _ := nodeSchema["items"].(map[string]any)
		resolved := resolveRef(items, defs)
		for i, item := range doc {
			validateNode(fmt.Sprintf("%s/%d", path, i), item, resolved, defs, out)
		}
	default:
		// primitive; type check if schema expects specific type
		typeStr, _ := nodeSchema["type"].(string)
		if typeStr == "" {
			return
		}
		switch typeStr {
		case "string":
			if _, ok := doc.(string); !ok {
				*out = append(*out, SchemaViolation{Path: path, Msg: "expected string"})
			}
		case "integer", "number":
			if _, ok := doc.(float64); !ok {
				if _, ok := doc.(int); !ok {
					*out = append(*out, SchemaViolation{Path: path, Msg: "expected number"})
				}
			}
		case "boolean":
			if _, ok := doc.(bool); !ok {
				*out = append(*out, SchemaViolation{Path: path, Msg: "expected boolean"})
			}
		}
	}
}

func resolveRef(n map[string]any, defs map[string]any) map[string]any {
	if n == nil {
		return nil
	}
	ref, ok := n["$ref"].(string)
	if !ok {
		return n
	}
	// only #/$defs/XXX
	if len(ref) > 8 && ref[:8] == "#/$defs/" {
		name := ref[8:]
		if d, ok := defs[name].(map[string]any); ok {
			return d
		}
	}
	return n
}
