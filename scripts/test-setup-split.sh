#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." && pwd)"
cd "$ROOT_DIR"

pass() { echo "[PASS] $*"; }
fail() { echo "[FAIL] $*"; exit 1; }

# 1) scripts exist
[[ -f scripts/setup.sh ]] || fail "scripts/setup.sh missing"
[[ -f scripts/setup-doctor.sh ]] || fail "scripts/setup-doctor.sh missing"
pass "Both setup scripts exist"

# 2) syntax valid
bash -n scripts/setup.sh || fail "setup.sh syntax invalid"
bash -n scripts/setup-doctor.sh || fail "setup-doctor.sh syntax invalid"
pass "Syntax checks pass"

# 3) help output mentions doctor path
if scripts/setup.sh --help | grep -q "setup-doctor.sh"; then
  pass "setup.sh help references setup-doctor.sh"
else
  fail "setup.sh help missing setup-doctor.sh reference"
fi

# 4) simple setup contains golden-path markers + credential mode UX
grep -q "Simple setup (golden path)" scripts/setup.sh || fail "setup.sh not simplified"
grep -q "auth add" scripts/setup.sh || fail "setup.sh missing auth add flow"
grep -q "Select \[1/2\]" scripts/setup.sh || fail "setup.sh missing explicit credential mode selector"
grep -q -- "--credentials-stdin" scripts/setup.sh || fail "setup.sh missing --credentials-stdin option"
pass "setup.sh appears to be golden-path with improved credential UX"

# 5) doctor script retains advanced flags
grep -q -- "--clean-reset" scripts/setup-doctor.sh || fail "setup-doctor.sh missing advanced flags"
grep -q -- "--advanced" scripts/setup-doctor.sh || fail "setup-doctor.sh missing advanced mode"
pass "setup-doctor.sh retains advanced controls"

echo "All setup split tests passed."
