# Analysis of remaining origin branches (post–merge cleanup)

After merging `feature/upstream-sedmat-import` into `main` and deleting the six branches that were fully contained in main, eight branches remain. This document summarizes what makes each unique and whether it is **mergeable** (unique value, should be merged or cherry-picked) or **obsolete** (already in main or superseded).

---

## Deleted (fully in main)

These were deleted from `origin`:

- `feat/calendar-users`
- `fix/normalize-mime`
- `takeover/pr-18`
- `temp/fix-58-sheets-readonly-drive`
- `temp/issue-58-readonly`
- `temp/test-more-sheets-scopes`

---

## 1. `add-thread-count-indicator-new` — **Mergeable (small)**

- **Commits ahead of main:** 2  
- **Unique content:** Gmail thread message counts in search results (#99); “Add thread message count to search results”.
- **Main today:** Has `GmailThreadGetCmd`, `GmailThreadModifyCmd`, `GmailThreadAttachmentsCmd` but search/list output does not show message counts.
- **Verdict:** **Mergeable.** Small, focused feature; no overlap with main. Merge or cherry-pick the two commits.

---

## 2. `drive-comments` — **Obsolete**

- **Commits ahead of main:** 3 (drive comments subcommand, changelog, lint fix).
- **Main today:** Already has “Add drive comments subcommand” (commit 3e890f5 and later refactors). `internal/cmd/drive_comments.go` and related files exist on main.
- **Verdict:** **Obsolete.** Same feature was brought into main via the big merge. Safe to delete the branch.

---

## 3. `feat/gmail-thread-attachments-v2` — **Obsolete**

- **Commits ahead of main:** 9 (thread attachments, mail/email aliases, keychain/secrets fixes, inline attachments, flattened headers).
- **Main today:** Has `GmailThreadAttachmentsCmd`, keychain and secrets work, and extensive gmail/drive/sheets changes from the merge.
- **Verdict:** **Obsolete.** Functionality is present on main; branch is an older line of development. Safe to delete.

---

## 4. `feature/sheets-format-command` — **Obsolete**

- **Commits ahead of main:** 2 (“add format command”, “reuse sheets format helpers” #72).
- **Main today:** Has `SheetsFormatCmd`, `sheets_format.go`, and full format/edit support.
- **Verdict:** **Obsolete.** Sheets format command is on main. Safe to delete.

---

## 5. `feature/thread-modify` — **Obsolete**

- **Commits ahead of main:** 3 (thread modify for batch label operations, refine command, changelog).
- **Main today:** Has `GmailThreadModifyCmd` and “Modify labels on all messages in a thread”.
- **Verdict:** **Obsolete.** Thread modify is on main. Safe to delete.

---

## 6. `fix/auth-template-query-flag` — **Mergeable (tiny)**

- **Commits ahead of main:** 1  
- **Unique content:** “fix: correct gmail search example in auth template (#89) (thanks @rvben)” — CHANGELOG + template fix.
- **Main today:** CHANGELOG does not mention #89; auth template may still have the old example.
- **Verdict:** **Mergeable.** Single-commit doc/template fix. Prefer cherry-pick onto current main (may need conflict resolution in CHANGELOG).

---

## 7. `fix/auth-ui-improvements` — **Review (possible partial overlap)**

- **Commits ahead of main:** 12  
- **Unique content:** Auth UI refactors: countdown timer, success template consolidation, login/manage alias, cancellable post-success sleep, GitHub icon via CSS mask, tests.
- **Main today:** Auth and keychain logic have evolved; may include some of this or a different approach.
- **Verdict:** **Review.** Could be mergeable (distinct UX improvements) or partly obsolete. Recommend a focused diff of `internal/cmd/auth*` and templates (e.g. `main..origin/fix/auth-ui-improvements`) and a quick manual check before merging or dropping.

---

## 8. `fix/keychain-unlock-macos` — **Review (possible partial overlap)**

- **Commits ahead of main:** 12  
- **Unique content:** macOS keychain lock detection, unlock helpers, “check keychain before token import / in manage server flow”, no-input respect, platform-specific lint/test handling.
- **Main today:** Keychain and secrets handling were updated in the big merge; may already cover some of this.
- **Verdict:** **Review.** Likely some unique behavior (e.g. unlock guidance). Recommend diff `main..origin/fix/keychain-unlock-macos` on `internal/secrets` and auth paths, then either merge the branch or cherry-pick the non-duplicate commits.

---

## Obsolete branches (deleted from origin)

These were deleted from `origin` on cleanup:

- `drive-comments`
- `feat/gmail-thread-attachments-v2`
- `feature/sheets-format-command`
- `feature/thread-modify`

---

## Merge / cherry-pick steps (mergeable branches)

Use these from a clean `main` (e.g. `git checkout main && git pull origin main`).

### 1. `add-thread-count-indicator-new` (2 commits)

**Option A — merge (simplest)**  
Brings in both commits in one step; use if you want a single merge commit.

```bash
git checkout main
git pull origin main
git merge origin/add-thread-count-indicator-new -m "Merge branch 'add-thread-count-indicator-new' (gmail thread message counts #99)"
# Resolve conflicts if any, then:
git push origin main
```

**Option B — cherry-pick (linear history)**  
Apply the two commits on top of main in order:

```bash
git checkout main
git pull origin main
git cherry-pick 7fc1c96   # feat(gmail): Add thread message count to search results
git cherry-pick 49f1aa7   # feat: show gmail thread message counts (#99)
# Resolve conflicts if any, then:
git push origin main
```

---

### 2. `fix/auth-template-query-flag` (1 commit)

Single commit: correct Gmail search example in auth template (#89). Cherry-pick is ideal; watch for CHANGELOG conflicts.

```bash
git checkout main
git pull origin main
git cherry-pick daec53a   # fix: correct gmail search example in auth template (#89)
# If CHANGELOG conflicts: keep both the new #89 entry and existing entries, then git add CHANGELOG.md && git cherry-pick --continue
git push origin main
```

---

### 3. `fix/auth-ui-improvements` (review first)

12 commits; possible overlap with main. Inspect then merge or cherry-pick.

```bash
# Inspect what’s unique
git fetch origin
git log main..origin/fix/auth-ui-improvements --oneline
git diff main..origin/fix/auth-ui-improvements -- internal/cmd/auth internal/cmd/*auth* --stat

# If you want to merge the whole branch:
git checkout main
git merge origin/fix/auth-ui-improvements -m "Merge branch 'fix/auth-ui-improvements'"
# Resolve conflicts (auth/templates), run tests, then git push origin main

# If you prefer cherry-picking only some commits, list and pick by SHA:
git log main..origin/fix/auth-ui-improvements --oneline
git cherry-pick <SHA1> <SHA2> ...
```

---

### 4. `fix/keychain-unlock-macos` (review first)

12 commits; keychain/secrets may overlap with main. Inspect then merge or cherry-pick.

```bash
# Inspect what’s unique
git fetch origin
git log main..origin/fix/keychain-unlock-macos --oneline
git diff main..origin/fix/keychain-unlock-macos -- internal/secrets internal/cmd/auth internal/cmd/*auth* --stat

# If you want to merge the whole branch:
git checkout main
git merge origin/fix/keychain-unlock-macos -m "Merge branch 'fix/keychain-unlock-macos'"
# Resolve conflicts, run tests (including on macOS if possible), then git push origin main

# Or cherry-pick selected commits after reviewing:
git cherry-pick <SHA1> <SHA2> ...
```

---

## Summary table (updated)

Obsolete branches below have been **deleted** from origin.

| Branch | Commits | Verdict | Action |
|--------|--------|--------|--------|
| `add-thread-count-indicator-new` | 2 | Mergeable | See merge/cherry-pick steps above |
| ~~`drive-comments`~~ | ~~3~~ | ~~Obsolete~~ | **Deleted** |
| ~~`feat/gmail-thread-attachments-v2`~~ | ~~9~~ | ~~Obsolete~~ | **Deleted** |
| ~~`feature/sheets-format-command`~~ | ~~2~~ | ~~Obsolete~~ | **Deleted** |
| ~~`feature/thread-modify`~~ | ~~3~~ | ~~Obsolete~~ | **Deleted** |
| `fix/auth-template-query-flag` | 1 | Mergeable | See cherry-pick steps above |
| `fix/auth-ui-improvements` | 12 | Review | See review/merge steps above |
| `fix/keychain-unlock-macos` | 12 | Review | See review/merge steps above |

**Recommended next steps**

1. ~~Delete obsolete branches~~ **Done:** `drive-comments`, `feat/gmail-thread-attachments-v2`, `feature/sheets-format-command`, `feature/thread-modify` have been deleted from origin.
2. Merge or cherry-pick (exact steps in “Merge / cherry-pick steps” above): `add-thread-count-indicator-new`, `fix/auth-template-query-flag`.
3. Review then merge or cherry-pick (steps above): `fix/auth-ui-improvements`, `fix/keychain-unlock-macos`.

---

## Remaining branch conflicts (merge into current main)

Dry-run merges were run from current `main` to see what conflicts each branch would introduce.

### `add-thread-count-indicator-new`

- **Conflicts:** `CHANGELOG.md` (content).
- **Why:** Main’s CHANGELOG has grown and reorganized; the branch adds a short #99 entry that lands in the same area. Both sides changed the same lines.
- **Resolution:** Keep main’s structure, add a single line for “Gmail: show thread message counts in search results (#99)” in the right section (e.g. under 0.12.0 Added), or accept the branch’s block and remove any duplicate.

### `fix/auth-template-query-flag`

- **Conflicts:** `CHANGELOG.md` only.
- **Why:** Branch adds one CHANGELOG entry for #89; main has many new entries in the same region.
- **Resolution:** Trivial. Resolve CHANGELOG by adding the #89 fix line in the correct place (Fixed section for the right version). No code conflicts.

### `fix/auth-ui-improvements`

- **Conflicts:**  
  `CHANGELOG.md`, `internal/cmd/auth.go`, `internal/googleauth/accounts_server.go`, `internal/googleauth/oauth_flow.go`, `internal/googleauth/templates/accounts.html`, `internal/googleauth/templates/success.html`, `internal/googleauth/wait_post_success_test.go` (add/add).
- **Why:** Main and the branch both changed auth flow, server behavior, and templates (e.g. countdown, success page, login/manage). Same areas edited differently.
- **Resolution:** Non-trivial. Resolve by hand: keep main’s behavior where it’s the source of truth, then re-apply the branch’s UX improvements (countdown, template consolidation, cancellable sleep) on top, or merge the branch’s version and re-add any main-only behavior. Run auth tests after.

### `fix/keychain-unlock-macos`

- **Conflicts:**  
  `CHANGELOG.md`, `internal/cmd/auth.go`, `internal/cmd/auth_add_test.go`, `internal/cmd/auth_cmd_test.go`, `internal/cmd/auth_keychain_test.go` (add/add), `internal/cmd/auth_text_test.go`, `internal/cmd/execute_auth_add_test.go`, `internal/googleauth/accounts_server.go`, `internal/secrets/keychain_darwin.go` (add/add), `internal/secrets/keychain_darwin_test.go` (add/add), `internal/secrets/keychain_other.go` (add/add), `internal/secrets/store.go`, `internal/secrets/store_test.go`.
- **Why:** Main already has keychain and secrets changes (different API surface and tests). The branch adds macOS unlock detection, unlock helpers, and “check keychain before OAuth” in the same files. Many add/add conflicts mean both sides added similar or overlapping code (e.g. darwin keychain files, auth tests).
- **Resolution:** Most work. File-by-file: in `internal/secrets` keep main’s structure and inject the branch’s unlock logic and helpers; in auth commands and `accounts_server` merge “check keychain before flow” with main’s current flow; in tests merge or replace so both keychain and auth tests pass. Run full test suite and, if possible, test on macOS.

**Summary:** `add-thread-count-indicator-new` and `fix/auth-template-query-flag` only conflict in CHANGELOG and are easy to fix. `fix/auth-ui-improvements` and `fix/keychain-unlock-macos` conflict in auth and secrets code and need careful manual merge and testing.
