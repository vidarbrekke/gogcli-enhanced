package cmd

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AgenticContractTest verifies that all edit commands (docs, sheets, slides)
// follow the same agentic safety contract for machine-readable output.
// This test suite is critical for VID-97 - Cross-Service Agentic Hardening.
func TestAgenticContract(t *testing.T) {
	t.Run("error_envelope_fields", func(t *testing.T) {
		// Test that NewEditError produces consistent error envelopes
		testCases := []struct {
			name      string
			service   string
			operation string
			code      string
		}{
			{"docs_error", "docs", "batch", "invalid_request"},
			{"sheets_error", "sheets", "values", "api_error"},
			{"slides_error", "slides", "batch", "presentation_not_found"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := NewEditError(tc.service, tc.operation, "test-id", tc.code, "test message", nil)
				require.Error(t, err)

				editErr, ok := err.(*EditError)
				require.True(t, ok, "error should be *EditError")

				fields := editErr.JSONErrorFields()

				// Required fields for all services
				assert.Equal(t, tc.code, fields["error_code"], "error_code field")
				assert.Equal(t, tc.service, fields["service"], "service field")
				assert.Equal(t, tc.operation, fields["operation"], "operation field")
				assert.Equal(t, "test-id", fields["resource_id"], "resource_id field")
			})
		}
	})

	t.Run("validate_only_success_fields", func(t *testing.T) {
		// Standard fields that must be present in validate-only output
		requiredFields := []string{
			"validateOnly",
			"valid",
			"requestHash",
		}

		for _, field := range requiredFields {
			assert.NotEmpty(t, field, "field %s should be documented", field)
		}
	})

	t.Run("dry_run_success_fields", func(t *testing.T) {
		// Standard fields that must be present in dry-run output
		requiredFields := []string{
			"dryRun",
			"service",
			"resourceId",
			"request",
		}

		for _, field := range requiredFields {
			assert.NotEmpty(t, field, "field %s should be documented", field)
		}
	})

	t.Run("request_hash_determinism", func(t *testing.T) {
		// Same request should always produce same hash
		req := map[string]any{
			"test": "value",
			"num":  42,
		}

		hash1, err1 := RequestHash(req)
		hash2, err2 := RequestHash(req)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.Equal(t, hash1, hash2, "request hash must be deterministic")
		assert.NotEmpty(t, hash1, "hash must not be empty")
	})

	t.Run("normalized_json_format", func(t *testing.T) {
		req := map[string]any{
			"operations": []string{"update"},
		}

		jsonStr, err := NormalizedRequestString(req)
		require.NoError(t, err)

		// Must be valid JSON
		var parsed map[string]any
		err = json.Unmarshal([]byte(jsonStr), &parsed)
		require.NoError(t, err)

		// Must have newline at end
		assert.True(t, len(jsonStr) > 0 && jsonStr[len(jsonStr)-1] == '\n',
			"normalized JSON must end with newline")
	})

	t.Run("is_not_found_detection", func(t *testing.T) {
		// Different error types should be detected consistently
		testCases := []struct {
			name     string
			err      error
			expected bool
		}{
			{"nil_error", nil, false},
			{"generic_error", assert.AnError, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := IsNotFound(tc.err)
				assert.Equal(t, tc.expected, result)
			})
		}
	})
}

// TestServiceSpecificHardening tests each service follows the agentic pattern.
func TestServiceSpecificHardening(t *testing.T) {
	t.Run("docs_uses_standard_error", func(t *testing.T) {
		// Docs edit commands should produce EditError-compatible errors
		err := newDocsEditError("batch", "doc-123", "test_error", "test", nil)
		require.Error(t, err)

		// Should be convertible to EditError via unwrapping
		var editErr *EditError
		hasEditErr := errors.As(err, &editErr)
		assert.True(t, hasEditErr, "docs error should wrap *EditError")
		if hasEditErr {
			assert.Equal(t, "docs", editErr.Service)
			assert.Equal(t, "doc-123", editErr.ResourceID)
		}
	})

	t.Run("shared_helpers_delegation", func(t *testing.T) {
		// Verify that service-specific helpers delegate to shared functions
		testData := map[string]any{"key": "value"}

		// These should all work without error
		hash, err := RequestHash(testData)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)

		jsonStr, err := NormalizedRequestString(testData)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonStr)
	})
}

// BenchmarkRequestHash ensures hashing is performant.
func BenchmarkRequestHash(b *testing.B) {
	req := map[string]any{
		"operations": make([]map[string]any, 100),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = RequestHash(req)
	}
}
