// Command gog-parity runs fixture-only parity checks: load goldens, classify outcomes,
// and emit a JSON report. No live API calls.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/steipete/gogcli/internal/parity/classify"
	"github.com/steipete/gogcli/internal/parity/diff"
	"github.com/steipete/gogcli/internal/parity/io"
	"github.com/steipete/gogcli/internal/parity/normalize"
	"github.com/steipete/gogcli/internal/parity/schema"
)

type Report struct {
	Provider              string         `json:"provider"`
	ProviderVersion       string         `json:"provider_version,omitempty"`
	DiscoverySnapshotHash string         `json:"discovery_snapshot_hash,omitempty"`
	Cases                 []CaseResult   `json:"cases,omitempty"`
	Breaking              []DiffEntry    `json:"breaking,omitempty"`
	Drift                 []DiffEntry    `json:"drift,omitempty"`
	NormalizationsApplied []string       `json:"normalizations_applied,omitempty"`
	Meta                  map[string]any `json:"meta,omitempty"`
}

type CaseResult struct {
	Case     string      `json:"case"`
	Outcome  string      `json:"outcome"`
	Breaking []DiffEntry `json:"breaking,omitempty"`
	Drift    []DiffEntry `json:"drift,omitempty"`
}

type DiffEntry struct {
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

func reportPath(caseName, path string) string {
	if strings.HasPrefix(path, "/") {
		return caseName + path
	}

	return caseName + "/" + path
}

func isHardGatedBreakingPath(path string) bool {
	return strings.HasPrefix(path, "gmail-labels-401/") ||
		path == "gmail-labels-get-not-found" ||
		strings.HasPrefix(path, "gmail-labels-get-not-found/")
}

func main() {
	fixturesRoot := flag.String("fixtures", "", "path to goldens root (required)")
	schemasRoot := flag.String("schemas", "", "path to schemas root (required)")
	provider := flag.String("provider", "gws", "provider name (e.g. native, gws)")
	flag.Parse()

	if *fixturesRoot == "" || *schemasRoot == "" {
		fmt.Fprintln(os.Stderr, "missing --fixtures or --schemas")
		os.Exit(2)
	}

	cases, err := io.DiscoverCases(*fixturesRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "discover cases: %v\n", err)
		os.Exit(1)
	}
	sort.Strings(cases)

	report := Report{
		Provider: *provider,
		Meta: map[string]any{
			"fixtures": *fixturesRoot,
			"schemas":  *schemasRoot,
		},
	}

	for _, caseName := range cases {
		providers, err := io.ProvidersForCase(*fixturesRoot, caseName)
		if err != nil {
			report.Cases = append(report.Cases, CaseResult{Case: caseName, Outcome: "ERROR"})
			continue
		}
		hasProvider := false
		for _, p := range providers {
			if p == *provider {
				hasProvider = true
				break
			}
		}
		if !hasProvider {
			continue
		}

		fd, err := io.LoadFixture(*fixturesRoot, caseName, *provider)
		if err != nil {
			report.Cases = append(report.Cases, CaseResult{Case: caseName, Outcome: "ERROR"})
			continue
		}
		outcome := classify.Classify(fd)
		cr := CaseResult{Case: caseName, Outcome: string(outcome)}

		if outcome == classify.OutcomeERROR {
			ctx := invocationCtxForCase(caseName)
			if env, ok := normalize.NormalizeError(fd.Stdout, fd.Stderr, ctx); ok {
				report.NormalizationsApplied = append(report.NormalizationsApplied,
					caseName+": "+env.ErrorCode+" (http "+fmt.Sprint(env.HTTPStatus)+")")
			}
		} else {
			schemaFile := schemaFileForCase(caseName)
			if schemaFile != "" {
				schemaJSON, err := schema.LoadSchema(*schemasRoot, schemaFile)
				if err == nil {
					violations, err := schema.Validate(fd.Stdout, schemaJSON)
					if err == nil && len(violations) > 0 {
						for _, v := range violations {
							cr.Breaking = append(cr.Breaking, DiffEntry{Path: v.Path, Summary: v.Msg})
							report.Breaking = append(report.Breaking, DiffEntry{
								Path:    reportPath(caseName, v.Path),
								Summary: v.Msg,
							})
						}
					}
				}
			}
			// Diff against native baseline if present
			if nativeFD, err := io.LoadFixture(*fixturesRoot, caseName, "native"); err == nil {
				var gwsJSON, nativeJSON map[string]any
				if json.Unmarshal(fd.Stdout, &gwsJSON) == nil && json.Unmarshal(nativeFD.Stdout, &nativeJSON) == nil {
					diffRules := diff.DiffRules{
						DriftPaths:    []string{"message", "google_reason"},
						LabelsSetByID: []string{"labels"},
					}
					breakingD, driftD := diff.Diff(gwsJSON, nativeJSON, diffRules)
					for _, e := range breakingD {
						cr.Breaking = append(cr.Breaking, DiffEntry{Path: e.Path, Summary: e.Summary})
						report.Breaking = append(report.Breaking, DiffEntry{
							Path:    reportPath(caseName, e.Path),
							Summary: e.Summary,
						})
					}
					for _, e := range driftD {
						cr.Drift = append(cr.Drift, DiffEntry{Path: e.Path, Summary: e.Summary})
						report.Drift = append(report.Drift, DiffEntry{
							Path:    reportPath(caseName, e.Path),
							Summary: e.Summary,
						})
					}
				}
			}
		}
		report.Cases = append(report.Cases, cr)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode report: %v\n", err)
		os.Exit(1)
	}
	// Hard gate: 401 and 404 cases must have no breaking diffs. 403 remains soft until real golden is committed.
	for _, e := range report.Breaking {
		if isHardGatedBreakingPath(e.Path) {
			fmt.Fprintf(os.Stderr, "parity: breaking diff in hard-gated case: %s\n", e.Path)
			os.Exit(1)
		}
	}
}

func schemaFileForCase(caseName string) string {
	switch caseName {
	case "gmail-labels-list":
		return "gmail-labels-list.json"
	case "gmail-labels-get":
		return "gmail-labels-get.json"
	case "drive-ls":
		return "drive-ls.json"
	default:
		return ""
	}
}

func invocationCtxForCase(caseName string) normalize.InvocationCtx {
	ctx := normalize.InvocationCtx{}
	switch caseName {
	case "gmail-labels-401-unauthenticated", "gmail-labels-403-forbidden":
		ctx.Service = "gmail"
		ctx.Operation = "labels list"
	case "gmail-labels-get-not-found":
		ctx.Service = "gmail"
		ctx.Operation = "labels get"
	}
	return ctx
}
