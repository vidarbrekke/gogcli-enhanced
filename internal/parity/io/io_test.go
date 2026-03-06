package io

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverCases(t *testing.T) {
	root := t.TempDir()

	// Empty root -> no cases
	cases, failures, err := DiscoverCases(root)
	require.NoError(t, err)
	assert.Empty(t, failures)
	assert.Empty(t, cases)

	// Case with no provider dir
	require.NoError(t, os.MkdirAll(filepath.Join(root, "empty-case"), 0o755))
	cases, failures, err = DiscoverCases(root)
	require.NoError(t, err)
	assert.Empty(t, failures)
	assert.Empty(t, cases)

	// Case with provider dir but missing files
	require.NoError(t, os.MkdirAll(filepath.Join(root, "incomplete", "gws"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "incomplete", "gws", "stdout.json"), []byte("{}"), 0o644))
	cases, failures, err = DiscoverCases(root)
	require.NoError(t, err)
	assert.Empty(t, failures)
	assert.Empty(t, cases)

	// Complete fixture
	require.NoError(t, os.MkdirAll(filepath.Join(root, "full", "gws"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "full", "gws", "stdout.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "full", "gws", "stderr.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "full", "gws", "exit_code.txt"), []byte("0\n"), 0o644))
	cases, failures, err = DiscoverCases(root)
	require.NoError(t, err)
	assert.Empty(t, failures)
	assert.Equal(t, []string{"full"}, cases)

	// Multiple cases
	require.NoError(t, os.MkdirAll(filepath.Join(root, "second", "native"), 0o755))
	for _, f := range []string{"stdout.json", "stderr.json", "exit_code.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "second", "native", f), []byte("{}"), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "second", "native", "exit_code.txt"), []byte("0"), 0o644))
	cases, failures, err = DiscoverCases(root)
	require.NoError(t, err)
	assert.Empty(t, failures)
	assert.Len(t, cases, 2)
	assert.Contains(t, cases, "full")
	assert.Contains(t, cases, "second")
}

func TestLoadFixture(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "mycase", "gws"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "mycase", "gws", "stdout.json"), []byte(`{"labels":[]}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "mycase", "gws", "stderr.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "mycase", "gws", "exit_code.txt"), []byte("0"), 0o644))

	fd, err := LoadFixture(root, "mycase", "gws")
	require.NoError(t, err)
	assert.Equal(t, []byte(`{"labels":[]}`), fd.Stdout)
	assert.Equal(t, []byte("{}"), fd.Stderr)
	assert.Equal(t, 0, fd.ExitCode)
}

func TestLoadFixture_ExitCodeNonZero(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "err", "gws"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "err", "gws", "stdout.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "err", "gws", "stderr.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "err", "gws", "exit_code.txt"), []byte("1"), 0o644))

	fd, err := LoadFixture(root, "err", "gws")
	require.NoError(t, err)
	assert.Equal(t, 1, fd.ExitCode)
}

func TestLoadFixture_MissingFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "x", "gws"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "x", "gws", "stdout.json"), []byte("{}"), 0o644))
	// no stderr.json or exit_code.txt

	_, err := LoadFixture(root, "x", "gws")
	assert.Error(t, err)
}

func TestLoadFixture_InvalidExitCode(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "y", "gws"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "y", "gws", "stdout.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "y", "gws", "stderr.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "y", "gws", "exit_code.txt"), []byte("not-a-number"), 0o644))

	_, err := LoadFixture(root, "y", "gws")
	assert.Error(t, err)
}

func TestProvidersForCase(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "multi", "native"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "multi", "gws"), 0o755))
	for _, p := range []string{"native", "gws"} {
		for _, f := range []string{"stdout.json", "stderr.json", "exit_code.txt"} {
			require.NoError(t, os.WriteFile(filepath.Join(root, "multi", p, f), []byte("{}"), 0o644))
		}
		require.NoError(t, os.WriteFile(filepath.Join(root, "multi", p, "exit_code.txt"), []byte("0"), 0o644))
	}

	providers, err := ProvidersForCase(root, "multi")
	require.NoError(t, err)
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "native")
	assert.Contains(t, providers, "gws")
}
