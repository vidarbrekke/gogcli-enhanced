# Setup Paths: `setup.sh` vs `setup-doctor.sh`

## `scripts/setup.sh` (default)
Use this for normal installs and first-time onboarding.

It does only the core golden path:
1. Build/install gog
2. Ensure OAuth client credentials are present
3. Authorize account (`auth add`, auto then manual fallback)
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

### Use when
- Existing install is broken
- Keyring/auth state is inconsistent
- You need reset + repair behavior
- You are operating in unusual/headless/legacy conditions

---

## Recommendation
Start with:
```bash
./scripts/setup.sh
```

Only if that fails, run:
```bash
./scripts/setup-doctor.sh
```
