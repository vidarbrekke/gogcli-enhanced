# OpenClaw upstream sync — 2026-08 (surgical)

Branch: `sync/openclaw-2026-08-security`  
Upstream: `openclaw/gogcli` (`openclaw/main`, tip `eb85a993` at assessment)  
Merge-base with our `main`: `0ed89978`

## Ported (this branch)

HTTP hardening adapted from upstream `#995` / `#998` (no Photos/Zoom/module rename):

- `internal/googleapi`: `ResponseHeaderTimeout` on base transport + `NewBoundedHTTPClient`
- `internal/cmd`: Gmail tracking uses `outboundHTTPClient`
- Gmail watch serve: `ReadTimeout` / `IdleTimeout` / `MaxHeaderBytes`
- OAuth callback server: same bounded timeouts via `newOAuthCallbackServer`

## Deferred

**Revoked OAuth auto-recovery** (`229da128` / `#974`): do **not** cherry-pick.

- Depends on upstream `AuthDependencies`, `persistingTokenSource`, and `app.Runtime` this fork lacks.
- MCP/GWS paths must stay non-interactive (no browser on stdio / external `gws`).
- Future work: fork-shaped adaptation on current `client.go` + `transport.go` only (invalid_grant → confirm → `Authorize` with force consent → reset token source), with `--no-input` / MCP returning a clear reauth error.
