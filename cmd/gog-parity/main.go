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
	Case           string          `json:"case"`
	Outcome        string          `json:"outcome"` // OK|ERROR|FAIL_RUNNER|SKIPPED_PLACEHOLDER
	RunnerFailures []RunnerFailure `json:"runner_failures,omitempty"`
	Breaking       []DiffEntry     `json:"breaking,omitempty"`
	Drift          []DiffEntry     `json:"drift,omitempty"`
}

type RunnerFailure struct {
	Kind   string `json:"kind"` // IO_ERROR|JSON_PARSE_ERROR|SCHEMA_ERROR|NORMALIZE_ERROR|DISCOVERY_ERROR|SETUP_ERROR
	Detail string `json:"detail"`
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

func main() {
	fixturesRoot := flag.String("fixtures", "", "path to goldens root (required)")
	schemasRoot := flag.String("schemas", "", "path to schemas root (required)")
	provider := flag.String("provider", "gws", "provider name (e.g. native, gws)")
	flag.Parse()

	if *fixturesRoot == "" || *schemasRoot == "" {
		fmt.Fprintln(os.Stderr, "missing --fixtures or --schemas")
		os.Exit(2)
	}

	report := Report{
		Provider: *provider,
		Meta: map[string]any{
			"fixtures": *fixturesRoot,
			"schemas":  *schemasRoot,
		},
	}
	cases, discoveryFailures, err := io.DiscoverCases(*fixturesRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "discover cases: %v\n", err)
		os.Exit(1)
	}
	for _, f := range discoveryFailures {
		report.Cases = append(report.Cases, CaseResult{
			Case:           f.CaseDir,
			Outcome:        "FAIL_RUNNER",
			RunnerFailures: []RunnerFailure{{Kind: "DISCOVERY_ERROR", Detail: f.Err.Error()}},
		})
	}
	sort.Strings(cases)

	for _, caseName := range cases {
		providers, err := io.ProvidersForCase(*fixturesRoot, caseName)
		if err != nil {
			report.Cases = append(report.Cases, CaseResult{
				Case:           caseName,
				Outcome:        "FAIL_RUNNER",
				RunnerFailures: []RunnerFailure{{Kind: "SETUP_ERROR", Detail: err.Error()}},
			})
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
			report.Cases = append(report.Cases, CaseResult{
				Case:           caseName,
				Outcome:        "FAIL_RUNNER",
				RunnerFailures: []RunnerFailure{{Kind: "IO_ERROR", Detail: err.Error()}},
			})
			continue
		}
		if io.IsPlaceholder(*fixturesRoot, caseName, *provider) {
			report.Cases = append(report.Cases, CaseResult{Case: caseName, Outcome: "SKIPPED_PLACEHOLDER"})
			continue
		}
		outcome := classify.Classify(fd)
		cr := CaseResult{Case: caseName, Outcome: string(outcome)}

		if outcome == classify.OutcomeERROR {
			ctx := invocationCtxForCase(caseName)
			env, normalizeOk := normalize.NormalizeError(fd.Stdout, fd.Stderr, ctx)
			if expectedErrorForCase(caseName).hardGated {
				if !normalizeOk {
					cr.Outcome = "FAIL_RUNNER"
					cr.RunnerFailures = append(cr.RunnerFailures, RunnerFailure{
						Kind:   "NORMALIZE_ERROR",
						Detail: "hard-gated error case: could not parse error envelope from stdout/stderr",
					})
				} else {
					report.NormalizationsApplied = append(report.NormalizationsApplied,
						caseName+": "+env.ErrorCode+" (http "+fmt.Sprint(env.HTTPStatus)+")")
					exp := expectedErrorForCase(caseName)
					if env.HTTPStatus != exp.HTTPStatus || env.ErrorCode != exp.ErrorCode {
						cr.Breaking = append(cr.Breaking, DiffEntry{
							Path:    "/error",
							Summary: fmt.Sprintf("expected %s/%d got %s/%d", exp.ErrorCode, exp.HTTPStatus, env.ErrorCode, env.HTTPStatus),
						})
						report.Breaking = append(report.Breaking, DiffEntry{
							Path:    reportPath(caseName, "/error"),
							Summary: cr.Breaking[len(cr.Breaking)-1].Summary,
						})
					}
				}
			} else if normalizeOk {
				report.NormalizationsApplied = append(report.NormalizationsApplied,
					caseName+": "+env.ErrorCode+" (http "+fmt.Sprint(env.HTTPStatus)+")")
			}
		} else {
			schemaFile := schemaFileForCase(caseName)
			if schemaFile != "" {
				schemaJSON, err := schema.LoadSchema(*schemasRoot, schemaFile)
				if err != nil {
					cr.RunnerFailures = append(cr.RunnerFailures, RunnerFailure{Kind: "SCHEMA_ERROR", Detail: err.Error()})
				} else {
					violations, verr := schema.Validate(fd.Stdout, schemaJSON)
					if verr != nil {
						cr.RunnerFailures = append(cr.RunnerFailures, RunnerFailure{Kind: "SCHEMA_ERROR", Detail: verr.Error()})
					} else if len(violations) > 0 {
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
				if err1 := json.Unmarshal(fd.Stdout, &gwsJSON); err1 != nil {
					cr.RunnerFailures = append(cr.RunnerFailures, RunnerFailure{Kind: "JSON_PARSE_ERROR", Detail: "provider stdout: " + err1.Error()})
				} else if err2 := json.Unmarshal(nativeFD.Stdout, &nativeJSON); err2 != nil {
					cr.RunnerFailures = append(cr.RunnerFailures, RunnerFailure{Kind: "JSON_PARSE_ERROR", Detail: "native stdout: " + err2.Error()})
				} else {
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

	// Deterministic report: sort slices that are built from iteration order.
	sort.Strings(report.NormalizationsApplied)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode report: %v\n", err)
		os.Exit(1)
	}
	hardGated := map[string]bool{
		"gmail-labels-401-unauthenticated": true,
		"gmail-labels-get-not-found":       true,
	}
	for _, c := range report.Cases {
		if len(c.RunnerFailures) > 0 {
			fmt.Fprintf(os.Stderr, "parity: runner failure in case %s\n", c.Case)
			os.Exit(1)
		}
		if hardGated[c.Case] && len(c.Breaking) > 0 {
			fmt.Fprintf(os.Stderr, "parity: breaking diff in hard-gated case %s\n", c.Case)
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
	case "drive-ls":
		ctx.Service = "drive"
		ctx.Operation = "ls"
	}
	return ctx
}

type caseErrorPolicy struct {
	hardGated  bool
	HTTPStatus int
	ErrorCode  string
}

func expectedErrorForCase(caseName string) caseErrorPolicy {
	switch caseName {
	case "gmail-labels-401-unauthenticated":
		return caseErrorPolicy{hardGated: true, HTTPStatus: 401, ErrorCode: "unauthenticated"}
	case "gmail-labels-get-not-found":
		return caseErrorPolicy{hardGated: true, HTTPStatus: 404, ErrorCode: "not_found"}
	default:
		return caseErrorPolicy{}
	}
}
