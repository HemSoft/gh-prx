# gh-prx

`gh-prx` is a Go-based GitHub CLI extension that adds a richer pull request list view with:

- cleaner table output
- derived review and checks columns
- branch context in a single column
- optional JSON output for scripting

## Status

This repository contains the source for the extension command:

```text
gh prx list
```

## What `gh prx list` adds

Compared to `gh pr list`, this command keeps GitHub CLI's existing filters but renders a denser table:

- `State` shows `open`, `draft`, `closed`, or `merged`
- `Review` normalizes review state to `approved`, `changes`, or `review`
- `Checks` condenses status checks to `pass`, `fail`, `pending`, or `-`
- `Branch` shows `head -> base`
- `Updated` renders a short relative time like `12m`, `3h`, or `2d`

## Prerequisites

- GitHub CLI authenticated with the desired account
- Go 1.22 or newer

## Local development

From the repository root:

```powershell
go mod tidy
go build -o gh-prx.exe .
gh extension install .
gh prx list
```

On macOS or Linux:

```bash
go mod tidy
go build -o gh-prx .
gh extension install .
gh prx list
```

## Usage

```text
gh prx list [flags]
```

### Supported flags

- `-R, --repo`
- `-L, --limit`
- `-s, --state`
- `-A, --author`
- `-a, --assignee`
- `--app`
- `-B, --base`
- `-H, --head`
- `-l, --label` (repeatable)
- `-S, --search`
- `-d, --draft`
- `-w, --web`
- `--json`

## Examples

```powershell
gh prx list
gh prx list --author "@me" --state all
gh prx list --label bug --label urgent
gh prx list --search "review:required status:success"
gh prx list --json
```

## Notes

- This project intentionally wraps `gh pr list` so it can reuse GitHub CLI authentication and repository context.
- For public installation from GitHub releases, add precompiled release assets in a later step.
