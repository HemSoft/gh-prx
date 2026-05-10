# Project TODO

| Status | Priority | Task | Notes |
|--------|----------|------|-------|
| ⏸️ | High | [Install Go locally and validate the scaffold](#install-go-locally-and-validate-the-scaffold) | Blocked: `go` is not installed in the current environment |
| 📋 | High | [Verify `gh prx list` end-to-end](#verify-gh-prx-list-end-to-end) | Install the extension locally and test filters, JSON, and table output |
| 📋 | High | [Design `gh prx atm` org-scoped behavior](#design-gh-prx-atm-org-scoped-behavior) | Define default org behavior, `--org`, and `--review-required` semantics |
| 📋 | High | [Implement `gh prx atm`](#implement-gh-prx-atm) | Add `gh prx atm`, `gh prx atm --org HemSoft`, and `gh prx atm --org HemSoft --review-required` |
| 📋 | Medium | [Add tests for `list` and `atm`](#add-tests-for-list-and-atm) | Cover flag parsing, query construction, and output formatting |
| 📋 | Medium | [Document install and usage flows](#document-install-and-usage-flows) | Expand README with `list`, `atm`, and local install examples |
| 📋 | Low | [Prepare release packaging](#prepare-release-packaging) | Add a release workflow and precompiled binaries for public installs |
| ✅ | High | Initialize public GitHub repository | 2026-05-10 - Created and pushed `HemSoft/gh-prx` |
| ✅ | High | Scaffold Go extension for `gh prx list` | 2026-05-10 - Added README, Go module, source files, and initial tests |

## Progress

**Completed: 2 / 9** (22%)

---

## Remaining Items

### Install Go locally and validate the scaffold

**Location**: local development environment, `go.mod`, `main.go`, `prlist.go`, `main_test.go`

**Problem**: The repository was scaffolded, but the current environment does not have the Go toolchain installed, so the extension has not been built or exercised yet.

**Proposed Solution**:

- Install Go 1.22 or newer locally.
- Run `go mod tidy`, `go build`, and `go test ./...`.
- Fix any compile or test issues that show up once the toolchain is available.

---

### Verify `gh prx list` end-to-end

**Location**: `main.go`, `prlist.go`, README usage examples

**Problem**: The command shape is scaffolded, but the real UX still needs validation against live repository data.

**Proposed Solution**:

- Install the extension locally with `gh extension install .`.
- Run `gh prx list` against the current repo and at least one repo with active PRs.
- Verify table columns, truncation, relative timestamps, `--json`, and pass-through filters.

---

### Design `gh prx atm` org-scoped behavior

**Location**: command design and query strategy

**Problem**: `gh pr list` is repo-scoped by default, while `gh prx atm` needs to surface pull requests assigned to the current user across an organization.

**Proposed Solution**:

- Decide whether bare `gh prx atm` should infer the org from the current repo owner or require an explicit org for non-org repos.
- Define the underlying GitHub search or API query for "assigned to me across an org".
- Decide whether `--review-required` means "review requested from me" or "PRs still needing review".
- Reuse the same display model as `gh prx list` where it makes sense.

---

### Implement `gh prx atm`

**Location**: `main.go`, new command handler(s), shared query/render helpers

**Problem**: The extension only supports `gh prx list` right now.

**Proposed Solution**:

- Add an `atm` subcommand to the root command dispatcher.
- Support:
  - `gh prx atm`
  - `gh prx atm --org HemSoft`
  - `gh prx atm --org HemSoft --review-required`
- Use authenticated GitHub CLI context so the command works without separate login handling.
- Prefer shared formatting helpers so `list` and `atm` stay visually consistent.

---

### Add tests for `list` and `atm`

**Location**: `main_test.go` and any new test files

**Problem**: The initial tests only cover a small slice of current behavior.

**Proposed Solution**:

- Add tests for subcommand routing and flag parsing.
- Add tests for the org-scoped query builder used by `atm`.
- Add tests for review/check/status normalization and edge-case output formatting.

---

### Document install and usage flows

**Location**: `README.md`

**Problem**: The README explains the current scaffold, but it does not yet cover the future `atm` workflow or the expected development loop after Go is installed.

**Proposed Solution**:

- Add setup steps after Go installation.
- Add usage examples for `gh prx list` and `gh prx atm`.
- Document any org-scoping assumptions and the meaning of `--review-required`.

---

### Prepare release packaging

**Location**: release workflow and binary naming

**Problem**: The repo is public, but users still need a convenient install path for a compiled extension release.

**Proposed Solution**:

- Add a release workflow for cross-platform binaries.
- Publish assets named for GitHub CLI extension discovery.
- Document versioning and release steps in the repository.

## Notes

- The repository and local folder were renamed from `gh-extensions` to `gh-prx` so the extension command resolves correctly to `gh prx`.
- The current scaffold intentionally wraps GitHub CLI behavior so it can reuse existing `gh` authentication and repository context.
