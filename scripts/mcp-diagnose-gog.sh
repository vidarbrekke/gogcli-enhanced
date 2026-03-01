#!/usr/bin/env bash
# Diagnose gog-agentic MCP: read mcporter.json, run the gog command with same env,
# send initialize + tools/list and report whether tools are returned (or why it failed).
# Usage: ./scripts/mcp-diagnose-gog.sh [path-to-mcporter.json]

set -euo pipefail

MCP_CONFIG="${1:-/root/openclaw-stock-home/.openclaw/workspace/config/mcporter.json}"

echo "MCP config: $MCP_CONFIG"
if [[ ! -f "$MCP_CONFIG" ]]; then
  echo "ERROR: Config file not found."
  exit 1
fi

python3 - "$MCP_CONFIG" <<'PY'
import json
import os
import subprocess
import sys

config_path = sys.argv[1]
with open(config_path, "r", encoding="utf-8") as f:
    data = json.load(f)

entry = (data.get("mcpServers") or {}).get("gog-agentic")
if not entry:
    print("ERROR: gog-agentic not found in mcpServers.")
    sys.exit(1)

cmd = entry.get("command") or "gog"
args = entry.get("args") or ["mcp", "serve"]
env = os.environ.copy()
env.update(entry.get("env") or {})

print("command:", cmd)
print("args:", args)
print("env keys added:", list(entry.get("env") or {}).keys())

# Send initialize then tools/list (newline-delimited JSON-RPC)
init_req = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"diagnose","version":"1.0"}}}\n'
list_req = '{"jsonrpc":"2.0","id":2,"method":"tools/list"}\n'
stdin = init_req + list_req

proc = subprocess.Popen(
    [cmd] + args,
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    env=env,
    text=True,
)
stdout, stderr = proc.communicate(stdin, timeout=15)

if proc.returncode != 0:
    print("ERROR: gog mcp serve exited with code", proc.returncode)
    if stderr:
        print("stderr:", stderr[:2000])
    sys.exit(1)

# Parse last line as tools/list response
lines = [l for l in stdout.strip().split("\n") if l.strip()]
tools_count = 0
for line in lines:
    try:
        r = json.loads(line)
        if r.get("id") == 2 and "result" in r:
            tools = r["result"].get("tools") or []
            tools_count = len(tools)
            names = [t.get("name") for t in tools if t.get("name")]
            print("tools/list returned", tools_count, "tools")
            if "drive.ensureFolder" in names:
                print("  (drive.ensureFolder present)")
            if "docs.create" in names:
                print("  (docs.create present)")
            break
    except json.JSONDecodeError:
        pass

if tools_count == 0:
    print("WARNING: No tools in response. Raw last line:", lines[-1][:500] if lines else "no lines")
    sys.exit(1)
print("OK: gog-agentic responds and exposes tools.")
PY
