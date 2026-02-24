# MCP Tooling Baseline

This repo now includes a minimal internal operations layer in `internal/ops` to support MCP-oriented plan/execute workflows without coupling tool contracts to CLI flag parsing.

## Current baseline

- Shared envelope and hash primitives:
  - `internal/ops/ops.go`
- Existing CLI agentic primitives continue to be the execution backbone:
  - `internal/cmd/edit_helpers.go`
  - `internal/cmd/root.go`

## Planned MCP-facing operations (initial scope)

- Docs: plan/execute batch update
- Slides: plan/execute batch update
- Sheets: plan/execute values/batch updates
- Drive: resolve/ensure-folder/plan-execute file operations

## Contract principles

- JSON-only responses for tools
- Stable `error_code` mapping
- `opId` echoed in success and error envelopes
- Deterministic `requestHash` for replayability
