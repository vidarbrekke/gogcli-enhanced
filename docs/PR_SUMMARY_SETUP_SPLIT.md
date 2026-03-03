# PR Summary: Split setup into simple + doctor paths

## Problem
The original `scripts/setup.sh` had grown into a large, multi-purpose flow that mixed:
- first-time onboarding
- advanced reset/repair
- environment diagnostics
- recovery branches

This made common installs harder than necessary.

## Solution
Implemented a two-path strategy:

### 1) `scripts/setup.sh` (simple golden path)
Focused on the 90% case:
- build/install
- credentials check/import
- account auth
- default alias
- auth + Drive validation

### 2) `scripts/setup-doctor.sh` (advanced/repair)
Preserves the original complex flow for:
- reset/clean-reset
- diagnostics and remediation
- edge cases

## Added test coverage
`./scripts/test-setup-split.sh` validates:
- both scripts exist
- syntax valid
- simple setup references doctor flow
- simple setup includes auth add path
- doctor setup retains advanced flags

## Backward compatibility
- Advanced functionality is not removed, only relocated.
- Existing operators can use `setup-doctor.sh` for prior behavior.

## Operator guidance
Use:
```bash
./scripts/setup.sh
```
Fallback:
```bash
./scripts/setup-doctor.sh
```
