#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." && pwd)"
cd "$ROOT_DIR"

pass() { echo "[PASS] $*"; }
fail() { echo "[FAIL] $*"; exit 1; }

# 1) scripts exist
[[ -f scripts/setup.sh ]] || fail "scripts/setup.sh missing"
[[ -f scripts/setup-doctor.sh ]] || fail "scripts/setup-doctor.sh missing"
[[ -f scripts/lib/gws-auth-bridge.sh ]] || fail "gws auth bridge helper missing"
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

# 4) simple setup contains golden-path markers and no native onboarding fallback
grep -q "Simple setup (golden path)" scripts/setup.sh || fail "setup.sh not simplified"
grep -q "require official gws auth setup/login" scripts/setup.sh || fail "setup.sh missing strict gws onboarding"
if grep -q -- "--credentials-stdin" scripts/setup.sh; then fail "setup.sh should not expose native credentials stdin flow"; fi
if grep -q "Select \[1/2\]" scripts/setup.sh; then fail "setup.sh should not expose native credential selector"; fi
pass "setup.sh enforces strict gws onboarding"

# 5) doctor script retains advanced flags
grep -q -- "--clean-reset" scripts/setup-doctor.sh || fail "setup-doctor.sh missing advanced flags"
grep -q -- "--advanced" scripts/setup-doctor.sh || fail "setup-doctor.sh missing advanced mode"
grep -q "Homebrew is required on macOS" scripts/setup-doctor.sh || fail "setup-doctor.sh missing macOS package handling"
pass "setup-doctor.sh retains advanced controls"

# 6) both setup flows prefer the official gws auth bridge
grep -q "gws-auth-bridge.sh" scripts/setup.sh || fail "setup.sh missing gws auth bridge"
grep -q "gws-auth-bridge.sh" scripts/setup-doctor.sh || fail "setup-doctor.sh missing gws auth bridge"
grep -q "gws auth setup" scripts/lib/gws-auth-bridge.sh || fail "gws auth bridge missing official setup flow"
grep -q "gws auth login" scripts/lib/gws-auth-bridge.sh || fail "gws auth bridge missing official login flow"
if grep -q "auth add" scripts/setup-doctor.sh; then fail "setup-doctor.sh should not fall back to native gog auth add"; fi
pass "setup scripts prefer the official gws auth bridge"

# 7) deploy is the single operational entrypoint
grep -q "single entrypoint" scripts/deploy.sh || fail "deploy.sh missing single entrypoint positioning"
grep -q "setup-doctor.sh" scripts/deploy.sh || fail "deploy.sh missing first-time bootstrap path"
grep -q "@googleworkspace/cli" scripts/deploy.sh || fail "deploy.sh missing gws dependency bootstrap"
grep -q "mcporter" scripts/deploy.sh || fail "deploy.sh missing mcporter handling"
grep -q "Homebrew is required on macOS" scripts/deploy.sh || fail "deploy.sh missing macOS dependency handling"
pass "deploy.sh covers bootstrap and update flow"

# 8) embedded TOOLS.md injector should execute without undefined variables
python3 - <<'PY' || fail "setup-doctor.sh TOOLS.md injector is not runnable"
import pathlib
import re
import sys
import tempfile

text = pathlib.Path("scripts/setup-doctor.sh").read_text(encoding="utf-8")
match = re.search(r'python3 - "\$tools_md" <<\'PY\'\n(.*?)\nPY', text, re.S)
if not match:
    raise SystemExit("missing tools injector block")

tmp = tempfile.NamedTemporaryFile(delete=False)
tmp.close()
sys.argv = ["-", tmp.name]
ns = {"__name__": "__main__"}
exec(match.group(1), ns, ns)
PY
pass "setup-doctor.sh TOOLS.md injector runs cleanly"

echo "All setup split tests passed."
