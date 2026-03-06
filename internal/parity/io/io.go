package io

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// FixtureData holds raw stdout, stderr, and exit code for a single case/provider.
type FixtureData struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// GetStdout returns stdout bytes (implements classify.FixtureData).
func (f FixtureData) GetStdout() []byte { return f.Stdout }

// GetStderr returns stderr bytes (implements classify.FixtureData).
func (f FixtureData) GetStderr() []byte { return f.Stderr }

// GetExitCode returns exit code (implements classify.FixtureData).
func (f FixtureData) GetExitCode() int { return f.ExitCode }

// DiscoveryFailure records a case directory that could not be read during discovery.
type DiscoveryFailure struct {
	CaseDir string // case directory name (e.g. "gmail-labels-list")
	Err     error
}

// DiscoverCases returns case names under root and any discovery failures (e.g. unreadable case dirs).
// A case is a directory that contains at least one provider subdirectory with stdout.json, stderr.json, and exit_code.txt.
// Caller must fail CI if len(failures) > 0.
func DiscoverCases(root string) (cases []string, failures []DiscoveryFailure, err error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, fmt.Errorf("read goldens root: %w", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		casePath := filepath.Join(root, name)
		providers, readErr := os.ReadDir(casePath)
		if readErr != nil {
			failures = append(failures, DiscoveryFailure{CaseDir: name, Err: readErr})
			continue
		}
		for _, p := range providers {
			if !p.IsDir() {
				continue
			}
			providerPath := filepath.Join(casePath, p.Name())
			if fixturePresent(providerPath) {
				cases = append(cases, name)
				break
			}
		}
	}
	return cases, failures, nil
}

func fixturePresent(providerPath string) bool {
	for _, f := range []string{"stdout.json", "stderr.json", "exit_code.txt"} {
		if _, err := os.Stat(filepath.Join(providerPath, f)); err != nil {
			return false
		}
	}
	return true
}

// LoadFixture reads stdout.json, stderr.json, and exit_code.txt for the given case and provider.
// Returns error if any required file is missing or unreadable.
func LoadFixture(root, caseName, provider string) (FixtureData, error) {
	dir := filepath.Join(root, caseName, provider)
	stdoutPath := filepath.Clean(filepath.Join(dir, "stdout.json"))
	stderrPath := filepath.Clean(filepath.Join(dir, "stderr.json"))
	exitPath := filepath.Clean(filepath.Join(dir, "exit_code.txt"))

	stdout, err := os.ReadFile(stdoutPath) // #nosec G304 -- path from Join(root, caseName, provider), not user input
	if err != nil {
		return FixtureData{}, fmt.Errorf("read stdout.json: %w", err)
	}
	stderr, err := os.ReadFile(stderrPath) // #nosec G304 -- path from Join(root, caseName, provider)
	if err != nil {
		return FixtureData{}, fmt.Errorf("read stderr.json: %w", err)
	}
	rawExit, err := os.ReadFile(exitPath) // #nosec G304 -- path from Join(root, caseName, provider)
	if err != nil {
		return FixtureData{}, fmt.Errorf("read exit_code.txt: %w", err)
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(rawExit)))
	if err != nil {
		return FixtureData{}, fmt.Errorf("parse exit_code.txt: %w", err)
	}

	return FixtureData{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: code,
	}, nil
}

// ProvidersForCase returns provider names that have a complete fixture set for the case.
func ProvidersForCase(root, caseName string) ([]string, error) {
	casePath := filepath.Join(root, caseName)
	entries, err := os.ReadDir(casePath)
	if err != nil {
		return nil, fmt.Errorf("read case dir: %w", err)
	}

	var providers []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if fixturePresent(filepath.Join(casePath, e.Name())) {
			providers = append(providers, e.Name())
		}
	}
	sort.Strings(providers)
	return providers, nil
}

// IsPlaceholder returns true if a provider fixture is intentionally a placeholder (e.g. 403 until captured).
func IsPlaceholder(root, caseName, provider string) bool {
	p := filepath.Join(root, caseName, provider, "PLACEHOLDER.txt")
	_, err := os.Stat(p)
	return err == nil
}
