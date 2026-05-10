---
name: perfection
description: "V1.0 - Commands: audit, coverage, crap, mutate, simplify, harden. Comprehensive Go code quality enforcement for gh-prx: test coverage analysis, CRAP score tracking, mutation testing, simplification patterns, and static analysis. Use when improving code quality, writing tests, or before merging."
compatibility: Requires go 1.23+, git
hooks:
  PostToolUse:
    - matcher: "Read|Write|Edit"
      hooks:
        - type: prompt
          prompt: |
            If a file was read, written, or edited in the perfection directory (path contains 'perfection'), verify that history logging occurred.
            
            Check if History/{YYYY-MM-DD}.md exists and contains an entry for this interaction with:
            - Format: "## HH:MM - {Action Taken}"
            - One-line summary
            - Accurate timestamp (obtained via `Get-Date -Format "HH:mm"` command, never guessed)
            
            If history entry is missing or incomplete, provide specific feedback on what needs to be added.
            If history entry exists and is properly formatted, acknowledge completion.
  Stop:
    - matcher: "*"
      hooks:
        - type: prompt
          prompt: |
            Before stopping, if perfection was used (check if any files in perfection directory were modified), verify that the interaction was logged:
            
            1. Check if History/{YYYY-MM-DD}.md exists in perfection directory
            2. Verify it contains an entry with format "## HH:MM - {Action Taken}" where HH:MM was obtained via `Get-Date -Format "HH:mm"` (never guessed)
            3. Ensure the entry includes a one-line summary of what was done
            
            If history entry is missing:
            - Return {"decision": "block", "reason": "History entry missing. Please log this interaction to History/{YYYY-MM-DD}.md with format: ## HH:MM - {Action Taken}\n{One-line summary}\n\nCRITICAL: Get the current time using `Get-Date -Format \"HH:mm\"` command - never guess the timestamp."}
            
            If history entry exists:
            - Return {"decision": "approve"}
            
            Include a systemMessage with details about the history entry status.
---

# Perfection — Go Code Quality Enforcement

Comprehensive quality enforcement for this Go CLI extension. Every command operates from the repo root and assumes `go.mod` is present.

## Project Context

- **Module**: `github.com/HemSoft/gh-prx`
- **Package**: `main` (single package)
- **Source files**: `main.go`, `prlist.go`
- **Test file**: `main_test.go`
- **Build**: `go build -o gh-prx.exe .`
- **Test**: `go test ./...`
- **Vet**: `go vet ./...`

## Required Tools

Install missing tools before first use:

```powershell
# Static analysis
go install honnef.co/go/tools/cmd/staticcheck@latest

# Cyclomatic complexity
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

# Cognitive complexity
go install github.com/uudashr/gocognit/cmd/gocognit@latest

# Mutation testing
go install github.com/zimmski/go-mutesting/cmd/go-mutesting@latest

# Dead code detection
go install golang.org/x/tools/cmd/deadcode@latest
```

Check if tools are installed: `Get-Command staticcheck, gocyclo, gocognit, go-mutesting, deadcode -ErrorAction SilentlyContinue | Select-Object Name`

## Commands

### `audit` — Full Quality Report

Run all checks and produce a single quality scorecard. Execute in this order:

1. `go vet ./...`
2. `staticcheck ./...`
3. `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
4. `gocyclo -over 5 .`
5. `gocognit -over 5 .`
6. `deadcode .`

Present results as a scorecard table:

```
## Quality Scorecard

| Metric              | Value   | Target  | Status |
|---------------------|---------|---------|--------|
| Vet                 | pass    | pass    | ✅     |
| Staticcheck         | pass    | pass    | ✅     |
| Test Coverage       | 44.5%   | ≥70%    | ❌     |
| Max Cyclomatic      | 12      | ≤10     | ⚠️     |
| Max Cognitive       | 8       | ≤8      | ✅     |
| Dead Code           | 0 funcs | 0       | ✅     |
| Mutation Score      | --      | ≥60%    | ⏭️     |
```

After the table, list the top 3 actionable improvements ranked by impact.

### `coverage` — Detailed Coverage Analysis

Generate and analyze test coverage:

```powershell
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

