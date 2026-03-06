# Toyota-style 5 Whys: Broken-pipe tests & Lint

## Issue 1: `encode json: write |1: broken pipe`

### 5 Whys (data-only)

1. **Why does the test fail?**  
   The test process gets `encode json: write |1: broken pipe` from `outfmt.WriteJSON` (outfmt.go:96). So something is writing JSON to file descriptor 1 (stdout) and the write fails with broken pipe.

2. **Why is the pipe broken?**  
   A broken pipe on write means the read end of the pipe was closed before the write finished. In the test process, the only code that creates a pipe on stdout is `captureStdout` in testutil_test.go: it does `r, w, err := os.Pipe()`, sets `os.Stdout = w`, runs `fn()`, then `w.Close()` and reads from `r`. So the write end is closed **after** `fn()` returns. So the pipe breaks when the **writer** (command code) is still writing after the **reader** (the test) has closed the write end—i.e. when the command writes to stdout **after** the test’s `fn()` has returned. That can happen only if (a) the writer runs in a different goroutine and outlives `fn()`, or (b) the writer is in a **different test** that is writing to the **same** global `os.Stdout` that another test replaced with a pipe and then closed.

3. **Why would another test’s pipe be the current stdout?**  
   Because `os.Stdout` is a **process-global** variable. When test A runs `captureStdout(t, fn)`, it sets `os.Stdout = w` (the pipe). While `fn()` runs, **any** code in the process that writes to `os.Stdout` writes to that pipe. If test B runs in parallel (e.g. `go test -p 4`) and its code path calls `outfmt.WriteJSON(ctx, os.Stdout, ...)`, it writes to whatever `os.Stdout` is **at that moment**—which can be test A’s pipe. When test A’s `fn()` returns, test A closes `w` and restores `os.Stdout`. So test B can hit “broken pipe” if it writes after test A closed the pipe but before test B’s view of `os.Stdout` is updated (or if test B started writing while test A’s pipe was still active and test A then closed it).

4. **Why does the command write to `os.Stdout`?**  
   Because many command handlers call `outfmt.WriteJSON(ctx, os.Stdout, ...)` (and similar) **directly**. They do not take the writer from the context/UI. So they always use the global stdout. Evidence: grep shows 100+ call sites of `outfmt.WriteJSON(ctx, os.Stdout, ...)` across internal/cmd. The CLI **does** bind a UI to the context in root.go (ExecuteWithIO builds a UI from the passed stdout/stderr and calls ui.WithUI(ctx, u)), but the command code does not use that UI for JSON output; it uses `os.Stdout`.

5. **Why is that the root cause?**  
   The design assumes “stdout” is the same as “where the CLI was asked to write” when running under ExecuteWithIO. Under tests, “where the CLI was asked to write” is either (a) the pipe passed to ExecuteWithIO (when a test uses captureStdout and then calls Execute), or (b) io.Discard when a test builds its own context with a UI that has Stdout: io.Discard (e.g. runSedIntegration). But the code ignores the UI and uses `os.Stdout`, so (a) it can write to another test’s pipe when tests run in parallel, and (b) in runSedIntegration it still writes to global stdout (which may be another test’s pipe) instead of the UI’s Discard. So the **root cause** is: **JSON (and any stdout) output uses the global `os.Stdout` instead of the writer from the UI stored in context, causing races when tests run in parallel.**

### Validation

- **Reproduction:** `go test -count=1 ./internal/cmd/ -run 'TestSedIntegration_RegexDigitClass|TestExecute_ClassroomMoreCommands_JSON' -parallel 4` can fail with broken pipe; with `-p 1` or running a single test it typically passes.
- **Proof of cause:** The failing tests either (1) use runSedIntegration (context has UI with io.Discard) but the sed path calls `outfmt.WriteJSON(ctx, os.Stdout, ...)` in docs_sed_helpers.go:143, or (2) use Execute(... --json) from a test that only captures stderr while another test may have replaced stdout—so any command that writes JSON uses os.Stdout and can hit the other test’s closed pipe.

### Fix (root-cause, DRY)

1. **Single source of “stdout”:** Get the stdout writer from the UI in context when present; otherwise fall back to `os.Stdout`. So: add `Stdout() io.Writer` to the UI (store the writer used to create the UI), and in cmd add `stdoutWriter(ctx context.Context) io.Writer` that returns `ui.FromContext(ctx).Stdout()` when the UI is present, else `os.Stdout`.
2. **Use it everywhere JSON (or stdout) is written:** Replace all `outfmt.WriteJSON(ctx, os.Stdout, ...)` and any other direct `os.Stdout` use for command output with `stdoutWriter(ctx)`. That way:
   - runSedIntegration’s context has UI with Stdout: io.Discard → writes go to Discard, no pipe.
   - Execute() from a test that uses captureStdout passes the pipe as stdout to ExecuteWithIO → UI gets the pipe → stdoutWriter(ctx) returns the pipe → writes stay on the correct pipe and no other test closes it.

---

## Issue 2: Lint (73 issues: wsl_v5, err113, gosec, etc.)

### 5 Whys (summary)

1. **Why does `make lint` fail?**  
   The linter (golangci-lint with many analyzers) reports 73 issues across the repo.

2. **Why are there so many?**  
   Different rules: wsl_v5 (whitespace/statement grouping), err113 (error wrapping/sentinel), gosec (security), etc. They are mostly **style and best-practice** violations, not introduced by a single change.

3. **Why weren’t they fixed earlier?**  
   Either the linter config or rule set was updated and started flagging existing code, or these packages (e.g. internal/mcp) were added without satisfying all rules.

4. **Why is the main failure from wsl_v5 in transport_stdio.go?**  
   wsl_v5 requires blank lines between certain statement groups. The MCP transport file has many consecutive statements without the required blank lines, so the linter reports “missing whitespace above this line.”

5. **What is the root cause?**  
   **Root cause for lint failure:** Code does not satisfy the current linter rules (wsl_v5 and others). There is no single underlying bug; the “fix” is to either (a) satisfy the rules (add blanks, wrap errors, etc.) or (b) narrow the rule set / exclude certain files if the project decides to.

### Fix (YAGNI)

- **Broken-pipe:** Fix first (above); it is a real concurrency bug.
- **Lint:** Address in a separate pass: either fix wsl_v5 (and other) violations in the reported files, or adjust .golangci.yml to disable or relax specific rules for specific paths, per project policy.
