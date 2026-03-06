# docs/merge — Parity, contracts, and drift control

This directory is the **permanent home** for the parity/merge workstream (not a temporary workspace).

**What "merge" means here:** We are **merging** gogcli-enhanced's control plane with optional backends (e.g. `gws`) under strict **parity** and **contract** rules. The folder name reflects that intent. You can read it as: "merge plan + parity specs + contract/drift control."

## Contents

| Path | Purpose |
|------|---------|
| `goldens/` | Canonical fixture JSON (native + gws). See `goldens/README.md`. |
| `schemas/` | Minimal JSON schemas for envelope and command payloads. |
| `commands/` | Command dossiers (gmail-labels-read, drive-read, docs-info-cat, etc.). |
| `discovery-drift-policy.md` | When to pin/capture vs accept+detect; §7 `google_reason` drift-only. |
| `GWS-SAMPLES.md` | gws stdout/stderr samples and capture commands. |
| `CAPTURE-403-RUNBOOK.md` | Maintainer-only 403 golden capture. |
| `command-migration-matrix.md` | Per-command migration and risk. |
| `GWS-VS-GOG-ROUTING.md` | When gws vs gog; how to implement live routing; how to test. |
| `HANDOFF-FOR-REVIEWER.md` | Paste-ready samples for reviewers. |
| `NATIVE-ENVELOPE-SAMPLES.md` | Native envelope examples. |

**Canonical handover:** Repo-root `handover.md`. Do not duplicate full handover content here.

**Naming:** We keep the name `docs/merge/` unless we migrate ≥5 command groups and decide a rename (e.g. `docs/parity/`) reduces confusion. Prefer adding this README over renaming midstream.
