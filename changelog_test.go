package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseChangelogOptionsDefaults(t *testing.T) {
	var stderr bytes.Buffer
	options, err := parseChangelogOptions(nil, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.limit != 5 {
		t.Fatalf("expected limit 5, got %d", options.limit)
	}
	if options.version != "" {
		t.Fatalf("expected empty version, got %q", options.version)
	}
}

func TestParseChangelogOptionsAllFlags(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"--limit", "10", "--version", "0.3.0"}
	options, err := parseChangelogOptions(args, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.limit != 10 {
		t.Fatalf("expected limit 10, got %d", options.limit)
	}
	if options.version != "v0.3.0" {
		t.Fatalf("expected v0.3.0, got %q", options.version)
	}
}

func TestParseChangelogOptionsShortFlags(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"-L", "3"}
	options, err := parseChangelogOptions(args, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.limit != 3 {
		t.Fatalf("expected limit 3, got %d", options.limit)
	}
}

func TestParseChangelogOptionsInvalidLimit(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseChangelogOptions([]string{"--limit", "0"}, &stderr)
	if err == nil {
		t.Fatal("expected error for zero limit")
	}
}

func TestParseChangelogOptionsHelp(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseChangelogOptions([]string{"-h"}, &stderr)
	if err != errHelpDisplayed {
		t.Fatalf("expected errHelpDisplayed, got %v", err)
	}
}

func TestParseChangelogOptionsUnexpectedArgs(t *testing.T) {
	var stderr bytes.Buffer
	_, err := parseChangelogOptions([]string{"extra"}, &stderr)
	if err == nil {
		t.Fatal("expected error for unexpected arguments")
	}
}

func TestNormalizeReleaseVersion(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", ""},
		{"v0.3.0", "v0.3.0"},
		{"0.3.0", "v0.3.0"},
		{" 0.3.0 ", "v0.3.0"},
		{"v1.0.0-beta", "v1.0.0-beta"},
	}
	for _, tc := range tests {
		if got := normalizeReleaseVersion(tc.input); got != tc.want {
			t.Errorf("normalizeReleaseVersion(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestStripLeadingDate(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{
			"with date",
			"2026-05-11\n\n- feat: add me command\n",
			"- feat: add me command",
		},
		{
			"no date",
			"- feat: add me command\n",
			"- feat: add me command\n",
		},
		{
			"empty",
			"",
			"",
		},
		{
			"date only",
			"2026-05-11",
			"",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripLeadingDate(tc.input); got != tc.want {
				t.Errorf("stripLeadingDate(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatReleaseDate(t *testing.T) {
	got := formatReleaseDate("2026-05-11T19:43:04Z")
	if got != "2026-05-11" {
		t.Fatalf("expected 2026-05-11, got %q", got)
	}
	if formatReleaseDate("invalid") != "" {
		t.Fatal("expected empty for invalid date")
	}
}

func TestRenderChangelog(t *testing.T) {
	releases := []releaseEntry{
		{
			TagName:     "v0.3.2",
			PublishedAt: "2026-05-11T19:43:04Z",
			Body:        "2026-05-11\n\n- feat: add me command\n",
		},
		{
			TagName:     "v0.3.1",
			PublishedAt: "2026-05-11T19:32:19Z",
			Body:        "2026-05-11\n\n- ci: auto-release generates notes\n",
		},
	}

	var buf bytes.Buffer
	renderChangelog(&buf, releases)
	output := buf.String()

	if !strings.Contains(output, "v0.3.2") {
		t.Fatal("expected v0.3.2 in output")
	}
	if !strings.Contains(output, "v0.3.1") {
		t.Fatal("expected v0.3.1 in output")
	}
	if !strings.Contains(output, "- feat: add me command") {
		t.Fatal("expected release body in output")
	}
	// The leading date should be stripped (not duplicated)
	if strings.Count(output, "2026-05-11") != 2 {
		t.Fatalf("expected exactly 2 date occurrences (one per release header), got:\n%s", output)
	}
}

func TestRenderChangelogCurrentVersion(t *testing.T) {
	saved := version
	version = "v0.3.1"
	defer func() { version = saved }()

	releases := []releaseEntry{
		{TagName: "v0.3.2", PublishedAt: "2026-05-11T19:43:04Z", Body: "- feat: add me command"},
		{TagName: "v0.3.1", PublishedAt: "2026-05-11T19:32:19Z", Body: "- ci: notes"},
	}

	var buf bytes.Buffer
	renderChangelog(&buf, releases)
	output := buf.String()

	if !strings.Contains(output, "← installed") {
		t.Fatal("expected '← installed' marker for current version")
	}
	// Should only appear once (for v0.3.1, not v0.3.2)
	if strings.Count(output, "← installed") != 1 {
		t.Fatal("marker should appear exactly once")
	}
}

func TestRenderChangelogDevVersion(t *testing.T) {
	saved := version
	version = "dev"
	defer func() { version = saved }()

	releases := []releaseEntry{
		{TagName: "v0.3.2", PublishedAt: "2026-05-11T19:43:04Z", Body: "- feat: me"},
	}

	var buf bytes.Buffer
	renderChangelog(&buf, releases)
	if strings.Contains(buf.String(), "← installed") {
		t.Fatal("dev builds should not show installed marker")
	}
}

func TestExecuteChangelogEmpty(t *testing.T) {
	saved := fetchReleasesFunc
	fetchReleasesFunc = func(limit int) ([]releaseEntry, error) {
		return nil, nil
	}
	defer func() { fetchReleasesFunc = saved }()

	var buf bytes.Buffer
	err := executeChangelog(changelogOptions{limit: 5}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No releases found") {
		t.Fatalf("expected empty message, got %q", buf.String())
	}
}

func TestExecuteChangelogByVersion(t *testing.T) {
	saved := fetchReleaseByTagFunc
	fetchReleaseByTagFunc = func(tag string) (*releaseEntry, error) {
		if tag != "v0.3.0" {
			t.Fatalf("expected v0.3.0, got %q", tag)
		}
		return &releaseEntry{
			TagName:     "v0.3.0",
			PublishedAt: "2026-05-11T19:09:56Z",
			Body:        "- scrub real data",
		}, nil
	}
	defer func() { fetchReleaseByTagFunc = saved }()

	var buf bytes.Buffer
	err := executeChangelog(changelogOptions{limit: 5, version: "v0.3.0"}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "v0.3.0") {
		t.Fatal("expected v0.3.0 in output")
	}
}

func TestExecuteChangelogList(t *testing.T) {
	saved := fetchReleasesFunc
	fetchReleasesFunc = func(limit int) ([]releaseEntry, error) {
		if limit != 3 {
			t.Fatalf("expected limit 3, got %d", limit)
		}
		return []releaseEntry{
			{TagName: "v0.3.2", PublishedAt: "2026-05-11T19:43:04Z", Body: "- feat: me"},
			{TagName: "v0.3.1", PublishedAt: "2026-05-11T19:32:19Z", Body: "- ci: notes"},
		}, nil
	}
	defer func() { fetchReleasesFunc = saved }()

	var buf bytes.Buffer
	err := executeChangelog(changelogOptions{limit: 3}, &buf)
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "v0.3.2") || !strings.Contains(output, "v0.3.1") {
		t.Fatalf("expected both versions, got:\n%s", output)
	}
}

func TestChangelogSubcommandRouting(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"changelog", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected no error for changelog --help, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "gh prx changelog") {
		t.Fatalf("expected changelog usage in stderr, got: %q", stderr.String())
	}
}

func TestRootUsageMentionsChangelog(t *testing.T) {
	if !strings.Contains(rootUsage, "changelog") {
		t.Fatal("root usage should mention changelog subcommand")
	}
}
