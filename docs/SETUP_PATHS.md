# Setup Paths: `setup.sh` vs `setup-doctor.sh`

## `scripts/setup.sh` (default)
Use this for normal installs and first-time onboarding.

It does only the core golden path:
1. Build/install gog
2. Require the official `gws auth setup` / `gws auth login` flow
3. Import that auth into `gog`
4. Set default alias
5. Validate non-interactive auth and Drive access

### Use when
- New install
- Single account
- No broken prior state
- You want minimal prompts and fast success

---

## `scripts/setup-doctor.sh` (advanced/repair)
Use this for recovery and edge-case environments.

It retains the full advanced flow from the previous monolithic setup:
- reset/clean-reset options
- deeper diagnostics
- environment/keyring troubleshooting
- migration/repair paths
- OpenClaw/MCP registration (`mcporter.json`, `TOOLS.md`, daemon restart)

### Use when
- Existing install is broken
- Keyring/auth state is inconsistent
- You need reset + repair behavior
- You are operating in unusual/headless/legacy conditions

---

## Recommendation
For local onboarding, start with:
```bash
./scripts/setup.sh
```

Use `setup-doctor.sh` immediately instead of `setup.sh` when you need OpenClaw/MCP bootstrap or a headless Linode deployment.

For install/update/deploy on a server, prefer:

```bash
./scripts/deploy.sh
```

It is the single operational entrypoint and can bootstrap first-time setup when needed.

Only if the simple path is not enough, run:
```bash
./scripts/setup-doctor.sh
```
