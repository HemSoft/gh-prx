# gh-prx

A GitHub CLI extension that supercharges `gh pr list` with a richer, color-coded table view — approvals, AI reviewer status, check details, comment resolution, and clickable PR links. Also includes `gh prx atm` for org-wide PR visibility.

```
#    Title                                             Author         State  Review    AI    Appv  Checks   Cmts   Branch                 Updated
#12  RPLAT-18678: Migrate user-groups to .NET 10       Saiprasa1994   open   approved  pass  2     pending  19/19  feature/RPLAT-18678    23h
#10  .net 10 upgradation                               Aparna-Relias  open   review    -     0     fail     -      feature/RPLAT-8516     17d
#5   feat(user-groups): Add golden-path IaC structure  jomartin1191   open   review    fail  0     pass     2/4    golden-path-alignment  4mo
```

## Installation

Requires [GitHub CLI](https://cli.github.com/) (`gh`) authenticated with your account.

```bash
gh extension install HemSoft/gh-prx
```

That's it. Prebuilt binaries are available for all platforms — no Go toolchain needed.

## Usage

```bash
gh prx list [flags]    # enriched PR list for current repo
gh prx atm [flags]     # org-wide PRs needing your attention
gh prx version         # show version and check for updates (also: --version, -v)
```

## What `gh prx list` adds

Compared to `gh pr list`, this command keeps all existing filters but renders a denser, color-coded table:

| Column   | Description |
|----------|-------------|
| **#**    | PR number — clickable link to the PR on GitHub (terminals with OSC 8 support) |
| **Title**| Truncated to 51 chars |
| **Author**| PR author login |
| **State**| `open`, `draft`, `closed`, or `merged` |
| **Review**| Review decision: `approved`, `changes`, or `review` (pending) |
| **AI**   | AI reviewer status: `pass` (approved/no issues), `fail` (issues found), or `-` (no AI review). Detects CodeRabbit, Copilot PR reviewer, and other `[bot]` reviewers |
| **Appv** | Count of human approvals |
| **Checks**| CI status: `pass`, `fail`, `pending`, or `-`. Includes required checks from repo rulesets that haven't reported yet |
| **Cmts** | Review thread resolution: `resolved/total` (e.g., `3/5`). `-` if no threads |
| **Branch**| Head branch name |
| **Updated**| Relative time: `12m`, `3h`, `2d`, `4mo` |

### Supported flags

| Flag | Description |
|------|-------------|
| `-R, --repo OWNER/REPO` | Target a specific repository |
| `-L, --limit N` | Maximum PRs to show (default: 30) |
| `-s, --state STATE` | Filter: `open`, `closed`, `merged`, `all` |
| `-A, --author USER` | Filter by PR author |
| `-a, --assignee USER` | Filter by assignee |
| `--app APP` | Filter by GitHub App |
| `-B, --base BRANCH` | Filter by base branch |
| `-H, --head BRANCH` | Filter by head branch |
| `-l, --label LABEL` | Filter by label (repeatable) |
| `-S, --search QUERY` | GitHub search syntax |
| `-d, --draft` | Show only draft PRs |
| `-w, --web` | Open in browser |
| `--json` | Output as JSON |

### Examples

```bash
gh prx list
gh prx list --author "@me" --state all
gh prx list --repo owner/repo --limit 10
gh prx list --label bug --label urgent
gh prx list --search "review:required status:success"
gh prx list --json
```

## What `gh prx atm` adds

An org-wide view of PRs that need your attention — no more checking each repo individually.

```
#    Title                                       Repo       Author         State  Review  AI    Appv  Checks  Cmts   Updated
#69  feat: add repo governance (CI lint, Cop...  ai-skills  fhemmerrelias  open   review  fail  0     fail    0/1    2d
#68  feat: add contract-testing skill for Pa...  ai-skills  fhemmerrelias  open   review  fail  0     pass    12/12  2d
```

By default, shows open PRs you authored across the org. Use `--review-required` to see PRs awaiting your review.

### `atm` flags

| Flag | Description |
|------|-------------|
| `-o, --org ORG` | Organization to search (default: inferred from current repo) |
| `-L, --limit N` | Maximum PRs to show (default: 30) |
| `-r, --review-required` | Show PRs where your review is requested |
| `--json` | Output as JSON |

### `atm` examples

```bash
gh prx atm                              # my PRs across current org
gh prx atm --org HemSoft                # my PRs in a specific org
gh prx atm --review-required            # PRs awaiting my review
gh prx atm --org Relias -r --limit 10   # review requests, capped
gh prx atm --json                       # machine-readable output
```

## Checking for updates

```bash
gh prx version
```

```
gh-prx v0.1.2 by HemSoft · gh extension install HemSoft/gh-prx
✓ Up to date
```

If a newer release exists:

```
gh-prx v0.1.0 by HemSoft · gh extension install HemSoft/gh-prx
↑ v0.1.2 available · gh extension upgrade gh-prx
```

## Local development

Requires Go 1.23+.

```bash
# Build and install locally (one-time symlink setup)
go build -o gh-prx.exe .   # Windows
go build -o gh-prx .        # macOS/Linux
gh extension install .

# After code changes, just rebuild — no reinstall needed
go build -o gh-prx.exe .
gh prx list
```

A convenience script is provided for Windows:

```powershell
.\build.ps1   # runs vet → test → build
```

## How it works

- Wraps `gh pr list --json` for core PR data and authentication
- Makes a single GraphQL call for supplemental data (review threads, AI reviewer detection, comment counts)
- Fetches required status check contexts from repo rulesets to detect pending-but-unreported CI checks
- Uses [termenv](https://github.com/muesli/termenv) for color output, respecting `NO_COLOR` and `CLICOLOR` conventions
- SSH host aliases (e.g., `github-work:org/repo`) are handled gracefully via `gh repo view` fallback

## Releases

Every push to `main` that includes code changes automatically creates a new patch release with prebuilt binaries for all platforms. Documentation-only changes are skipped.

For major or minor version bumps, tag manually:

```bash
git tag v1.0.0
git push origin v1.0.0
```

## License

MIT
