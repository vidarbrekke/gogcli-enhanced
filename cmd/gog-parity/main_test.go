package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var parityBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gog-parity-testbin-")
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(dir)
	parityBin = filepath.Join(dir, "gog-parity")
	if out, err := exec.Command("go", "build", "-o", parityBin, ".").Output(); err != nil {
		os.Stderr.Write(out)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// runParity runs the parity binary with the given fixtures and schemas roots; returns stdout, stderr, exit code.
func runParity(t *testing.T, fixturesRoot, schemasRoot string) (stdout, stderr []byte, exitCode int) {
	t.Helper()
	cmd := exec.Command(parityBin, "-fixtures", fixturesRoot, "-schemas", schemasRoot, "-provider", "gws")
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	stdout = outBuf.Bytes()
	stderr = errBuf.Bytes()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("run parity: %v", err)
		}
	}
	return stdout, stderr, exitCode
}

// parseReportCases parses report JSON and returns the cases slice for assertions.
func parseReportCases(t *testing.T, stdout []byte) []CaseResult {
	t.Helper()
	var report Report
	if err := json.Unmarshal(stdout, &report); err != nil {
		t.Fatalf("parse report: %v", err)
	}
	return report.Cases
}

func TestReportPath(t *testing.T) {
	t.Run("preserves json pointer style suffix", func(t *testing.T) {
		got := reportPath("gmail-labels-list", "/labels/id:INBOX")
		want := "gmail-labels-list/labels/id:INBOX"
		if got != want {
			t.Fatalf("reportPath() = %q, want %q", got, want)
		}
	})

		t.Run("normalizes non-pointer suffix", func(t *testing.T) {
			got := reportPath("gmail-labels-list", "labels")
			want := "gmail-labels-list/labels"
			if got != want {
				t.Fatalf("reportPath() = %q, want %q", got, want)
			}
		})
}

// TestHardGatedError_UnnormalizablePayload: hard-gated ERROR case with payload that cannot be normalized → FAIL_RUNNER, exit 1.
func TestHardGatedError_UnnormalizablePayload(t *testing.T) {
	fixtures := t.TempDir()
	schemas := t.TempDir()
	caseDir := filepath.Join(fixtures, "gmail-labels-401-unauthenticated", "gws")
	if err := os.MkdirAll(caseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caseDir, "exit_code.txt"), []byte("1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caseDir, "stdout.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caseDir, "stderr.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, exitCode := runParity(t, fixtures, schemas)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit, got 0")
	}
	cases := parseReportCases(t, stdout)
	var found *CaseResult
	for i := range cases {
		if cases[i].Case == "gmail-labels-401-unauthenticated" {
			found = &cases[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("report missing case gmail-labels-401-unauthenticated")
	}
	if found.Outcome != "FAIL_RUNNER" {
		t.Fatalf("outcome = %q, want FAIL_RUNNER", found.Outcome)
	}
	if len(found.RunnerFailures) == 0 {
		t.Fatalf("expected RunnerFailures")
	}
	hasNorm := false
	for _, r := range found.RunnerFailures {
		if r.Kind == "NORMALIZE_ERROR" {
			hasNorm = true
			break
		}
	}
	if !hasNorm {
		t.Fatalf("expected NORMALIZE_ERROR in RunnerFailures, got %v", found.RunnerFailures)
	}
}

// TestDiscovery_UnreadableDir: unreadable case dir during discovery → DISCOVERY_ERROR in report, exit 1.
func TestDiscovery_UnreadableDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 000 on dir not reliable on Windows")
	}
	fixtures := t.TempDir()
	schemas := t.TempDir()
	badCase := filepath.Join(fixtures, "unreadable-case")
	if err := os.MkdirAll(badCase, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(badCase, 0o000); err != nil {
		t.Skip("chmod 000 not supported")
	}
	defer os.Chmod(badCase, 0o755)

	stdout, _, exitCode := runParity(t, fixtures, schemas)
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when case dir is unreadable, got 0")
	}
	cases := parseReportCases(t, stdout)
	var found *CaseResult
	for i := range cases {
		if cases[i].Case == "unreadable-case" {
			found = &cases[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("report should include discovery failure for unreadable-case")
	}
	if found.Outcome != "FAIL_RUNNER" {
		t.Fatalf("outcome = %q, want FAIL_RUNNER", found.Outcome)
	}
	hasDiscovery := false
	for _, r := range found.RunnerFailures {
		if r.Kind == "DISCOVERY_ERROR" {
			hasDiscovery = true
			break
		}
	}
	if !hasDiscovery {
		t.Fatalf("expected DISCOVERY_ERROR in RunnerFailures, got %v", found.RunnerFailures)
	}
}

// TestPlaceholder_SkippedNoFailures: case with PLACEHOLDER.txt → SKIPPED_PLACEHOLDER, exit 0.
func TestPlaceholder_SkippedNoFailures(t *testing.T) {
	fixtures := t.TempDir()
	schemas := t.TempDir()
	providerDir := filepath.Join(fixtures, "placeholder-case", "gws")
	if err := os.MkdirAll(providerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"stdout.json", "stderr.json", "exit_code.txt"} {
		if err := os.WriteFile(filepath.Join(providerDir, f), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(providerDir, "exit_code.txt"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "PLACEHOLDER.txt"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, exitCode := runParity(t, fixtures, schemas)
	if exitCode != 0 {
		t.Fatalf("expected exit 0 for placeholder case, got %d", exitCode)
	}
	cases := parseReportCases(t, stdout)
	var found *CaseResult
	for i := range cases {
		if cases[i].Case == "placeholder-case" {
			found = &cases[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("report missing case placeholder-case")
	}
	if found.Outcome != "SKIPPED_PLACEHOLDER" {
		t.Fatalf("outcome = %q, want SKIPPED_PLACEHOLDER", found.Outcome)
	}
	if len(found.RunnerFailures) != 0 {
		t.Fatalf("placeholder must not have RunnerFailures, got %v", found.RunnerFailures)
	}
}

// TestDeterministic_ReportByteIdentical: two runs with same fixtures produce identical JSON report.
func TestDeterministic_ReportByteIdentical(t *testing.T) {
	fixtures := t.TempDir()
	schemas := t.TempDir()
	providerDir := filepath.Join(fixtures, "det-case", "gws")
	if err := os.MkdirAll(providerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "stdout.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "stderr.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "exit_code.txt"), []byte("0"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout1, _, _ := runParity(t, fixtures, schemas)
	stdout2, _, _ := runParity(t, fixtures, schemas)
	if !bytes.Equal(stdout1, stdout2) {
		t.Errorf("two parity runs produced different output (first %d bytes, second %d bytes)", len(stdout1), len(stdout2))
	}
}
