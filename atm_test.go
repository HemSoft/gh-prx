package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseAtmOptionsDefaults(t *testing.T) {
	var stderr bytes.Buffer
	options, err := parseAtmOptions(nil, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.org != "" {
		t.Fatalf("expected empty org, got %q", options.org)
	}
	if options.limit != 30 {
		t.Fatalf("expected limit 30, got %d", options.limit)
	}
	if options.reviewRequired {
		t.Fatal("expected reviewRequired false")
	}
	if options.json {
		t.Fatal("expected json false")
	}
}

func TestParseAtmOptionsAllFlags(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"--org", "HemSoft", "--limit", "10", "--review-required", "--json"}
	options, err := parseAtmOptions(args, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.org != "HemSoft" {
		t.Fatalf("expected org HemSoft, got %q", options.org)
	}
	if options.limit != 10 {
		t.Fatalf("expected limit 10, got %d", options.limit)
	}
	if !options.reviewRequired {
		t.Fatal("expected reviewRequired true")
	}
	if !options.json {
		t.Fatal("expected json true")
	}
}

func TestParseAtmOptionsShortFlags(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"-o", "Relias", "-L", "5", "-r"}
	options, err := parseAtmOptions(args, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.org != "Relias" {
		t.Fatalf("expected org Relias, got %q", options.org)
	}
	if options.limit != 5 {
		t.Fatalf("expected limit 5, got %d", options.limit)
	}
	if !options.reviewRequired {
		t.Fatal("expected reviewRequired true")
	}
}

func TestParseAtmOptionsInvalidLimit(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"--limit", "0"}
	_, err := parseAtmOptions(args, &stderr)
	if err == nil {
		t.Fatal("expected error for zero limit")
	}
	if !strings.Contains(err.Error(), "limit must be greater than zero") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAtmOptionsUnexpectedArgs(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"extra"}
	_, err := parseAtmOptions(args, &stderr)
	if err == nil {
		t.Fatal("expected error for unexpected arguments")
	}
	if !strings.Contains(err.Error(), "unexpected arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAtmOptionsHelp(t *testing.T) {
	var stderr bytes.Buffer
	args := []string{"-h"}
	_, err := parseAtmOptions(args, &stderr)
	if err != errHelpDisplayed {
		t.Fatalf("expected errHelpDisplayed, got %v", err)
	}
}

func TestBuildAtmSearchQueryAuthor(t *testing.T) {
	got := buildAtmSearchQuery("HemSoft", "georufino", false)
	want := "is:pr is:open author:georufino org:HemSoft"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildAtmSearchQueryReviewRequired(t *testing.T) {
	got := buildAtmSearchQuery("Relias", "georufino", true)
	want := "is:pr is:open review-requested:georufino org:Relias"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildAtmGraphQLQuery(t *testing.T) {
	query := buildAtmGraphQLQuery("is:pr is:open author:georufino org:HemSoft", 10)
	if !strings.Contains(query, `"is:pr is:open author:georufino org:HemSoft"`) {
		t.Fatal("expected search query in GraphQL")
	}
	if !strings.Contains(query, "first: 10") {
		t.Fatal("expected limit in GraphQL")
	}
	if !strings.Contains(query, "search(") {
		t.Fatal("expected search clause")
	}
	if !strings.Contains(query, "statusCheckRollup") {
		t.Fatal("expected statusCheckRollup in query")
	}
}

func TestResolveAtmOrg(t *testing.T) {
	// With explicit org override
	got, err := resolveAtmOrg("MyOrg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "MyOrg" {
		t.Fatalf("expected MyOrg, got %q", got)
	}
}

func TestParseAtmSearchResponse(t *testing.T) {
	data := []byte(`{
		"data": {
			"search": {
				"nodes": [
					{
						"number": 42,
						"title": "Fix login",
						"author": {"login": "georufino"},
						"state": "OPEN",
						"isDraft": false,
						"reviewDecision": "APPROVED",
						"updatedAt": "2026-05-10T10:00:00Z",
						"headRefName": "fix/login",
						"baseRefName": "main",
						"url": "https://github.com/HemSoft/app/pull/42",
						"repository": {"nameWithOwner": "HemSoft/app"},
						"commits": {"nodes": [{"commit": {"statusCheckRollup": {"contexts": {"nodes": [
							{"__typename": "CheckRun", "name": "build", "status": "COMPLETED", "conclusion": "SUCCESS"}
						]}}}}]},
						"latestReviews": {"nodes": [
							{"state": "APPROVED", "author": {"login": "reviewer1"}, "comments": {"totalCount": 0}}
						]},
						"reviewThreads": {"totalCount": 3, "nodes": [
							{"isResolved": true}, {"isResolved": true}, {"isResolved": false}
						]}
					}
				]
			}
		}
	}`)

	nodes, err := parseAtmSearchResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Number != 42 {
		t.Fatalf("expected PR #42, got #%d", nodes[0].Number)
	}
	if nodes[0].Repository.NameWithOwner != "HemSoft/app" {
		t.Fatalf("unexpected repo: %s", nodes[0].Repository.NameWithOwner)
	}
}

func TestMapAtmNode(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	node := atmPullRequestNode{
		Number:         7,
		Title:          "Implement the new feature for user management across all services",
		State:          "OPEN",
		IsDraft:        true,
		ReviewDecision: "CHANGES_REQUESTED",
		UpdatedAt:      now.Add(-3 * time.Hour),
		HeadRefName:    "feature/users",
		BaseRefName:    "main",
		URL:            "https://github.com/Org/repo/pull/7",
		Author:         &author{Login: "georufino"},
	}
	node.Repository.NameWithOwner = "Org/repo"
	node.Commits.Nodes = []struct {
		Commit struct {
			StatusCheckRollup *struct {
				Contexts struct {
					Nodes []checkItem `json:"nodes"`
				} `json:"contexts"`
			} `json:"statusCheckRollup"`
		} `json:"commit"`
	}{
		{Commit: struct {
			StatusCheckRollup *struct {
				Contexts struct {
					Nodes []checkItem `json:"nodes"`
				} `json:"contexts"`
			} `json:"statusCheckRollup"`
		}{
			StatusCheckRollup: &struct {
				Contexts struct {
					Nodes []checkItem `json:"nodes"`
				} `json:"contexts"`
			}{
				Contexts: struct {
					Nodes []checkItem `json:"nodes"`
				}{
					Nodes: []checkItem{
						{Typename: "CheckRun", Name: "ci", Status: "COMPLETED", Conclusion: "SUCCESS"},
					},
				},
			},
		}},
	}
	node.LatestReviews.Nodes = []struct {
		State  string `json:"state"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		Comments struct {
			TotalCount int `json:"totalCount"`
		} `json:"comments"`
	}{
		{State: "APPROVED", Author: struct {
			Login string `json:"login"`
		}{Login: "reviewer1"}, Comments: struct {
			TotalCount int `json:"totalCount"`
		}{TotalCount: 0}},
	}
	node.ReviewThreads.TotalCount = 2
	node.ReviewThreads.Nodes = []struct {
		IsResolved bool `json:"isResolved"`
	}{
		{IsResolved: true}, {IsResolved: false},
	}

	dp := mapAtmNode(node, now)

	if dp.Number != 7 {
		t.Fatalf("expected number 7, got %d", dp.Number)
	}
	if dp.Repo != "repo" {
		t.Fatalf("expected repo 'repo', got %q", dp.Repo)
	}
	if dp.State != "draft" {
		t.Fatalf("expected state draft, got %q", dp.State)
	}
	if dp.Review != "changes" {
		t.Fatalf("expected review changes, got %q", dp.Review)
	}
	if dp.Checks != "pass" {
		t.Fatalf("expected checks pass, got %q", dp.Checks)
	}
	if dp.Approvals != 1 {
		t.Fatalf("expected 1 approval, got %d", dp.Approvals)
	}
	if dp.Comments != "1/2" {
		t.Fatalf("expected comments 1/2, got %q", dp.Comments)
	}
	if dp.Author != "georufino" {
		t.Fatalf("expected author georufino, got %q", dp.Author)
	}
	if dp.Updated != "3h" {
		t.Fatalf("expected updated 3h, got %q", dp.Updated)
	}
	if dp.URL != "https://github.com/Org/repo/pull/7" {
		t.Fatalf("unexpected URL: %s", dp.URL)
	}
	// Title should be trimmed to 42 chars
	if len(dp.Title) > 42 {
		t.Fatalf("expected title trimmed to 42, got %d chars: %q", len(dp.Title), dp.Title)
	}
}

func TestMapAtmNodeNoChecks(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	node := atmPullRequestNode{
		Number:    1,
		Title:     "Simple",
		State:     "OPEN",
		UpdatedAt: now,
	}
	node.Repository.NameWithOwner = "Org/simple-repo"

	dp := mapAtmNode(node, now)
	if dp.Checks != "-" {
		t.Fatalf("expected checks '-', got %q", dp.Checks)
	}
	if dp.Repo != "simple-repo" {
		t.Fatalf("expected repo 'simple-repo', got %q", dp.Repo)
	}
	if dp.Author != "-" {
		t.Fatalf("expected author '-', got %q", dp.Author)
	}
}

func TestRenderAtmTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := renderAtmTable(&buf, "HemSoft", "georufino", atmOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No open PRs authored by georufino in HemSoft") {
		t.Fatalf("unexpected empty message: %q", buf.String())
	}
}

func TestRenderAtmTableEmptyReviewRequired(t *testing.T) {
	var buf bytes.Buffer
	err := renderAtmTable(&buf, "Relias", "user", atmOptions{reviewRequired: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No open PRs requesting review from user in Relias") {
		t.Fatalf("unexpected empty message: %q", buf.String())
	}
}

func TestRenderAtmTableWithStyleNoColor(t *testing.T) {
	prs := []displayPullRequest{
		{Number: 10, Title: "Fix bug", Repo: "api", Author: "dev", State: "open", Review: "approved", AIReview: "pass", Approvals: 1, Checks: "pass", Comments: "2/2", Updated: "1h", URL: "https://github.com/Org/api/pull/10"},
	}
	var buf bytes.Buffer
	err := renderAtmTableWithStyle(&buf, prs, false)
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if strings.Contains(output, "\x1b[") {
		t.Fatal("expected no ANSI codes when color disabled")
	}
	if !strings.Contains(output, "#10") {
		t.Fatal("expected PR number")
	}
	if !strings.Contains(output, "api") {
		t.Fatal("expected repo column")
	}
	if !strings.Contains(output, "Repo") {
		t.Fatal("expected Repo header")
	}
}

func TestRenderAtmTableWithStyleColor(t *testing.T) {
	prs := []displayPullRequest{
		{Number: 5, Title: "Add feature", Repo: "web", Author: "user", State: "open", Review: "-", AIReview: "-", Approvals: 0, Checks: "pending", Comments: "-", Updated: "2d", URL: ""},
	}
	var buf bytes.Buffer
	err := renderAtmTableWithStyle(&buf, prs, true)
	if err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	if !strings.Contains(output, "\x1b[") {
		t.Fatal("expected ANSI codes when color enabled")
	}
}

func TestCountApprovalsFromNodes(t *testing.T) {
	nodes := []aiReviewNode{
		{State: "APPROVED", AuthorLogin: "a"},
		{State: "COMMENTED", AuthorLogin: "b"},
		{State: "APPROVED", AuthorLogin: "c"},
		{State: "CHANGES_REQUESTED", AuthorLogin: "d"},
	}
	if got := countApprovalsFromNodes(nodes); got != 2 {
		t.Fatalf("expected 2, got %d", got)
	}
}

func TestCountApprovalsFromNodesEmpty(t *testing.T) {
	if got := countApprovalsFromNodes(nil); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestRunAtmSubcommandRouting(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// "atm --help" should display help without error
	err := run([]string{"atm", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected no error for atm --help, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "gh prx atm") {
		t.Fatalf("expected atm usage in stderr, got: %q", stderr.String())
	}
}

func TestParseAtmSearchResponseGraphQLError(t *testing.T) {
	data := []byte(`{
		"data": {"search": {"nodes": []}},
		"errors": [{"type": "INSUFFICIENT_SCOPES", "message": "Your token has not been granted the required scopes"}]
	}`)
	_, err := parseAtmSearchResponse(data)
	if err == nil {
		t.Fatal("expected error for GraphQL error response")
	}
	if !strings.Contains(err.Error(), "Your token has not been granted") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootUsageMentionsAtm(t *testing.T) {
	if !strings.Contains(rootUsage, "atm") {
		t.Fatal("root usage should mention atm subcommand")
	}
}
