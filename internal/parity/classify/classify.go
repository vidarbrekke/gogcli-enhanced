package classify

import (
	"encoding/json"
)

// Outcome is the classified result for a fixture (OK or ERROR).
type Outcome string

const (
	OutcomeOK    Outcome = "OK"
	OutcomeERROR Outcome = "ERROR"
)

// FixtureData is the minimal interface needed for classification (avoids importing parity/io in tests with raw data).
type FixtureData interface {
	GetStdout() []byte
	GetStderr() []byte
	GetExitCode() int
}

// Classify determines outcome from exit code and presence of top-level "error" in stdout or stderr JSON.
// ERROR when: exit_code != 0, or stderr JSON has top-level "error", or stdout JSON has top-level "error" (gws often errors on stdout).
func Classify(fd FixtureData) Outcome {
	if fd.GetExitCode() != 0 {
		return OutcomeERROR
	}

	if hasTopLevelError(fd.GetStderr()) {
		return OutcomeERROR
	}

	if hasTopLevelError(fd.GetStdout()) {
		return OutcomeERROR
	}
	return OutcomeOK
}

func hasTopLevelError(raw []byte) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	_, ok := m["error"]
	return ok
}
