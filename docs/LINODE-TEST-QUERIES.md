# Test queries for Linode (deployed gog)

Use these after deploying the latest branch to confirm the binary and config work. Replace `YOU@gmail.com` with your configured account.

---

## 1. No auth required

```bash
# Version and build hash
gog version

# Help (sanity check binary + Kong parsing)
gog --help
gog auth --help
gog gmail labels list --help

# Stable exit codes (agent doc)
gog exit-codes
```

---

## 2. Auth / config (requires at least one account)

```bash
# Auth status — text
gog auth status

# Auth status — JSON (exercises JSON output path)
gog auth status --json

# Default account
gog auth status -a YOU@gmail.com
```

---

## 3. Read-only API (one account)

Use `-a YOU@gmail.com` if you have multiple accounts.

**Gmail**

```bash
# Labels — text table
gog gmail labels list -a YOU@gmail.com

# Labels — JSON (exercises captureStdout/JSON path)
gog gmail labels list -a YOU@gmail.com --json

# Drafts list (first page)
gog gmail drafts list -a YOU@gmail.com --max 3 --json
```

**Drive**

```bash
# List files — text
gog drive ls -a YOU@gmail.com --max 2

# List files — JSON
gog drive ls -a YOU@gmail.com --max 2 --json

# Search (no write)
gog drive search "test" -a YOU@gmail.com --max 1 --json
```

**Calendar**

```bash
# Calendar list
gog calendar list -a YOU@gmail.com --json

# Events (next 7 days)
gog calendar events -a YOU@gmail.com --from today --to 7d --max 5 --json
```

**People / Tasks**

```bash
# Profile
gog people me -a YOU@gmail.com --json

# Task lists
gog tasks list -a YOU@gmail.com --json
```

**Sheets / Docs** (if you have a sheet or doc ID)

```bash
# Sheets metadata (read-only)
gog sheets metadata SPREADSHEET_ID -a YOU@gmail.com --json

# Docs info (read-only)
gog docs info DOC_ID -a YOU@gmail.com --json
```

---

## 4. Dry-run (no writes)

```bash
# Drive upload dry-run
gog drive upload /etc/hostname -a YOU@gmail.com --name "test-upload.txt" -n

# Docs edit dry-run (replace in a doc)
gog docs sed DOC_ID "s/foo/bar/" -a YOU@gmail.com -n
```

---

## 5. MCP server (if mcporter/Cursor uses gog)

If the Linode deploy runs the MCP daemon that exposes gog tools:

- In Cursor/IDE, list MCP tools and confirm `gmail`, `drive`, `docs`, etc. appear.
- Call a read-only tool, e.g. `gmail.labelsList` or `drive.filesList`, with `account: "YOU@gmail.com"` and check JSON result.

---

## Quick smoke sequence (copy-paste)

```bash
gog version && \
gog auth status --json | head -c 200 && echo "..." && \
gog gmail labels list -a YOU@gmail.com --json | head -c 300 && echo "..."
```

If all three succeed (version prints, auth status returns JSON, labels return JSON), the binary, config, and JSON output paths are working.
