#!/usr/bin/env bash
set -euo pipefail

UPSTREAM_REMOTE="${1:-upstream}"
UPSTREAM_BRANCH="${2:-main}"
BASE_REF="${UPSTREAM_REMOTE}/${UPSTREAM_BRANCH}"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "Error: must run inside a git repository." >&2
  exit 1
fi

if ! git remote get-url "${UPSTREAM_REMOTE}" >/dev/null 2>&1; then
  echo "Error: remote '${UPSTREAM_REMOTE}' not found." >&2
  echo "Usage: $0 [remote] [branch]" >&2
  exit 1
fi

echo "== Upstream Sync Check =="
echo "Remote: ${UPSTREAM_REMOTE}"
echo "Branch: ${UPSTREAM_BRANCH}"
echo

git fetch "${UPSTREAM_REMOTE}" --prune >/dev/null

CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
read -r AHEAD BEHIND < <(git rev-list --left-right --count "${CURRENT_BRANCH}...${BASE_REF}")

echo "Current branch: ${CURRENT_BRANCH}"
echo "Divergence vs ${BASE_REF}: ahead ${AHEAD}, behind ${BEHIND}"
echo

if [[ "${BEHIND}" -eq 0 ]]; then
  echo "Status: no upstream commits to merge right now."
else
  echo "Status: upstream has ${BEHIND} new commit(s) available to merge/rebase."
fi
echo

echo "Top changed areas since ${BASE_REF}:"
git diff --name-only "${BASE_REF}...${CURRENT_BRANCH}" \
  | awk -F/ 'NF>=2 {print $1"/"$2} NF<2 {print $1"/"}' \
  | sort | uniq -c | sort -nr | head -n 15
echo

echo "Likely conflict hotspots (internal/cmd churn):"
git log --name-only --pretty=format: "${BASE_REF}..${CURRENT_BRANCH}" \
  | awk '/^internal\/cmd\// {print}' \
  | sort | uniq -c | sort -nr | head -n 20
echo

if [[ "${BEHIND}" -gt 0 ]]; then
  echo "Upcoming upstream commits (first 20):"
  git log --oneline "${CURRENT_BRANCH}..${BASE_REF}" | head -n 20
  echo
fi

echo "Done."