For each function with <80% coverage:
1. List the function name, file, and current coverage
2. Identify which branches/paths are untested
3. Write the missing test cases using table-driven tests (this repo's established pattern)

**Coverage targets:**
- 🟢 ≥80% — excellent
- 🟡 60-79% — acceptable, improve opportunistically
- 🔴 <60% — requires immediate attention

After analysis, **write the tests** — don't just list what's missing. Follow existing patterns in `main_test.go`:
- Table-driven tests with `t.Run`
- Descriptive test case names
- `t.Fatalf` for assertions (not `t.Errorf` — fail fast)
- Test edge cases: nil inputs, empty slices, boundary values, error paths

Always clean up: `Remove-Item coverage.out -ErrorAction SilentlyContinue`

### `crap` — CRAP Score Analysis

CRAP = Complexity² × (1 - Coverage)³ + Complexity

For each exported and significant unexported function:

1. Get cyclomatic complexity: `gocyclo -over 0 .`
2. Get per-function coverage: `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
3. Calculate CRAP score per function

**CRAP score interpretation:**
- 🟢 ≤5 — clean code, well tested
- 🟡 5-30 — manageable, consider simplifying or adding tests
- 🔴 >30 — high risk: too complex + undertested

Present as table sorted by CRAP score descending:

```
| Function              | Complexity | Coverage | CRAP  | Risk |
|-----------------------|------------|----------|-------|------|
| normalizeCheckState   | 8          | 85%      | 8.0   | 🟡   |
| executeList           | 12         | 0%       | 156   | 🔴   |
```

For any 🔴 function, recommend: simplify first (extract helpers, reduce nesting), then add tests.

Always clean up: `Remove-Item coverage.out -ErrorAction SilentlyContinue`

### `mutate` — Mutation Testing

Mutation testing validates test quality by injecting faults and checking if tests catch them.

```powershell
go-mutesting ./...
```

If `go-mutesting` is not installed or fails, use manual mutation approach:
1. Identify testable pure functions (no side effects, no external calls)
2. For each function, create temporary mutations:
   - Swap `<` with `<=`, `==` with `!=`
   - Change `+` to `-`, `&&` to `||`
   - Remove early returns
   - Change string constants
3. Run `go test ./...` for each mutation
4. If tests still pass → the mutation survived → tests are weak for that code path
5. **Revert every mutation immediately** after testing

Report format:

```
## Mutation Testing Results

| Function            | Mutations | Killed | Survived | Score |
|---------------------|-----------|--------|----------|-------|
| normalizeCheckState | 6         | 5      | 1        | 83%   |
| formatRelativeTime  | 8         | 8      | 0        | 100%  |

### Surviving Mutations (tests need strengthening)
1. `normalizeCheckState`: swapping `hasFail` check order — no test catches this
   → Add test: single FAILURE item should return "fail" even with passing items
```

**Target**: ≥60% mutation kill rate per function.

### `simplify` — Simplification Patterns

Scan for Go-specific simplification opportunities:

1. **`gofmt -s -d .`** — mechanical simplifications (slice expressions, range loops, composite literals)
2. **Complexity reduction** — functions with cyclomatic complexity >10:
   - Extract switch cases into helper functions
   - Replace nested if/else with early returns
   - Use lookup tables (maps) instead of long switch statements
3. **Duplication detection** — look for repeated patterns across functions:
   - Similar switch/case structures
   - Repeated error handling patterns
   - Copy-paste code with minor variations
4. **Interface extraction** — identify groups of methods on the same type that could be interfaces for testability
5. **Dead code** — `deadcode .` to find unreachable functions

For each finding:
- Show the current code
- Show the simplified version
- Explain the improvement (readability, testability, maintainability)
- **Apply the change** if it's safe (preserves behavior)

Run `go vet ./... && go test ./...` after each batch of simplifications to verify nothing broke.

### `harden` — Test Hardening

Improve test quality and robustness:

1. **Analyze existing tests** in `main_test.go`:
   - Are edge cases covered? (nil, empty, zero, negative, max values)
   - Are error paths tested?
   - Do table-driven tests have sufficient case variety?
   - Are assertions checking the right things?

2. **Identify missing test categories**:
   - Functions with 0% coverage
   - Functions tested only for happy path
   - Boundary conditions (off-by-one, empty input, single element)
   - Error injection (malformed JSON, nil pointers, empty strings)

3. **Write hardened tests** following repo conventions:
   - Table-driven with `t.Run`
   - Descriptive case names that explain the scenario
   - `t.Fatalf` for assertions
   - Test both the function output AND side effects
   - Add `t.Helper()` to test helper functions

4. **Verify test independence**:
   - No test should depend on another test's state
   - No test should depend on external services (use test doubles)
   - Tests should be deterministic (inject `time.Now`, avoid random)

After writing tests, run:
```powershell
go test -v -count=1 ./...
go test -race ./...
```

## Quality Gates

These are the quality thresholds. When any command runs, report status against these gates:

| Gate               | Threshold | Enforced |
|--------------------|-----------|----------|
| `go vet`           | 0 errors  | Always   |
| `staticcheck`      | 0 errors  | Always   |
| Test coverage      | ≥70%      | Before merge |
| Max cyclomatic     | ≤10       | Advisory |
| Max cognitive      | ≤8        | Advisory |
| CRAP score         | ≤30       | Before merge |
| Mutation kill rate | ≥60%      | Advisory |
| `go test -race`    | 0 races   | Always   |

## Patterns Established in This Repo

Follow these when writing code or tests:

- **Table-driven tests**: Always use `[]struct` + `t.Run` (see `TestFormatRelativeTime`, `TestCountApprovals`)
- **Fail-fast assertions**: Use `t.Fatalf`, not `t.Errorf`
- **Time injection**: Pass `now time.Time` as parameter, don't call `time.Now()` inside testable functions
- **Nil safety**: Check pointer fields before access (see `Author` nil check in `buildDisplayPullRequest`)
- **Error wrapping**: Use `fmt.Errorf("context: %w", err)` for wrapped errors
- **Sentinel errors**: Define as `var errX = errors.New(...)` at package level (see `errHelpDisplayed`)
- **Graceful degradation**: External calls (GraphQL) use best-effort pattern — failures produce fallback values, not crashes
- **IO abstraction**: Functions accept `io.Writer` for testability (see `run`, `executeList`, `renderTableWithStyle`)
- **Flag parsing**: Use `flag.NewFlagSet` with `ContinueOnError` for subcommand flags

## Anti-Patterns to Flag

When running any command, also flag these if found:

- `fmt.Println` in library code (use `fmt.Fprintln(w, ...)` with injected writer)
- Naked `panic` or `log.Fatal` in non-main functions
- `time.Now()` called directly in testable logic
- Unchecked type assertions
- `interface{}` or `any` without type switches
- Goroutines without synchronization
- `os.Exit` outside of `main()`
- Ignoring returned errors (use `errcheck` if available)
