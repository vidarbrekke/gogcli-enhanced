#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." && pwd)"
cd "$ROOT_DIR"

pass() { echo "[PASS] $*"; }
fail() { echo "[FAIL] $*"; exit 1; }

source "$ROOT_DIR/scripts/lib/gog-agentic-config.sh"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

cfg="$tmpdir/mcporter.json"
export XDG_CONFIG_HOME="$tmpdir/xdg"
export GOG_KEYRING_BACKEND="file"
export GOG_KEYRING_PASSWORD_FILE="$tmpdir/keyring.password"
unset GOG_BACKEND
unset GOG_MCP_BACKEND

env_json="$(gog_agentic_env_json)"
python3 - "$env_json" <<'PY' || fail "gog_agentic_env_json missing expected fields"
import json
import sys

env = json.loads(sys.argv[1])
assert env["XDG_CONFIG_HOME"]
assert env["GOG_KEYRING_BACKEND"] == "file"
assert env["GOG_KEYRING_PASSWORD_FILE"]
assert env["GOG_BACKEND"] == "gws"
PY
pass "gog_agentic_env_json defaults backend to gws"

gog_agentic_upsert_mcporter_config "$cfg" "/abs/path/to/gog"
python3 - "$cfg" <<'PY' || fail "gog_agentic_upsert_mcporter_config wrote unexpected config"
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

entry = data["mcpServers"]["gog-agentic"]
assert entry["command"] == "/abs/path/to/gog"
assert entry["args"] == ["mcp", "serve"]
assert entry["lifecycle"]["mode"] == "keep-alive"
assert entry["env"]["GOG_BACKEND"] == "gws"
assert isinstance(data["imports"], list)
PY
pass "gog_agentic_upsert_mcporter_config writes gog-agentic entry"

python3 - "$cfg" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

data["mcpServers"]["gog-agentic"]["env"]["CUSTOM_KEEP"] = "1"
with open(sys.argv[1], "w", encoding="utf-8") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
PY

gog_agentic_upsert_mcporter_config "$cfg" "/abs/path/to/gog"
python3 - "$cfg" <<'PY' || fail "gog_agentic_upsert_mcporter_config should preserve existing env keys"
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as f:
    data = json.load(f)

entry = data["mcpServers"]["gog-agentic"]
assert entry["env"]["CUSTOM_KEEP"] == "1"
assert entry["env"]["GOG_BACKEND"] == "gws"
PY
pass "gog_agentic_upsert_mcporter_config preserves existing env keys"

echo "All gog-agentic config tests passed."
