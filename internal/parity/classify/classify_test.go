package classify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type fixture struct {
	stdout   []byte
	stderr   []byte
	exitCode int
}

func (f fixture) GetStdout() []byte { return f.stdout }
func (f fixture) GetStderr() []byte { return f.stderr }
func (f fixture) GetExitCode() int  { return f.exitCode }

func TestClassify(t *testing.T) {
	tests := []struct {
		name     string
		fd       fixture
		expected Outcome
	}{
		{
			name:     "exit_code_non_zero",
			fd:       fixture{stdout: []byte("{}"), stderr: []byte("{}"), exitCode: 1},
			expected: OutcomeERROR,
		},
		{
			name:     "exit_code_non_zero_stderr_error",
			fd:       fixture{stdout: []byte("{}"), stderr: []byte(`{"error":{"code":500}}`), exitCode: 1},
			expected: OutcomeERROR,
		},
		{
			name:     "stderr_has_error",
			fd:       fixture{stdout: []byte("{}"), stderr: []byte(`{"error":{"code":401}}`), exitCode: 0},
			expected: OutcomeERROR,
		},
		{
			name:     "stdout_has_error_gws_style",
			fd:       fixture{stdout: []byte(`{"error":{"code":401,"reason":"authError"}}`), stderr: []byte("{}"), exitCode: 0},
			expected: OutcomeERROR,
		},
		{
			name:     "stdout_has_error_with_extra_key",
			fd:       fixture{stdout: []byte(`{"_placeholder":true,"error":{"code":403}}`), stderr: []byte("{}"), exitCode: 0},
			expected: OutcomeERROR,
		},
		{
			name:     "success_no_error",
			fd:       fixture{stdout: []byte(`{"labels":[]}`), stderr: []byte("{}"), exitCode: 0},
			expected: OutcomeOK,
		},
		{
			name:     "success_empty_json_both",
			fd:       fixture{stdout: []byte("{}"), stderr: []byte("{}"), exitCode: 0},
			expected: OutcomeOK,
		},
		{
			name:     "malformed_stdout_no_error_key",
			fd:       fixture{stdout: []byte("not json"), stderr: []byte("{}"), exitCode: 0},
			expected: OutcomeOK,
		},
		{
			name:     "malformed_stderr_no_error_key",
			fd:       fixture{stdout: []byte("{}"), stderr: []byte("not json"), exitCode: 0},
			expected: OutcomeOK,
		},
		{
			name:     "both_stdout_and_stderr_error_prefer_stderr",
			fd:       fixture{stdout: []byte(`{"error":{}}`), stderr: []byte(`{"error":{}}`), exitCode: 0},
			expected: OutcomeERROR,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.fd)
			assert.Equal(t, tt.expected, got)
		})
	}
}
