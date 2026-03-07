#!/usr/bin/env bash

sync_openclaw_gog_skill() {
  local workspace_dir="$1"
  local skill_src="${ROOT_DIR:?}/openclaw/skills/gog/SKILL.md"
  local skill_dest="$workspace_dir/skills/gog/SKILL.md"

  [[ -f "$skill_src" ]] || return 0

  mkdir -p "$(dirname "$skill_dest")"
  cp "$skill_src" "$skill_dest"
}
