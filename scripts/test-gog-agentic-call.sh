#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." && pwd)"
cd "$ROOT_DIR"

pass() { echo "[PASS] $*"; }
fail() { echo "[FAIL] $*"; exit 1; }

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

mkdir -p "$tmpdir/bin" "$tmpdir/workspace/config"
printf '{}\n' > "$tmpdir/workspace/config/mcporter.json"

cat > "$tmpdir/bin/mcporter" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" > "${MCPORTER_ARGS_OUT:?}"
EOF
chmod +x "$tmpdir/bin/mcporter"

export PATH="$tmpdir/bin:$PATH"
export WORKSPACE_DIR="$tmpdir/workspace"
export MCPORTER_ARGS_OUT="$tmpdir/args.txt"

bash "$ROOT_DIR/scripts/gog-agentic-call.sh" "drive.listFiles" '{"page":"abc"}'
args="$(<"$tmpdir/args.txt")"
case "$args" in
  *"--config $tmpdir/workspace/config/mcporter.json call gog-agentic.drive_listFiles --args {\"page\":\"abc\"} --output json"*)
    pass "wrapper normalizes dotted names"
    ;;
  *)
    fail "wrapper did not normalize dotted name correctly: $args"
    ;;
esac

bash "$ROOT_DIR/scripts/gog-agentic-call.sh" "gog-agentic.drive_searchFiles" '{"query":"test"}' text
args="$(<"$tmpdir/args.txt")"
case "$args" in
  *"call gog-agentic.drive_searchFiles --args {\"query\":\"test\"} --output text"*)
    pass "wrapper preserves underscored names and output mode"
    ;;
  *)
    fail "wrapper did not preserve underscored name correctly: $args"
    ;;
esac

unset WORKSPACE_DIR
repo_root="$tmpdir/openclaw/workspace/repositories/gogcli-enhanced"
mkdir -p "$repo_root/scripts" "$tmpdir/home/.local/bin" "$tmpdir/openclaw/workspace/config"
printf '{}\n' > "$tmpdir/openclaw/workspace/config/mcporter.json"
cp "$ROOT_DIR/scripts/gog-agentic-call.sh" "$repo_root/scripts/gog-agentic-call.sh"
chmod +x "$repo_root/scripts/gog-agentic-call.sh"
ln -sf "$repo_root/scripts/gog-agentic-call.sh" "$tmpdir/home/.local/bin/gog-agentic-call"
resolved_workspace="$(cd "$tmpdir/openclaw/workspace" && pwd -P)"

bash "$tmpdir/home/.local/bin/gog-agentic-call" "drive.listFiles" '{}'
args="$(<"$tmpdir/args.txt")"
case "$args" in
  *"--config $resolved_workspace/config/mcporter.json call gog-agentic.drive_listFiles --args {} --output json"*)
    pass "wrapper resolves repo path through symlink"
    ;;
  *)
    fail "wrapper did not resolve symlinked repo path correctly: $args"
    ;;
esac

echo "All gog-agentic call wrapper tests passed."
