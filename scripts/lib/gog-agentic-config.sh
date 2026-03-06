#!/usr/bin/env bash

gog_agentic_backend() {
  local backend="${GOG_MCP_BACKEND:-${GOG_BACKEND:-gws}}"
  backend="$(printf '%s' "$backend" | tr '[:upper:]' '[:lower:]' | xargs 2>/dev/null || true)"
  if [[ -z "$backend" ]]; then
    backend="gws"
  fi
  printf '%s\n' "$backend"
}

gog_agentic_env_json() {
  python3 - <<'PY'
import json
import os

env = {}
for key in ("XDG_CONFIG_HOME", "GOG_KEYRING_BACKEND"):
    value = os.environ.get(key, "").strip()
    if value:
        env[key] = value

password_file = os.environ.get("GOG_KEYRING_PASSWORD_FILE", "").strip()
if not password_file:
    password_file = os.environ.get("MCP_PASSWORD_FILE_PATH", "").strip()
if password_file:
    env["GOG_KEYRING_PASSWORD_FILE"] = password_file

backend = os.environ.get("GOG_MCP_BACKEND", "").strip() or os.environ.get("GOG_BACKEND", "").strip() or "gws"
env["GOG_BACKEND"] = backend.lower()

print(json.dumps(env, separators=(",", ":")))
PY
}

gog_agentic_upsert_mcporter_config() {
  local config_path="$1"
  local gog_cmd="$2"
  local env_json

  env_json="$(gog_agentic_env_json)"
  python3 - "$config_path" "$gog_cmd" "$env_json" <<'PY'
import json
import os
import sys

config_path, gog_cmd, env_json = sys.argv[1:4]
env_obj = json.loads(env_json)

if os.path.exists(config_path):
    with open(config_path, "r", encoding="utf-8") as f:
        data = json.load(f)
else:
    data = {}

if not isinstance(data, dict):
    data = {}

mcp_servers = data.get("mcpServers")
if not isinstance(mcp_servers, dict):
    mcp_servers = {}

existing_entry = mcp_servers.get("gog-agentic")
existing_env = {}
if isinstance(existing_entry, dict):
    maybe_env = existing_entry.get("env")
    if isinstance(maybe_env, dict):
        existing_env = maybe_env

entry = {
    "command": gog_cmd,
    "args": ["mcp", "serve"],
    "lifecycle": {"mode": "keep-alive"},
}
merged_env = dict(existing_env)
merged_env.update(env_obj)
if merged_env:
    entry["env"] = merged_env

mcp_servers["gog-agentic"] = entry
data["mcpServers"] = mcp_servers

if "imports" not in data or not isinstance(data.get("imports"), list):
    data["imports"] = []

os.makedirs(os.path.dirname(config_path), exist_ok=True)
with open(config_path, "w", encoding="utf-8") as f:
    json.dump(data, f, indent=2)
    f.write("\n")
PY
}
