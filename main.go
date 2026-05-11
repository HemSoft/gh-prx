package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	gh "github.com/cli/go-gh/v2"
)

// version is injected at build time via -ldflags "-X main.version=vX.Y.Z"
var version = "dev"

var errHelpDisplayed = errors.New("help displayed")

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		writeRootUsage(stdout)
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		writeRootUsage(stdout)
		return nil
	case "version", "-v", "--version":
		return runVersion(stdout)
	case "list":
		return runList(args[1:], stdout, stderr)
	case "atm":
		return runAtm(args[1:], stdout, stderr)
	default:
		writeRootUsage(stderr)
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func runList(args []string, stdout io.Writer, stderr io.Writer) error {
	options, err := parseListOptions(args, stderr)
	if err != nil {
		if errors.Is(err, errHelpDisplayed) {
			return nil
		}

		return err
	}

	return executeList(options, stdout)
}

func parseListOptions(args []string, stderr io.Writer) (listOptions, error) {
	options := defaultListOptions()

	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		writeListUsage(stderr)
	}

	flags.Var(&options.labels, "label", "Filter by label (repeatable)")
	flags.Var(&options.labels, "l", "Filter by label (repeatable)")

	flags.StringVar(&options.repo, "repo", "", "Select another repository using the [HOST/]OWNER/REPO format")
	flags.StringVar(&options.repo, "R", "", "Select another repository using the [HOST/]OWNER/REPO format")
	flags.IntVar(&options.limit, "limit", 30, "Maximum number of pull requests to fetch")
	flags.IntVar(&options.limit, "L", 30, "Maximum number of pull requests to fetch")
	flags.StringVar(&options.state, "state", "open", "Filter by state: open, closed, merged, or all")
	flags.StringVar(&options.state, "s", "open", "Filter by state: open, closed, merged, or all")
	flags.StringVar(&options.author, "author", "", "Filter by author")
	flags.StringVar(&options.author, "A", "", "Filter by author")
	flags.StringVar(&options.assignee, "assignee", "", "Filter by assignee")
	flags.StringVar(&options.assignee, "a", "", "Filter by assignee")
	flags.StringVar(&options.app, "app", "", "Filter by GitHub App author")
	flags.StringVar(&options.base, "base", "", "Filter by base branch")
	flags.StringVar(&options.base, "B", "", "Filter by base branch")
	flags.StringVar(&options.head, "head", "", "Filter by head branch")
	flags.StringVar(&options.head, "H", "", "Filter by head branch")
	flags.StringVar(&options.search, "search", "", "Search pull requests with a GitHub search query")
	flags.StringVar(&options.search, "S", "", "Search pull requests with a GitHub search query")
	flags.BoolVar(&options.draftOnly, "draft", false, "Filter by draft state")
	flags.BoolVar(&options.draftOnly, "d", false, "Filter by draft state")
	flags.BoolVar(&options.web, "web", false, "Open the matching pull requests in the browser")
	flags.BoolVar(&options.web, "w", false, "Open the matching pull requests in the browser")
	flags.BoolVar(&options.json, "json", false, "Output enriched JSON instead of a table")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return options, errHelpDisplayed
		}

		return options, err
	}

	if flags.NArg() > 0 {
		return options, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), ", "))
	}

	if options.limit < 1 {
		return options, errors.New("limit must be greater than zero")
	}

	if options.web && options.json {
		return options, errors.New("--web and --json cannot be used together")
	}

	return options, nil
}

func runVersion(w io.Writer) error {
	return runVersionTestable(w, version)
}

func fetchLatestRelease(owner, repo string) (string, error) {
	stdoutBuf, stderrBuf, err := gh.Exec(
		"api", fmt.Sprintf("repos/%s/%s/releases/latest", owner, repo),
		"--jq", ".tag_name",
	)
	if err != nil {
		return "", fmt.Errorf("%s: %w", stderrBuf.String(), err)
	}
	return strings.TrimSpace(stdoutBuf.String()), nil
}

// fetchLatestReleaseFunc is swapped in tests to avoid real API calls.
var fetchLatestReleaseFunc = fetchLatestRelease

func runVersionTestable(w io.Writer, ver string) error {
	const (
		author     = "HemSoft"
		repo       = "gh-prx"
		installCmd = "gh extension install HemSoft/gh-prx"
		upgradeCmd = "gh extension upgrade gh-prx"
	)

	fmt.Fprintf(w, "%s %s by %s · %s\n", repo, ver, author, installCmd)

	latest, err := fetchLatestReleaseFunc(author, repo)
	if err != nil || latest == "" {
		fmt.Fprintf(w, "⚠ Could not check for updates\n")
		return nil
	}

	if ver == "dev" {
		fmt.Fprintf(w, "⚙ Dev build · latest release: %s\n", latest)
	} else if latest != ver {
		fmt.Fprintf(w, "↑ %s available · %s\n", latest, upgradeCmd)
	} else {
		fmt.Fprintf(w, "✓ Up to date\n")
	}

	return nil
}

func writeRootUsage(w io.Writer) {
	fmt.Fprint(w, rootUsage)
}

func writeListUsage(w io.Writer) {
	fmt.Fprint(w, listUsage)
}

const rootUsage = `gh-prx adds opinionated pull request commands for GitHub CLI.

Usage:
  gh prx <command> [flags]

Available Commands:
  list      Render a denser pull request list than gh pr list
  atm       Show open PRs across an org that need your attention
  version   Show version, author, and update availability

Examples:
  gh prx list
  gh prx list --author "@me" --state all
  gh prx list --json
  gh prx atm
  gh prx atm --org HemSoft
  gh prx atm --review-required
  gh prx version
`

const listUsage = `Usage:
  gh prx list [flags]

Flags:
  -R, --repo string       Select another repository using the [HOST/]OWNER/REPO format
  -L, --limit int         Maximum number of pull requests to fetch (default 30)
  -s, --state string      Filter by state: open, closed, merged, or all (default "open")
  -A, --author string     Filter by author
  -a, --assignee string   Filter by assignee
      --app string        Filter by GitHub App author
  -B, --base string       Filter by base branch
  -H, --head string       Filter by head branch
  -l, --label string      Filter by label (repeatable)
  -S, --search string     Search pull requests with a GitHub search query
  -d, --draft             Filter by draft state
  -w, --web               Open the matching pull requests in the browser
      --json              Output enriched JSON instead of a table
`
