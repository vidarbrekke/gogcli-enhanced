Let’s do a TypeScript + Go Codebase Audit

**Audit completed.** See **[docs/TS-Go-AUDIT-RESULTS.md](docs/TS-Go-AUDIT-RESULTS.md)** for Findings, Code Fixes (TS worker typing), Action Plan, and Tooling. TypeScript fixes were applied in `internal/tracking/worker` (CfGeo, D1 row types, params typing).

	•	Review Area:
	•	TypeScript: Type safety, syntax/style, modularity, architecture, dependency boundaries, performance.
	•	Go: Correctness, idiomatic Go, error handling, concurrency safety, performance, package structure, testability.
	•	Cross-Language: API boundaries, data contracts (JSON/DTOs), error propagation, logging/observability, configuration patterns, security, and build/deployment consistency.

⸻

Deliverables

1. Findings

Identify gaps, inconsistencies, and risks per module/package.

Focus on:
	•	TypeScript
	•	Weak or unsafe typing (any, unsafe casts)
	•	Inconsistent async patterns
	•	Module boundaries and dependency cycles
	•	Runtime vs compile-time validation gaps
	•	Go
	•	Non-idiomatic patterns
	•	Improper error handling
	•	Goroutine lifecycle issues or potential leaks
	•	Misuse of context.Context
	•	Poor package boundaries or overly large packages
	•	Cross-Language
	•	Mismatched data models between TS and Go
	•	API contract inconsistencies
	•	Serialization/deserialization risks
	•	Error format inconsistencies
	•	Logging or tracing gaps

⸻

2. Code Fixes

Provide minimal, idiomatic fixes using:
	•	diff style patches or
	•	concise replacement snippets

Each fix should include a short rationale explaining:
	•	why the change improves correctness or maintainability
	•	whether it reduces technical debt or risk

⸻

3. Action Plan

Produce a prioritized improvement roadmap:

Quick Wins
	•	Formatting
	•	Linter fixes
	•	Small refactors

Medium Improvements
	•	Module/package restructuring
	•	Error-handling consistency
	•	Improved typing or interfaces

Major Refactors
	•	Architecture changes
	•	API contract redesign
	•	Performance improvements

Include impact and risk notes for each item.

⸻

4. Tooling Suggestions

TypeScript
	•	eslint
	•	prettier
	•	typescript-eslint
	•	tsc --noEmit
	•	dependency cycle detection (madge)

Go
	•	gofmt
	•	goimports
	•	golangci-lint
	•	staticcheck
	•	govulncheck
	•	gosec
	•	go test -race

Cross-Language
	•	OpenAPI / schema validation
	•	contract tests between TS and Go services
	•	CI checks for linting, formatting, and security scans

⸻

Additional Focus Areas

Review for:
	•	DRY & YAGNI adherence
	•	maintainable abstractions
	•	performance hot spots
	•	safe concurrency
	•	consistent configuration patterns
	•	observability (logs, metrics, traces)

Flag any design patterns or architectural decisions that may cause scalability or maintainability issues later.