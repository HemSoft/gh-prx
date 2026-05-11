package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseMeOptionsDefaults(t *testing.T) {
	var stderr bytes.Buffer
	options, err := parseMeOptions(nil, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.org != "" {
		t.Fatalf("expected empty org, got %q", options.org)
	}
	if options.limit != 30 {
		t.Fatalf("expected limit 30, got %d", options.limit)
	}
	if options.json {
		t.Fatal("expected json false")
	}
}

func TestParseMeOptionsAllFlags(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"--org", "AcmeCorp", "--limit", "10", "--json"}
	options, err := parseMeOptions(args, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.org != "AcmeCorp" {
		t.Fatalf("expected org AcmeCorp, got %q", options.org)
	}
	if options.limit != 10 {
		t.Fatalf("expected limit 10, got %d", options.limit)
	}
	if !options.json {
		t.Fatal("expected json true")
	}
}

func TestParseMeOptionsShortFlags(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"-o", "AcmeCorp", "-L", "5"}
	options, err := parseMeOptions(args, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.org != "AcmeCorp" {
		t.Fatalf("expected org AcmeCorp, got %q", options.org)
	}
	if options.limit != 5 {
		t.Fatalf("expected limit 5, got %d", options.limit)
	}
}

func TestParseMeOptionsInvalidLimit(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"--limit", "0"}
	_, err := parseMeOptions(args, &stderr)
	if err == nil {
		t.Fatal("expected error for zero limit")
	}
	if !strings.Contains(err.Error(), "limit must be greater than zero") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMeOptionsUnexpectedArgs(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"extra"}
	_, err := parseMeOptions(args, &stderr)
	if err == nil {
		t.Fatal("expected error for unexpected arguments")
	}
}

func TestParseMeOptionsHelp(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"-h"}
	_, err := parseMeOptions(args, &stderr)
	if err != errHelpDisplayed {
		t.Fatalf("expected errHelpDisplayed, got %v", err)
	}
}

func TestBuildMeQueries(t *testing.T) {
	queries := buildMeQueriesWithQualifier("org:AcmeCorp", "octocat")
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(queries))
	}
	if !strings.Contains(queries[0], "author:octocat") {
		t.Fatalf("query 0 should contain author, got %q", queries[0])
	}
	if !strings.Contains(queries[0], "org:AcmeCorp") {
		t.Fatalf("query 0 should contain org, got %q", queries[0])
	}
	if !strings.Contains(queries[1], "assignee:octocat") {
		t.Fatalf("query 1 should contain assignee, got %q", queries[1])
	}
	if !strings.Contains(queries[1], "-author:octocat") {
		t.Fatalf("query 1 should exclude author, got %q", queries[1])
	}
}

func TestBuildMeQueriesForUser(t *testing.T) {
	queries := buildMeQueriesWithQualifier("user:octocat", "octocat")
	if !strings.Contains(queries[0], "user:octocat") {
		t.Fatalf("query 0 should contain user qualifier, got %q", queries[0])
	}
}

func TestRenderMeTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := renderMeTable(&buf, "AcmeCorp", "octocat", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No open PRs authored by or assigned to octocat in AcmeCorp") {
		t.Fatalf("unexpected empty message: %q", buf.String())
	}
}

func TestRenderMeTableWithPRs(t *testing.T) {
	prs := []displayPullRequest{
		{Number: 10, Title: "Fix bug", Repo: "api", Author: "octocat", State: "open", Review: "approved", AIReview: "pass", Approvals: 1, Checks: "pass", Comments: "2/2", Updated: "1h"},
	}
	var buf bytes.Buffer
	err := renderMeTable(&buf, "AcmeCorp", "octocat", prs)
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "Open PRs for octocat in AcmeCorp") {
		t.Fatalf("expected header, got %q", output)
	}
	if !strings.Contains(output, "#10") {
		t.Fatal("expected PR number in output")
	}
}

func TestRunMeSubcommandRouting(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"me", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected no error for me --help, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "gh prx me") {
		t.Fatalf("expected me usage in stderr, got: %q", stderr.String())
	}
}

func TestRootUsageMentionsMe(t *testing.T) {
	if !strings.Contains(rootUsage, "me") {
		t.Fatal("root usage should mention me subcommand")
	}
}
