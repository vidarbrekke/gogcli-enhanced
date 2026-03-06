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

echo "All gog-agentic call wrapper tests passed."
