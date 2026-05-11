package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	gh "github.com/cli/go-gh/v2"
)

type changelogOptions struct {
	limit   int
	version string
}

func parseChangelogOptions(args []string, stderr io.Writer) (changelogOptions, error) {
	var options changelogOptions

	flags := flag.NewFlagSet("changelog", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		writeChangelogUsage(stderr)
	}

	flags.IntVar(&options.limit, "limit", 5, "Number of releases to show")
	flags.IntVar(&options.limit, "L", 5, "Number of releases to show")
	flags.StringVar(&options.version, "version", "", "Show a specific version (e.g. v0.3.0 or 0.3.0)")

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

	options.version = normalizeReleaseVersion(options.version)

	return options, nil
}

func normalizeReleaseVersion(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "v") {
		return s
	}
	return "v" + s
}

type releaseEntry struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

// fetchReleasesFunc is swapped in tests to avoid real API calls.
var fetchReleasesFunc = fetchReleases

func fetchReleases(limit int) ([]releaseEntry, error) {
	stdoutBuf, stderrBuf, err := gh.Exec(
		"api", fmt.Sprintf("repos/HemSoft/gh-prx/releases?per_page=%d", limit),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", stderrBuf.String(), err)
	}

	var releases []releaseEntry
	if err := json.Unmarshal(stdoutBuf.Bytes(), &releases); err != nil {
		return nil, fmt.Errorf("parsing releases: %w", err)
	}
	return releases, nil
}

var fetchReleaseByTagFunc = fetchReleaseByTag

func fetchReleaseByTag(tag string) (*releaseEntry, error) {
	stdoutBuf, stderrBuf, err := gh.Exec(
		"api", fmt.Sprintf("repos/HemSoft/gh-prx/releases/tags/%s", tag),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", stderrBuf.String(), err)
	}

	var release releaseEntry
	if err := json.Unmarshal(stdoutBuf.Bytes(), &release); err != nil {
		return nil, fmt.Errorf("parsing release: %w", err)
	}
	return &release, nil
}

func runChangelog(args []string, stdout io.Writer, stderr io.Writer) error {
	options, err := parseChangelogOptions(args, stderr)
	if err != nil {
		if errors.Is(err, errHelpDisplayed) {
			return nil
		}
		return err
	}

	return executeChangelog(options, stdout)
}

func executeChangelog(options changelogOptions, stdout io.Writer) error {
	if options.version != "" {
		release, err := fetchReleaseByTagFunc(options.version)
		if err != nil {
			return fmt.Errorf("release %s not found", options.version)
		}
		renderChangelog(stdout, []releaseEntry{*release})
		return nil
	}

	releases, err := fetchReleasesFunc(options.limit)
	if err != nil {
		return fmt.Errorf("fetching releases: %w", err)
	}

	if len(releases) == 0 {
		fmt.Fprintln(stdout, "No releases found.")
		return nil
	}

	renderChangelog(stdout, releases)
	return nil
}

func renderChangelog(stdout io.Writer, releases []releaseEntry) {
	for i, r := range releases {
		if i > 0 {
			fmt.Fprintln(stdout)
		}

		date := formatReleaseDate(r.PublishedAt)
		marker := ""
		if version != "dev" && r.TagName == version {
			marker = "  ← installed"
		}

		fmt.Fprintf(stdout, "\033[1m%s\033[0m  %s%s\n", r.TagName, date, marker)

		body := stripLeadingDate(strings.TrimSpace(r.Body))
		if body != "" {
			fmt.Fprintln(stdout, body)
		}
	}
}

func formatReleaseDate(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func stripLeadingDate(body string) string {
	lines := strings.SplitN(body, "\n", 3)
	if len(lines) == 0 {
		return body
	}
	// Check if first line looks like a date (YYYY-MM-DD)
	first := strings.TrimSpace(lines[0])
	if len(first) == 10 && first[4] == '-' && first[7] == '-' {
		rest := ""
		if len(lines) > 1 {
			rest = strings.Join(lines[1:], "\n")
		}
		return strings.TrimSpace(rest)
	}
	return body
}

func writeChangelogUsage(w io.Writer) {
	fmt.Fprint(w, changelogUsage)
}

const changelogUsage = `Usage:
  gh prx changelog [flags]

Show release notes for gh-prx versions.

Flags:
  -L, --limit int        Number of releases to show (default 5)
      --version string   Show a specific version (e.g. v0.3.0 or 0.3.0)
`
