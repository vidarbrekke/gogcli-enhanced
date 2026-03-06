// cmd/gog-parity/main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type Report struct {
	Provider              string         `json:"provider"`
	ProviderVersion       string         `json:"provider_version,omitempty"`
	DiscoverySnapshotHash string         `json:"discovery_snapshot_hash,omitempty"`
	Breaking              []DiffEntry    `json:"breaking,omitempty"`
	Drift                 []DiffEntry    `json:"drift,omitempty"`
	NormalizationsApplied []string       `json:"normalizations_applied,omitempty"`
	Meta                  map[string]any `json:"meta,omitempty"`
}

type DiffEntry struct {
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

func main() {
	fixturesRoot := flag.String("fixtures", "", "path to goldens root (required)")
	schemasRoot := flag.String("schemas", "", "path to schemas root (required)")
	provider := flag.String("provider", "gws", "provider name (native|gws)")
	flag.Parse()

	if *fixturesRoot == "" || *schemasRoot == "" {
		fmt.Fprintln(os.Stderr, "missing --fixtures or --schemas")
		os.Exit(2)
	}

	// TODO (DRY/YAGNI): implement loaders + validators + diff. Keep it minimal:
	// 1) enumerate cases under fixturesRoot
	// 2) for each case/provider: load stdout/stderr/exit_code
	// 3) classify outcome (stderr.error OR stdout.error OR exit_code!=0)
	// 4) normalize provider output to canonical
	// 5) validate against envelope schema + command schema
	// 6) diff against native canonical (if present) and produce report

	report := Report{
		Provider: *provider,
		Meta: map[string]any{
			"fixtures": *fixturesRoot,
			"schemas":  *schemasRoot,
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)
}
