package gws

import (
	"os"
	"testing"
)

func TestHasTopLevelError_stdout(t *testing.T) {
	stdout := []byte(`{"error":{"code":401,"message":"x","reason":"authError"}}`)

	stderr := []byte(`{}`)
	if !HasTopLevelError(stdout, stderr) {
		t.Error("expected true when stdout has error")
	}
}

func TestHasTopLevelError_stderr(t *testing.T) {
	stdout := []byte(`{}`)

	stderr := []byte(`{"error":{"code":404}}`)
	if !HasTopLevelError(stdout, stderr) {
		t.Error("expected true when stderr has error")
	}
}

func TestHasTopLevelError_noError(t *testing.T) {
	stdout := []byte(`{"labels":[]}`)
	stderr := []byte(`{}`)

	if HasTopLevelError(stdout, stderr) {
		t.Error("expected false when no error key")
	}
}

func TestHasTopLevelError_invalidJSON(t *testing.T) {
	stdout := []byte(`not json`)
	stderr := []byte(``)

	if HasTopLevelError(stdout, stderr) {
		t.Error("expected false when not valid JSON")
	}
}

func TestPath_default(t *testing.T) {
	orig := os.Getenv("GOG_GWS_PATH")

	os.Unsetenv("GOG_GWS_PATH")

	defer func() {
		if orig != "" {
			os.Setenv("GOG_GWS_PATH", orig)
		}
	}()

	if got := Path(); got != "gws" {
		t.Errorf("Path() = %q, want gws", got)
	}
}
