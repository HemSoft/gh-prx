# Project TODO

| Status | Priority | Task | Notes |
|--------|----------|------|-------|
| 📋 | High | [Design `gh prx atm` org-scoped behavior](#design-gh-prx-atm-org-scoped-behavior) | Define default org behavior, `--org`, and `--review-required` semantics |
| 📋 | High | [Implement `gh prx atm`](#implement-gh-prx-atm) | Add `gh prx atm`, `gh prx atm --org HemSoft`, and `gh prx atm --org HemSoft --review-required` |
| 📋 | Medium | [Add tests for `atm`](#add-tests-for-atm) | Cover flag parsing, query construction, and output formatting for atm |
| ✅ | High | Initialize public GitHub repository | 2026-05-10 - Created and pushed `HemSoft/gh-prx` |
| ✅ | High | Scaffold Go extension for `gh prx list` | 2026-05-10 - Added README, Go module, source files, and initial tests |
| ✅ | High | Install Go locally and validate the scaffold | 2026-05-10 - Go 1.23, all 36 tests pass |
| ✅ | High | Verify `gh prx list` end-to-end | 2026-05-10 - Extensively tested against live repos |
| ✅ | Medium | Add tests for `list` | 2026-05-10 - 36 tests covering formatting, normalization, and rendering |
| ✅ | Medium | Document install and usage flows | 2026-05-10 - README fully rewritten with install, usage, and examples |
| ✅ | Low | Prepare release packaging | 2026-05-10 - Auto-release workflow + manual release workflow, 12-platform binaries |

## Progress

**Completed: 7 / 10** (70%)

---

## Remaining Items

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

### Add tests for `atm`

**Location**: `atm_test.go`

**Problem**: The `atm` subcommand needs tests.

**Proposed Solution**:

- Add tests for subcommand routing and flag parsing.
- Add tests for the org-scoped query builder.
- Add tests for review/check normalization specific to atm's cross-repo display.

## Notes

- The repository and local folder were renamed from `gh-extensions` to `gh-prx` so the extension command resolves correctly to `gh prx`.
- The current scaffold intentionally wraps GitHub CLI behavior so it can reuse existing `gh` authentication and repository context.
- Auto-release workflow creates a new patch version on every push to `main`.
